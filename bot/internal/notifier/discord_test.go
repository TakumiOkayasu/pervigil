package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestDiscordNotifier_Send(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"success 200", 200, false},
		{"success 204", 204, false},
		{"client error 400", 400, true},
		{"server error 500", 500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.statusCode,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				},
			}

			n := NewDiscordNotifier("https://example.com/webhook", WithHTTPClient(client))
			err := n.Send("Test", "Message", ColorGreen, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiscordNotifier_Send_NetworkError(t *testing.T) {
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network error")
		},
	}

	n := NewDiscordNotifier("https://example.com/webhook", WithHTTPClient(client))
	err := n.Send("Test", "Message", ColorGreen, nil)

	if err == nil {
		t.Error("expected error for network failure")
	}
}

func TestDiscordNotifier_Send_PayloadFormat(t *testing.T) {
	var capturedBody []byte

	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			capturedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader([]byte{})),
			}, nil
		},
	}

	n := NewDiscordNotifier("https://example.com/webhook", WithHTTPClient(client))
	fields := []Field{
		{Name: "Temperature", Value: "75°C", Inline: true},
	}
	_ = n.Send("Title", "Description", ColorYellow, fields)

	var payload webhookPayload
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("invalid JSON payload: %v", err)
	}

	if payload.Username != "Pervigil" {
		t.Errorf("username = %q, want Pervigil", payload.Username)
	}
	if len(payload.Embeds) != 1 {
		t.Fatalf("embeds count = %d, want 1", len(payload.Embeds))
	}
	if payload.Embeds[0].Title != "Title" {
		t.Errorf("title = %q, want Title", payload.Embeds[0].Title)
	}
	if payload.Embeds[0].Color != int(ColorYellow) {
		t.Errorf("color = %d, want %d", payload.Embeds[0].Color, ColorYellow)
	}
	if len(payload.Embeds[0].Fields) != 1 {
		t.Errorf("fields count = %d, want 1", len(payload.Embeds[0].Fields))
	}
}

func TestColor_Values(t *testing.T) {
	if ColorGreen != 5763719 {
		t.Errorf("ColorGreen = %d, want 5763719", ColorGreen)
	}
	if ColorYellow != 16776960 {
		t.Errorf("ColorYellow = %d, want 16776960", ColorYellow)
	}
	if ColorRed != 15548997 {
		t.Errorf("ColorRed = %d, want 15548997", ColorRed)
	}
	if ColorBlue != 5793266 {
		t.Errorf("ColorBlue = %d, want 5793266", ColorBlue)
	}
}
