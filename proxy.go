package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func runProxyRequest(ctx context.Context, args map[string]any, pool *Pool) (string, error) {
	rawURL, ok := args["url"].(string)
	if !ok || rawURL == "" {
		return "", fmt.Errorf("url required")
	}
	method := strings.ToUpper(strings.TrimSpace(args["method"].(string)))
	if method == "" {
		method = "GET"
	}
	var headers map[string]string
	if h, ok := args["headers"].(map[string]any); ok {
		headers = make(map[string]string, len(h))
		for k, v := range h {
			headers[k] = fmt.Sprint(v)
		}
	}
	var body io.Reader
	if b, ok := args["body"].(string); ok && b != "" {
		body = strings.NewReader(b)
	}

	pr, st := pool.next(rawURL)
	st.InUse++
	defer func() { st.InUse-- }()

	client, err := pool.clientFor(*pr)
	if err != nil {
		return "", fmt.Errorf("proxy %s: %w", pr.URL, err)
	}

	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if _, ok := headers["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "proxy-mcp/1.0")
	}

	resp, err := client.Do(req)
	success := err == nil
	pool.markResult(pr, success)
	if err != nil {
		return "", fmt.Errorf("proxy %s: %w", pr.URL, err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 200_000))
	if err != nil {
		return "", err
	}
	out := map[string]any{
		"proxy":        pr.URL,
		"status":       resp.StatusCode,
		"headers":      resp.Header,
		"body_preview": truncate(string(b), 20000),
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data), nil
}

func runProxyHealth(ctx context.Context, _ map[string]any, pool *Pool) (string, error) {
	pool.checkAll(ctx)
	pool.mu.Lock()
	defer pool.mu.Unlock()
	type row struct {
		URL         string  `json:"url"`
		Alive       bool    `json:"alive"`
		LatencyMS   int64   `json:"latency_ms"`
		Failures    int     `json:"failures"`
		Successes   int     `json:"successes"`
		InUse       int     `json:"in_use"`
		LastChecked string  `json:"last_checked"`
		SuccessRate float64 `json:"success_rate"`
	}
	rows := make([]row, 0, len(pool.status))
	for _, st := range pool.status {
		total := st.Successes + st.Failures
		rate := 0.0
		if total > 0 {
			rate = float64(st.Successes) / float64(total)
		}
		rows = append(rows, row{
			URL:         st.URL,
			Alive:       st.Alive,
			LatencyMS:   st.Latency.Milliseconds(),
			Failures:    st.Failures,
			Successes:   st.Successes,
			InUse:       st.InUse,
			LastChecked: st.LastChecked.Format(time.RFC3339),
			SuccessRate: round(rate*100, 2),
		})
	}
	data, _ := json.MarshalIndent(rows, "", "  ")
	return string(data), nil
}

func runProxyList(ctx context.Context, _ map[string]any, pool *Pool) (string, error) {
	return runProxyHealth(ctx, nil, pool)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}

func round(f float64, n int) float64 {
	p := 1.0
	for i := 0; i < n; i++ {
		p *= 10
	}
	return float64(int64(f*p+0.5)) / p
}
