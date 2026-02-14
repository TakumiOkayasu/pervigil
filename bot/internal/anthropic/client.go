package anthropic

import (
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.anthropic.com"

// defaultHTTPClient provides a safety-net timeout longer than typical
// context deadlines (30s) so context cancellation fires first.
var defaultHTTPClient = &http.Client{Timeout: 60 * time.Second}

// httpClient abstracts HTTP operations for testing.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is an Anthropic Admin API client.
type Client struct {
	apiKey  string
	baseURL string
	http    httpClient
}

// ClientOption configures Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c httpClient) ClientOption {
	return func(cl *Client) {
		cl.http = c
	}
}

// WithBaseURL sets a custom base URL.
func WithBaseURL(url string) ClientOption {
	return func(cl *Client) {
		cl.baseURL = url
	}
}

// NewClient creates a new Anthropic Admin API client.
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		http:    defaultHTTPClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
