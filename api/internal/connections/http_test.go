package connections

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/o-mid/contract-ops/api/internal/credentials"
	"github.com/o-mid/contract-ops/api/internal/httpapi"
	"github.com/o-mid/contract-ops/api/internal/platform/auth"
	"github.com/o-mid/contract-ops/api/internal/platform/db"
	"github.com/o-mid/contract-ops/api/internal/platform/migrate"
)

const (
	testSecret    = "super-secret-value"
	testAPIKey    = "co_local_dev_key_not_for_production"
	badSecret     = "bad-credential-value"
	rotatedSecret = "rotated-secret-value"
)

func TestRotateKeepsTheOldSecretWhenVerifyFails(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if err := auth.NewStore(pool).EnsureBootstrap(ctx, testAPIKey); err != nil {
		t.Fatal(err)
	}

	key, err := credentials.NewEnvKey(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealer := credentials.NewSealer(key)
	store := NewStore(pool)
	handler := httpapi.NewServer(nil, httpapi.Options{
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Authenticate: auth.Middleware(auth.NewStore(pool).Resolve),
		Register:     NewHandler(NewService(store, sealer, rejectSecret{secret: badSecret})).Routes,
	}).Handler()

	created := call(t, handler, http.MethodPost, "/v1/connections", `{"kind":"fakevendor","name":"Demo","secret":"`+testSecret+`"}`, http.StatusCreated)
	if strings.Contains(created, testSecret) {
		t.Fatal("create response included the secret")
	}
	var conn Connection
	if err := json.Unmarshal([]byte(created), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.Status != StatusNeedsAuth || conn.Credential == nil || conn.Credential.Fingerprint == "" {
		t.Fatalf("created = %+v", conn)
	}
	original := conn.Credential.Fingerprint

	verified := call(t, handler, http.MethodPost, "/v1/connections/"+conn.ID+"/verify", "", http.StatusOK)
	if err := json.Unmarshal([]byte(verified), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.Status != StatusHealthy {
		t.Fatalf("status after verify = %s", conn.Status)
	}

	rejected := call(t, handler, http.MethodPost, "/v1/connections/"+conn.ID+"/rotate", `{"secret":"`+badSecret+`"}`, http.StatusUnauthorized)
	if strings.Contains(rejected, badSecret) {
		t.Fatal("rejected rotate response included the secret")
	}
	opened, err := openActive(ctx, store, sealer, conn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if opened != testSecret {
		t.Fatalf("secret after rejected rotate = %q", opened)
	}

	rotated := call(t, handler, http.MethodPost, "/v1/connections/"+conn.ID+"/rotate", `{"secret":"`+rotatedSecret+`"}`, http.StatusOK)
	if strings.Contains(rotated, rotatedSecret) || strings.Contains(rotated, testSecret) {
		t.Fatal("rotate response included a secret")
	}
	if err := json.Unmarshal([]byte(rotated), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.Credential == nil || conn.Credential.Fingerprint == original {
		t.Fatal("fingerprint did not change")
	}
	opened, err = openActive(ctx, store, sealer, conn.ID)
	if err != nil {
		t.Fatal(err)
	}
	if opened != rotatedSecret {
		t.Fatalf("opened after rotate = %q", opened)
	}

	paused := call(t, handler, http.MethodPatch, "/v1/connections/"+conn.ID, `{"paused":true}`, http.StatusOK)
	if err := json.Unmarshal([]byte(paused), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.Status != StatusPaused {
		t.Fatalf("paused status = %s", conn.Status)
	}
	resumed := call(t, handler, http.MethodPatch, "/v1/connections/"+conn.ID, `{"paused":false}`, http.StatusOK)
	if err := json.Unmarshal([]byte(resumed), &conn); err != nil {
		t.Fatal(err)
	}
	if conn.Status != StatusHealthy {
		t.Fatalf("resumed status = %s", conn.Status)
	}

	var events int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM activity_events WHERE connection_id = $1`, conn.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events < 4 {
		t.Fatalf("activity events = %d", events)
	}
}

type rejectSecret struct {
	secret string
}

func (r rejectSecret) Verify(ctx context.Context, kind, secret string) error {
	if secret == r.secret {
		return &Failure{Status: http.StatusUnauthorized, Code: "auth_invalid", Detail: "credential was rejected"}
	}
	return StaticVerifier{}.Verify(ctx, kind, secret)
}

func call(t *testing.T, handler http.Handler, method, path, body string, want int) string {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testAPIKey)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != want {
		t.Fatalf("%s %s = %d, body %s", method, path, response.Code, response.Body.String())
	}
	return response.Body.String()
}

func openActive(ctx context.Context, store *Store, sealer *credentials.Sealer, id string) (string, error) {
	_, sealed, err := store.ActiveSecret(ctx, auth.LocalWorkspaceID, id)
	if err != nil {
		return "", err
	}
	plain, err := sealer.Open(ctx, sealed)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	admin, err := sql.Open("pgx", base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.ExecContext(context.Background(), `DROP DATABASE IF EXISTS contract_ops_conn_test WITH (FORCE)`)
		_ = admin.Close()
	})
	if err := admin.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	_, _ = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS contract_ops_conn_test WITH (FORCE)`)
	if _, err := admin.ExecContext(ctx, `CREATE DATABASE contract_ops_conn_test`); err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(base)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = "/contract_ops_conn_test"
	sqlDB, err := sql.Open("pgx", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	pool, err := db.Open(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
