package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// ProxyConfig describes one proxy in the pool.
type ProxyConfig struct {
	URL         string        `json:"url"` // scheme://host:port
	Username    string        `json:"username,omitempty"`
	Password    string        `json:"password,omitempty"`
	Country     string        `json:"country,omitempty"` // optional hint
	Weight      int           `json:"weight,omitempty"`  // higher = more traffic
	Timeout     time.Duration `json:"timeout,omitempty"`
	Concurrency int           `json:"concurrency,omitempty"`
}

// ProxyStatus is the runtime health of one proxy.
type ProxyStatus struct {
	URL         string
	Alive       bool
	Latency     time.Duration
	Failures    int
	Successes   int
	LastChecked time.Time
	InUse       int
}

// Pool manages a set of proxies with health-aware rotation.
type Pool struct {
	mu           sync.Mutex
	proxies      []ProxyConfig
	status       map[string]*ProxyStatus
	healthTicker *time.Ticker
}

func newPoolFromConfig(path string) (*Pool, error) {
	if path == "" {
		return nil, fmt.Errorf("config path required (pass -config)")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var proxies []ProxyConfig
	if err := json.Unmarshal(data, &proxies); err != nil {
		return nil, err
	}
	if len(proxies) == 0 {
		return nil, fmt.Errorf("empty proxy pool")
	}
	p := &Pool{
		proxies: proxies,
		status:  make(map[string]*ProxyStatus, len(proxies)),
	}
	for _, pr := range proxies {
		p.status[pr.URL] = &ProxyStatus{URL: pr.URL, Alive: true}
	}
	return p, nil
}

func (p *Pool) healthLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		p.checkAll(context.Background())
	}
}

func (p *Pool) checkAll(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, pr := range p.proxies {
		st := p.status[pr.URL]
		st.LastChecked = time.Now()
		start := time.Now()
		alive, _ := p.probe(ctx, pr)
		st.Latency = time.Since(start)
		st.Alive = alive
		if !alive {
			st.Failures++
		}
	}
}

func (p *Pool) probe(ctx context.Context, pr ProxyConfig) (bool, error) {
	timeout := pr.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.meta.com/v1/ping", nil)
	client, err := p.clientFor(pr)
	if err != nil {
		return false, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500, nil
}

func (p *Pool) clientFor(pr ProxyConfig) (*http.Client, error) {
	timeout := pr.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	transport := &http.Transport{
		DialContext:     dialer.DialContext,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if pr.URL != "" && pr.URL != "direct" {
		proxyURL, err := ParseProxyURL(pr.URL)
		if err != nil {
			return nil, err
		}
		if proxyURL.Scheme == "socks5" {
			// Use proxy with auth
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp", pr.URL)
			}
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	return &http.Client{Timeout: timeout, Transport: transport}, nil
}

func (p *Pool) next(targetHost string) (*ProxyConfig, *ProxyStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var best *ProxyConfig
	var bestStatus *ProxyStatus
	bestScore := -1
	for _, pr := range p.proxies {
		st := p.status[pr.URL]
		if !st.Alive {
			continue
		}
		score := st.Successes*2 - st.Failures
		if bestScore < 0 || score > bestScore {
			best = &pr
			bestStatus = st
			bestScore = score
		}
	}
	if best == nil {
		// fallback direct if all dead
		return &ProxyConfig{URL: "direct"}, &ProxyStatus{URL: "direct", Alive: true}
	}
	return best, bestStatus
}

func (p *Pool) markResult(pr *ProxyConfig, success bool) {
	if pr == nil || pr.URL == "" || pr.URL == "direct" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	st := p.status[pr.URL]
	if st == nil {
		st = &ProxyStatus{URL: pr.URL, Alive: true}
		p.status[pr.URL] = st
	}
	if success {
		st.Successes++
	} else {
		st.Failures++
	}
}

func ParseProxyURL(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
