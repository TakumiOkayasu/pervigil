package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Color represents Discord embed colors
type Color int

const (
	ColorGreen  Color = 5763719  // 0x57f287
	ColorYellow Color = 16776960 // 0xffff00
	ColorRed    Color = 15548997 // 0xed4245
	ColorBlue   Color = 5793266  // 0x5865f2
)

// Field represents a Discord embed field
type Field struct {
	Name   string
	Value  string
	Inline bool
}

// Notifier sends notifications
type Notifier interface {
	Send(title, message string, color Color, fields []Field) error
}

// httpClient abstracts HTTP operations (ISP)
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DiscordNotifier sends notifications via Discord webhook
type DiscordNotifier struct {
	webhookURL string
	client     httpClient
}

// Option configures DiscordNotifier
type Option func(*DiscordNotifier)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(c httpClient) Option {
	return func(n *DiscordNotifier) {
		n.client = c
	}
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(webhookURL string, opts ...Option) *DiscordNotifier {
	n := &DiscordNotifier{
		webhookURL: webhookURL,
		client:     http.DefaultClient,
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

// webhookPayload is the Discord webhook JSON structure
type webhookPayload struct {
	Username string  `json:"username"`
	Embeds   []embed `json:"embeds"`
}

type embed struct {
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Color       int          `json:"color"`
	Fields      []embedField `json:"fields,omitempty"`
	Timestamp   string       `json:"timestamp"`
}

type embedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

// Send sends a notification to Discord
func (d *DiscordNotifier) Send(title, message string, color Color, fields []Field) error {
	embedFields := make([]embedField, len(fields))
	for i, f := range fields {
		embedFields[i] = embedField(f)
	}

	payload := webhookPayload{
		Username: "Pervigil",
		Embeds: []embed{
			{
				Title:       title,
				Description: message,
				Color:       int(color),
				Fields:      embedFields,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord API error: status %d", resp.StatusCode)
	}

	return nil
}
