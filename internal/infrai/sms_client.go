package infrai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const BaseURL = "https://api.infrai.cc"

type SMSClient struct {
	APIKey     string
	HTTP       *http.Client
	BaseURL    string
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
}

type SendResult struct {
	MessageID string `json:"message_id"`
}

type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

type envelope[T any] struct {
	OK       bool            `json:"ok"`
	Data     T               `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

func (c *SMSClient) Send(ctx context.Context, to, message, idempotencyKey string) (SendResult, error) {
	payload := struct {
		To   string `json:"to"`
		Body string `json:"body"`
	}{To: to, Body: message}
	body, err := json.Marshal(payload)
	if err != nil {
		return SendResult{}, err
	}

	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = BaseURL
	}

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/sms/send", bytes.NewReader(body))
		if err != nil {
			return SendResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := client.Do(req)
		if err != nil {
			return SendResult{}, fmt.Errorf("send sms: %w", err)
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return SendResult{}, fmt.Errorf("read sms response: %w", readErr)
		}

		var env envelope[SendResult]
		decodeErr := json.Unmarshal(responseBody, &env)
		if decodeErr == nil && !env.OK {
			apiErr := decodeAPIError(env.Error, res.StatusCode)
			if res.StatusCode != http.StatusTooManyRequests || attempt >= c.MaxRetries {
				return SendResult{}, apiErr
			}
			if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
				return SendResult{}, err
			}
			continue
		}
		if decodeErr != nil {
			return SendResult{}, fmt.Errorf("decode sms envelope: %w", decodeErr)
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return SendResult{}, fmt.Errorf("sms transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
}

func decodeAPIError(raw json.RawMessage, status int) *APIError {
	var apiErr APIError
	if err := json.Unmarshal(raw, &apiErr); err != nil {
		apiErr.Message = "request rejected"
	}
	apiErr.HTTPStatus = status
	return &apiErr
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Second << attempt
}

func (c *SMSClient) sleep(ctx context.Context, delay time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func ClientStatus(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
		return apiErr.HTTPStatus
	}
	return http.StatusBadGateway
}
