package infrai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestSendRetriesRateLimitWithSameKey(t *testing.T) {
	calls := 0
	client := &SMSClient{
		APIKey:     "test-key",
		BaseURL:    "https://example.test",
		MaxRetries: 1,
		Sleep:      func(context.Context, time.Duration) error { return nil },
		HTTP: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != http.MethodPost || req.Header.Get("Idempotency-Key") != "course-deadline:c1:l1" {
				t.Fatalf("unexpected request method or idempotency key")
			}
			status := http.StatusOK
			body := `{"ok":true,"data":{"message_id":"msg-42"},"metadata":{}}`
			if calls == 1 {
				status = http.StatusTooManyRequests
				body = `{"ok":false,"data":{},"error":{"code":"RATE_LIMITED","message":"retry later"},"metadata":{}}`
			}
			return &http.Response{StatusCode: status, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
		})},
	}

	result, err := client.Send(context.Background(), "+15551234567", "Course due soon", "course-deadline:c1:l1")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || result.MessageID != "msg-42" {
		t.Fatalf("calls=%d message_id=%q", calls, result.MessageID)
	}
}
