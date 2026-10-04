package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"
)

const protocolVersion = "2025-06-18"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type server struct {
	out  *bufio.Writer
	mu   sync.Mutex
	pool *Pool
}

func (s *server) serve(in io.Reader) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		s.handle(req)
	}
	return scanner.Err()
}

func (s *server) handle(req request) {
	if len(req.ID) == 0 {
		return
	}
	switch req.Method {
	case "initialize":
		s.send(response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "proxy-mcp", "version": "1.0.0"},
		}})
	case "ping":
		s.send(response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})
	case "tools/list":
		s.send(response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolSchemas()}})
	case "tools/call":
		s.handleToolCall(req)
	default:
		s.send(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found: " + req.Method}})
	}
}

func (s *server) handleToolCall(req request) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.send(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "invalid params"}})
		return
	}
	tool, ok := toolByName(params.Name)
	if !ok {
		s.send(response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32602, Message: "unknown tool: " + params.Name}})
		return
	}
	text, err := tool.Run(context.Background(), params.Arguments, s.pool)
	if err != nil {
		s.send(response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"content": []map[string]any{{"type": "text", "text": err.Error()}},
			"isError": true,
		}})
		return
	}
	s.send(response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}})
}

func (s *server) send(v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	s.out.Write(data)
	s.out.WriteByte('\n')
	s.out.Flush()
}

type tool struct {
	Name        string
	Description string
	Schema      map[string]any
	Run         func(ctx context.Context, args map[string]any, pool *Pool) (string, error)
}

func toolByName(name string) (tool, bool) {
	for _, candidate := range tools() {
		if candidate.Name == name {
			return candidate, true
		}
	}
	return tool{}, false
}

func tools() []tool {
	return []tool{
		{
			Name:        "proxy_request",
			Description: "Make an HTTP(S) request through the proxy pool. The pool picks the healthiest proxy automatically and rotates on failure.",
			Schema: object(map[string]any{
				"url":     strProp("Absolute http(s) URL to request."),
				"method":  strProp("HTTP method. Default GET."),
				"headers": strProp("Optional JSON object of headers."),
				"body":    strProp("Optional request body as string."),
			}, "url"),
			Run: runProxyRequest,
		},
		{
			Name:        "proxy_health",
			Description: "Run a quick health check against every proxy in the pool and report status.",
			Schema:      object(map[string]any{}),
			Run:         runProxyHealth,
		},
		{
			Name:        "proxy_list",
			Description: "List proxies in the pool with current health status and success rate.",
			Schema:      object(map[string]any{}),
			Run:         runProxyList,
		},
	}
}

func toolSchemas() []map[string]any {
	defs := tools()
	schemas := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		schemas = append(schemas, map[string]any{
			"name":        def.Name,
			"description": def.Description,
			"inputSchema": def.Schema,
		})
	}
	return schemas
}

func object(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func strProp(description string) map[string]any { return prop("string", description) }

func prop(kind, description string) map[string]any {
	return map[string]any{"type": kind, "description": description}
}
