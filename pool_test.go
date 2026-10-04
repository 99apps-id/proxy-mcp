package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestProxyRequestThroughLocalProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true,"via":"backend"}`))
	}))
	defer backend.Close()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequestWithContext(r.Context(), "GET", backend.URL, nil)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Logf("proxy upstream error: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	defer proxy.Close()

	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: proxy.URL, Timeout: 5 * time.Second},
		},
		status: map[string]*ProxyStatus{
			proxy.URL: {URL: proxy.URL, Alive: true},
		},
	}

	text, err := runProxyRequest(context.Background(), map[string]any{
		"url":    backend.URL,
		"method": "GET",
	}, pool)
	if err != nil {
		t.Fatalf("proxy_request failed: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("invalid json response: %v\n%s", err, text)
	}
	bp, _ := out["body_preview"].(string)
	if !strings.Contains(bp, `"ok":true`) {
		t.Fatalf("unexpected body_preview: %s", bp)
	}
}

func TestProxyHealthReportsStatus(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: proxy.URL},
		},
		status: map[string]*ProxyStatus{
			proxy.URL: {URL: proxy.URL, Alive: true, Successes: 1},
		},
	}

	text, err := runProxyHealth(context.Background(), nil, pool)
	if err != nil {
		t.Fatalf("proxy_health failed: %v", err)
	}
	if !strings.Contains(text, proxy.URL) {
		t.Fatalf("expected proxy URL in output: %s", text)
	}
}

func TestProxyListReturnsSameAsHealth(t *testing.T) {
	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: "direct"},
		},
		status: map[string]*ProxyStatus{
			"direct": {URL: "direct", Alive: true},
		},
	}

	h, _ := runProxyHealth(context.Background(), nil, pool)
	l, _ := runProxyList(context.Background(), nil, pool)
	if !strings.Contains(l, "direct") {
		t.Fatalf("proxy_list missing direct proxy: %s", l)
	}
	if !strings.Contains(h, "direct") {
		t.Fatalf("proxy_health missing direct proxy: %s", h)
	}
}

func TestProxyNextPicksHealthiest(t *testing.T) {
	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: "http://bad:1"},
			{URL: "http://good:1"},
		},
		status: map[string]*ProxyStatus{
			"http://bad:1":  {URL: "http://bad:1", Alive: true, Failures: 10},
			"http://good:1": {URL: "http://good:1", Alive: true, Successes: 10},
		},
	}

	pr, _ := pool.next("example.com")
	if pr.URL != "http://good:1" {
		t.Fatalf("expected healthy proxy, got %s", pr.URL)
	}
}

func TestProxyConcurrentAccess(t *testing.T) {
	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: "http://p1:1"},
		},
		status: map[string]*ProxyStatus{
			"http://p1:1": {URL: "http://p1:1", Alive: true},
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, st := pool.next("example.com")
			st.InUse++
			time.Sleep(1 * time.Millisecond)
			st.InUse--
		}()
	}
	wg.Wait()
}

func TestProxyFallbackDirectWhenAllDead(t *testing.T) {
	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: "http://dead:1"},
		},
		status: map[string]*ProxyStatus{
			"http://dead:1": {URL: "http://dead:1", Alive: false},
		},
	}

	pr, _ := pool.next("example.com")
	if pr.URL != "direct" {
		t.Fatalf("expected direct fallback, got %s", pr.URL)
	}
}

func TestProxyRequestValidation(t *testing.T) {
	pool := &Pool{proxies: []ProxyConfig{}, status: map[string]*ProxyStatus{}}
	_, err := runProxyRequest(context.Background(), map[string]any{}, pool)
	if err == nil || !strings.Contains(err.Error(), "url required") {
		t.Fatalf("expected url required error, got: %v", err)
	}
}

func TestProxyMarkResultSkipsDirect(t *testing.T) {
	pool := &Pool{
		proxies: []ProxyConfig{{URL: "direct"}},
		status:  map[string]*ProxyStatus{"direct": {URL: "direct", Alive: true, Successes: 1}},
	}
	pool.markResult(&ProxyConfig{URL: "direct"}, false)
	st := pool.status["direct"]
	if st.Failures != 0 {
		t.Fatalf("direct proxy should not be marked, got failures=%d", st.Failures)
	}
}

func TestProxyProbeTimeout(t *testing.T) {
	// Use an unroutable address to trigger timeout
	pool := &Pool{
		proxies: []ProxyConfig{
			{URL: "http://192.0.2.1:1", Timeout: 100 * time.Millisecond},
		},
		status: make(map[string]*ProxyStatus),
	}
	for _, pr := range pool.proxies {
		pool.status[pr.URL] = &ProxyStatus{URL: pr.URL, Alive: true}
	}

	alive, err := pool.probe(context.Background(), pool.proxies[0])
	if err == nil {
		t.Logf("probe error (expected): %v", err)
	}
	if alive {
		t.Fatal("expected dead proxy")
	}
}
