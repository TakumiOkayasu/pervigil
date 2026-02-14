package anthropic

// UsageReport represents the response from /v1/usage endpoint.
type UsageReport struct {
	Data []UsageBucket `json:"data"`
}

// UsageBucket represents a single usage data point.
type UsageBucket struct {
	Date         string `json:"date"`
	Model        string `json:"model"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
}

// CostReport represents the response from /v1/cost endpoint.
type CostReport struct {
	Data []CostBucket `json:"data"`
}

// CostBucket represents a single cost data point.
type CostBucket struct {
	Date    string  `json:"date"`
	CostUSD float64 `json:"cost_usd"`
}
