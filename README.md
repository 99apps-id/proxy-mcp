# proxy-mcp

MCP server that rotates outbound HTTP(S) requests through a pool of proxies.
Use it when an API (for example Meta/Muse) blocks your server IP or ISP DNS.

## Install

```bash
go build -o proxy-mcp.exe .
```

Binary: `C:/project/proxy-mcp/proxy-mcp.exe`

## Configure

Create a JSON file listing your proxies, for example `C:/project/proxy-mcp/proxies.json`:

```json
[
  {
    "url": "http://user:pass@proxy-host:8080",
    "country": "ID",
    "weight": 10,
    "timeout": "10s"
  },
  {
    "url": "socks5://127.0.0.1:9050",
    "country": "tor",
    "weight": 1,
    "timeout": "30s"
  }
]
```

Fields:
- `url`: required. `http://`, `https://`, or `socks5://`.
- `username` / `password`: optional basic auth.
- `country`: optional hint for the agent.
- `weight`: higher = more traffic.
- `timeout`: per-request timeout (`10s`, `30s`).
- `concurrency`: optional connection limit.

## Run

```bash
# as Termigo MCP server
proxy-mcp.exe -config C:\path\to\proxies.json

# or with environment variable
set PROXY_MCP_CONFIG=C:\path\to\proxies.json
proxy-mcp.exe
```

## Register in Termigo

Edit your user MCP config and add the server:

`C:/Users/Nesa/.termigo/mcp.json`

```json
{
  "mcpServers": {
    "doh": {
      "command": "C:/project/doh-mcp/doh-mcp.exe",
      "args": ["-resolver", "cloudflare"]
    },
    "proxy": {
      "command": "C:/project/proxy-mcp/proxy-mcp.exe",
      "args": ["-config", "C:/project/proxy-mcp/proxies.json"]
    }
  }
}
```

Then restart Termigo so it loads the new MCP server.

## Tools

### `proxy_request`
Make an HTTP(S) request through the healthiest available proxy.

```json
{
  "url": "https://api.meta.com/v1/models",
  "method": "GET",
  "headers": {"Authorization": "Bearer <token>"}
}
```

Returns status, proxy used, headers, and body preview.

### `proxy_health`
Run a quick connectivity check against every proxy and report status.

### `proxy_list`
Same as `proxy_health`; lists all proxies with current health status and success rate.

## Usage from AI agent

Once registered, you can ask the agent to use the proxy pool directly:

- "Fetch https://api.meta.com/v1/models through the proxy pool"
- "Check proxy health"
- "List proxies"

The agent will call `proxy_request`, `proxy_health`, or `proxy_list` via MCP.

## Notes

- Requests are pinned to a single proxy per call. The next call rotates based on success rate and health.
- Failed proxies are not removed, but they are de-prioritized automatically.
- All core Termigo SSRF protections still apply. This MCP only chooses the egress path; it does not bypass private-IP or metadata-IP filters.
- For Meta/Muse specifically, choose proxies with Indonesian egress IPs if Meta blocks datacenter ranges.
