// Package idempotency stores the response of a mutating request so a retry
// with the same key does not run the handler twice.
package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/problem"
)

const headerKey = "Idempotency-Key"

type Record struct {
	RequestHash []byte
	Status      int
	Body        []byte
	ContentType string
}

type Keeper interface {
	Claim(ctx context.Context, workspaceID, key string, requestHash []byte) (Record, bool, error)
	Complete(ctx context.Context, workspaceID, key string, status int, body []byte, contentType string) error
	Release(ctx context.Context, workspaceID, key string) error
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Claim(ctx context.Context, workspaceID, key string, requestHash []byte) (Record, bool, error) {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (workspace_id, idempotency_key, request_hash, status_code, response_body, content_type)
		VALUES ($1, $2, $3, 0, '', '')
		ON CONFLICT DO NOTHING`, workspaceID, key, requestHash)
	if err != nil {
		return Record{}, false, err
	}
	if tag.RowsAffected() == 1 {
		return Record{RequestHash: append([]byte(nil), requestHash...)}, true, nil
	}

	record, err := s.load(ctx, workspaceID, key)
	return record, false, err
}

func (s *Store) Complete(ctx context.Context, workspaceID, key string, status int, body []byte, contentType string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE idempotency_keys
		SET status_code = $3, response_body = $4, content_type = $5
		WHERE workspace_id = $1 AND idempotency_key = $2 AND status_code = 0`,
		workspaceID, key, status, body, contentType)
	return err
}

func (s *Store) Release(ctx context.Context, workspaceID, key string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM idempotency_keys WHERE workspace_id = $1 AND idempotency_key = $2 AND status_code = 0`, workspaceID, key)
	return err
}

func (s *Store) load(ctx context.Context, workspaceID, key string) (Record, error) {
	var record Record
	err := s.pool.QueryRow(ctx, `
		SELECT request_hash, status_code, response_body, content_type
		FROM idempotency_keys
		WHERE workspace_id = $1 AND idempotency_key = $2`, workspaceID, key).Scan(&record.RequestHash, &record.Status, &record.Body, &record.ContentType)
	return record, err
}

func Middleware(keeper Keeper) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if !mutating(request.Method) {
				next.ServeHTTP(writer, request)
				return
			}

			key := strings.TrimSpace(request.Header.Get(headerKey))
			workspaceID := auth.WorkspaceID(request.Context())
			if key == "" || workspaceID == "" {
				next.ServeHTTP(writer, request)
				return
			}
			if len(key) > 128 {
				problem.Write(writer, http.StatusBadRequest, "", "idempotency key is too long")
				return
			}

			body, err := io.ReadAll(request.Body)
			if err != nil {
				problem.Write(writer, http.StatusBadRequest, "", "could not read request body")
				return
			}
			request.Body = io.NopCloser(bytes.NewReader(body))

			hash := requestHash(request.Method, request.URL.Path, request.URL.RawQuery, body)
			record, claimed, err := keeper.Claim(request.Context(), workspaceID, key, hash)
			if err != nil {
				problem.Write(writer, http.StatusInternalServerError, "", "could not store idempotency key")
				return
			}
			if !claimed {
				writeReplay(writer, record, hash)
				return
			}

			finished := false
			defer func() {
				if !finished {
					_ = keeper.Release(request.Context(), workspaceID, key)
				}
			}()

			captured := &capture{header: make(http.Header)}
			next.ServeHTTP(captured, request)
			status := captured.status
			if status == 0 {
				status = http.StatusOK
			}
			contentType := captured.header.Get("Content-Type")
			if err := keeper.Complete(request.Context(), workspaceID, key, status, captured.body.Bytes(), contentType); err != nil {
				problem.Write(writer, http.StatusInternalServerError, "", "could not store idempotency result")
				return
			}
			finished = true
			copyHeader(writer.Header(), captured.header)
			writer.WriteHeader(status)
			_, _ = writer.Write(captured.body.Bytes())
		})
	}
}

func writeReplay(writer http.ResponseWriter, record Record, hash []byte) {
	if record.Status == 0 {
		problem.Write(writer, http.StatusConflict, "", "a request with this idempotency key is still running")
		return
	}
	if !bytes.Equal(record.RequestHash, hash) {
		problem.Write(writer, http.StatusConflict, "", "this idempotency key was used for a different request")
		return
	}
	if record.ContentType != "" {
		writer.Header().Set("Content-Type", record.ContentType)
	}
	writer.WriteHeader(record.Status)
	_, _ = writer.Write(record.Body)
}

func requestHash(method, path, rawQuery string, body []byte) []byte {
	sum := sha256.New()
	_, _ = io.WriteString(sum, method)
	_, _ = sum.Write([]byte{0})
	_, _ = io.WriteString(sum, path)
	_, _ = sum.Write([]byte{0})
	_, _ = io.WriteString(sum, rawQuery)
	_, _ = sum.Write([]byte{0})
	_, _ = sum.Write(body)
	return sum.Sum(nil)
}

func mutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

type capture struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (c *capture) Header() http.Header {
	return c.header
}

func (c *capture) WriteHeader(status int) {
	if c.status != 0 {
		return
	}
	c.status = status
}

func (c *capture) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	return c.body.Write(b)
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
