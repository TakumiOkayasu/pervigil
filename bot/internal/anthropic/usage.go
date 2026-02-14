package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const apiVersion = "2023-06-01"

// GetUsage fetches token usage data for the given date range.
// groupBy can be "model" or "date" (empty defaults to API behavior).
func (c *Client) GetUsage(ctx context.Context, start, end time.Time, groupBy string) (*UsageReport, error) {
	params := url.Values{
		"start_date": {start.Format("2006-01-02")},
		"end_date":   {end.Format("2006-01-02")},
	}
	if groupBy != "" {
		params.Set("group_by", groupBy)
	}

	var report UsageReport
	if err := c.doGet(ctx, c.baseURL+"/v1/usage?"+params.Encode(), &report); err != nil {
		return nil, fmt.Errorf("fetch usage: %w", err)
	}
	return &report, nil
}

// GetCost fetches cost data for the given date range.
func (c *Client) GetCost(ctx context.Context, start, end time.Time) (*CostReport, error) {
	params := url.Values{
		"start_date": {start.Format("2006-01-02")},
		"end_date":   {end.Format("2006-01-02")},
	}

	var report CostReport
	if err := c.doGet(ctx, c.baseURL+"/v1/cost?"+params.Encode(), &report); err != nil {
		return nil, fmt.Errorf("fetch cost: %w", err)
	}
	return &report, nil
}

func (c *Client) doGet(ctx context.Context, reqURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", apiVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// Limit response body to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody := string(body)
		if len(errBody) > 512 {
			errBody = errBody[:512] + "...(truncated)"
		}
		return fmt.Errorf("API error: status %d: %s", resp.StatusCode, errBody)
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
