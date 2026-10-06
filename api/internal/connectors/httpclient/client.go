// Package httpclient is the shared vendor HTTP client.
// It retries 429 and 5xx, and it waits for Retry-After before trying again.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const defaultMaxAttempts = 3

type Client struct {
	HTTP        *http.Client
	MaxAttempts int
	Sleep       func(context.Context, time.Duration) error
}

func (c Client) Do(ctx context.Context, request *http.Request) (*http.Response, error) {
	attempts := c.MaxAttempts
	if attempts <= 0 {
		attempts = defaultMaxAttempts
	}
	sleep := c.Sleep
	if sleep == nil {
		sleep = sleepCtx
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	var lastStatus int
	for attempt := 1; attempt <= attempts; attempt++ {
		cloned, err := clone(ctx, request)
		if err != nil {
			return nil, err
		}
		response, err := httpClient.Do(cloned)
		if err != nil {
			return nil, err
		}
		if response.StatusCode != http.StatusTooManyRequests && response.StatusCode < 500 {
			return response, nil
		}
		lastStatus = response.StatusCode
		wait := retryAfter(response.Header.Get("Retry-After"))
		if err := response.Body.Close(); err != nil {
			return nil, err
		}
		if attempt == attempts {
			break
		}
		if err := sleep(ctx, wait+jitter(wait)); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("vendor returned %d", lastStatus)
}

func clone(ctx context.Context, request *http.Request) (*http.Request, error) {
	cloned := request.Clone(ctx)
	if request.Body == nil || request.GetBody == nil {
		return cloned, nil
	}
	body, err := request.GetBody()
	if err != nil {
		return nil, err
	}
	cloned.Body = body
	return cloned, nil
}

func retryAfter(raw string) time.Duration {
	if raw == "" {
		return time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return time.Second
	}
	return time.Duration(seconds) * time.Second
}

func jitter(wait time.Duration) time.Duration {
	if wait <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(wait)/2 + 1))
}

func sleepCtx(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Discard reads and closes a response body so the connection can be reused.
func Discard(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
