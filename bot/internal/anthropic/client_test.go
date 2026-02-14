package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetUsage(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantErr   bool
		wantCount int
	}{
		{
			name:   "success",
			status: 200,
			body: `{"data":[
				{"date":"2025-01-01","model":"claude-sonnet-4-20250514","input_tokens":1000,"output_tokens":500},
				{"date":"2025-01-01","model":"claude-haiku-4-20250514","input_tokens":2000,"output_tokens":800}
			]}`,
			wantCount: 2,
		},
		{
			name:    "api_error",
			status:  401,
			body:    `{"error":"unauthorized"}`,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			status:  200,
			body:    `{invalid`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					if got := req.Header.Get("x-api-key"); got != "test-key" {
						t.Errorf("x-api-key = %q, want %q", got, "test-key")
					}
					if got := req.Header.Get("anthropic-version"); got != apiVersion {
						t.Errorf("anthropic-version = %q, want %q", got, apiVersion)
					}
					return jsonResponse(tt.status, tt.body), nil
				},
			}

			c := NewClient("test-key", WithHTTPClient(mock))
			start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

			report, err := c.GetUsage(context.Background(), start, end, "model")
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(report.Data) != tt.wantCount {
				t.Errorf("data count = %d, want %d", len(report.Data), tt.wantCount)
			}
		})
	}
}

func TestGetCost(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr bool
		wantUSD float64
	}{
		{
			name:    "success",
			status:  200,
			body:    `{"data":[{"date":"2025-01-01","cost_usd":3.45}]}`,
			wantUSD: 3.45,
		},
		{
			name:    "api_error",
			status:  500,
			body:    `{"error":"internal"}`,
			wantErr: true,
		},
		{
			name:    "invalid_json",
			status:  200,
			body:    `not-json`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockHTTPClient{
				doFunc: func(_ *http.Request) (*http.Response, error) {
					return jsonResponse(tt.status, tt.body), nil
				},
			}

			c := NewClient("test-key", WithHTTPClient(mock))
			start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			end := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

			report, err := c.GetCost(context.Background(), start, end)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(report.Data) == 0 {
				t.Fatal("expected data, got empty")
			}
			if report.Data[0].CostUSD != tt.wantUSD {
				t.Errorf("cost = %f, want %f", report.Data[0].CostUSD, tt.wantUSD)
			}
		})
	}
}

func TestGetUsageURL(t *testing.T) {
	var capturedURL string
	mock := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			capturedURL = req.URL.String()
			return jsonResponse(200, `{"data":[]}`), nil
		},
	}

	c := NewClient("key", WithHTTPClient(mock), WithBaseURL("https://example.com"))
	start := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)

	if _, err := c.GetUsage(context.Background(), start, end, "model"); err != nil {
		t.Fatal(err)
	}

	// url.Values.Encode() sorts keys alphabetically
	parsed, err := url.Parse(capturedURL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if parsed.Path != "/v1/usage" {
		t.Errorf("path = %q, want %q", parsed.Path, "/v1/usage")
	}
	q := parsed.Query()
	if got := q.Get("start_date"); got != "2025-03-01" {
		t.Errorf("start_date = %q, want %q", got, "2025-03-01")
	}
	if got := q.Get("end_date"); got != "2025-03-31" {
		t.Errorf("end_date = %q, want %q", got, "2025-03-31")
	}
	if got := q.Get("group_by"); got != "model" {
		t.Errorf("group_by = %q, want %q", got, "model")
	}
}

func TestContextCancellation(t *testing.T) {
	mock := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			// Respect the request's context like a real HTTP client
			if err := req.Context().Err(); err != nil {
				return nil, err
			}
			return jsonResponse(200, `{"data":[]}`), nil
		},
	}

	c := NewClient("key", WithHTTPClient(mock))
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.GetCost(ctx, time.Now(), time.Now())
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
