# proxy-mcp

MCP server that rotates outbound HTTP(S) requests through a pool of proxies.
Use it when an API (for example Meta/Muse) blocks your server IP or ISP DNS.

## Install

### Build from source

```bash
git clone https://github.com/99apps-id/proxy-mcp.git
cd proxy-mcp
go build -o proxy-mcp .
```

### Download binary

Pre-built binaries are available for Linux, macOS, and Windows on the GitHub Releases page.

## Configure

Create a JSON file listing your proxies, for example `./proxies.json`:

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
# Linux / macOS
./proxy-mcp -config ./proxies.json

# Windows (PowerShell)
.\proxy-mcp.exe -config .\proxies.json

# or with environment variable
# Linux / macOS
export PROXY_MCP_CONFIG=./proxies.json
./proxy-mcp

# Windows (PowerShell)
$env:PROXY_MCP_CONFIG = ".\proxies.json"
.\proxy-mcp.exe
```

## Register in Editors and Agents

### Termigo

Edit your user MCP config:

- **Linux/macOS**: `~/.termigo/mcp.json`
- **Windows**: `C:\Users\Nesa\.termigo\mcp.json`

```json
{
  "mcpServers": {
    "proxy": {
      "command": "/absolute/path/to/proxy-mcp",
      "args": ["-config", "/absolute/path/to/proxies.json"]
    }
  }
}
```

Use forward slashes or escaped backslashes in JSON. Restart Termigo to load the server.

### VS Code (MCP Extension)

Add to your VS Code MCP configuration:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "/absolute/path/to/proxy-mcp",
      "args": ["-config", "/absolute/path/to/proxies.json"]
    }
  }
}
```

### Claude Code

```bash
claude mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### Codex

```bash
codex mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### OpenCode

```bash
opencode mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### OpenClaw

```bash
openclaw mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### Hermes

```bash
hermes mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### 9router

```bash
9router mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### Termixgo

```bash
termixgo mcp add proxy -- /absolute/path/to/proxy-mcp -config /absolute/path/to/proxies.json
```

### Generic MCP Configuration

Any MCP client that supports stdio servers:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "/absolute/path/to/proxy-mcp",
      "args": ["-config", "/absolute/path/to/proxies.json"]
    }
  }
}
```

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

## Usage from AI Agent

Once registered, you can ask the agent to use the proxy pool directly:

- "Fetch https://api.meta.com/v1/models through the proxy pool"
- "Check proxy health"
- "List proxies"

The agent will call `proxy_request`, `proxy_health`, or `proxy_list` via MCP.

## Cross-Platform Notes

- Binary name is `proxy-mcp` on Linux/macOS, `proxy-mcp.exe` on Windows.
- Config paths use platform-native separators, but forward slashes work everywhere.
- No external dependencies beyond the Go standard library.
- SOCKS5 proxies require TCP dial support, which is available on all platforms.

## Notes

- Requests are pinned to a single proxy per call. The next call rotates based on success rate and health.
- Failed proxies are not removed, but they are de-prioritized automatically.
- All core Termigo SSRF protections still apply. This MCP only chooses the egress path; it does not bypass private-IP or metadata-IP filters.
- For Meta/Muse specifically, choose proxies with Indonesian egress IPs if Meta blocks datacenter ranges.
