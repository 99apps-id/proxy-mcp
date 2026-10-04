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
- **Windows**: `%USERPROFILE%\.termigo\mcp.json`

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<PROXY_MCP_DIR>/proxies.json"]
    }
  }
}
```

- On Linux/macOS, replace `<PROXY_MCP_DIR>` with the absolute path to the proxy-mcp binary, e.g. `/usr/local/bin` or `$HOME/.local/bin`.
- On Windows, replace `<PROXY_MCP_DIR>` with the absolute path to the proxy-mcp binary, e.g. `C:\\Users\\<USER>\\bin`.
- Use forward slashes or escaped backslashes in JSON. Restart Termigo to load the server.

### VS Code (MCP Extension)

Add to your VS Code MCP configuration:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<PROXY_MCP_DIR>/proxies.json"]
    }
  }
}
```

Replace `<PROXY_MCP_DIR>` with the absolute path to the proxy-mcp binary on your platform.

### Claude Code

```bash
claude mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### Codex

```bash
codex mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### OpenCode

```bash
opencode mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### OpenClaw

```bash
openclaw mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### Hermes

```bash
hermes mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### 9router

```bash
9router mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### Termixgo

```bash
termixgo mcp add proxy -- <PROXY_MCP_DIR>/proxy-mcp -config <PROXY_MCP_DIR>/proxies.json
```

### Generic MCP Configuration

Any MCP client that supports stdio servers:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<PROXY_MCP_DIR>/proxies.json"]
    }
  }
}
```

Replace `<PROXY_MCP_DIR>` with the absolute path to the proxy-mcp binary on your platform.

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

## Multi-Port SOCKS5 Configuration

You can run multiple SOCKS5 proxies on different local ports to avoid port-level blocking. Each proxy is a separate entry in `proxies.json`:

```json
[
  {
    "url": "socks5://127.0.0.1:9050",
    "country": "tor",
    "weight": 1,
    "timeout": "30s"
  },
  {
    "url": "socks5://127.0.0.1:9051",
    "country": "tor",
    "weight": 1,
    "timeout": "30s"
  },
  {
    "url": "socks5://127.0.0.1:9052",
    "country": "tor",
    "weight": 1,
    "timeout": "30s"
  }
]
```

### When multi-port helps

- **Port-level blocking**: Some ISPs or networks block specific ports (e.g., 9050 for Tor). Using multiple ports (9050, 9051, 9052) bypasses this if the ISP only blocks well-known ports.
- **Rate limiting per port**: Distributing traffic across ports can help avoid per-port rate limits.

### When multi-port does NOT help

- **Deep Packet Inspection (DPI)**: If the ISP inspects packet contents and detects SOCKS5 protocol handshakes, changing ports won't help. DPI sees the protocol, not just the port.
- **IP-based blocking**: If the target API (like Meta/Muse) blocks your server's IP range (e.g., Contabo datacenter IPs), multiple local SOCKS5 ports won't help unless the SOCKS5 proxy itself egresses through a different, unblocked IP.
- **SNI filtering**: If the ISP filters based on TLS SNI, you need a proxy that terminates TLS and forwards with a different SNI.

### For Meta/Muse specifically

The block is likely **IP/ASN-based** (Meta blocks datacenter IP ranges), not port-based. Multi-port SOCKS5 only helps if:
1. The SOCKS5 proxy is a **residential/mobile proxy** with Indonesian egress IP, OR
2. The SOCKS5 proxy routes through an ISP that Meta doesn't block

If you're running local Tor instances on multiple ports, all egress still comes from the same Tor circuit IP, which is likely still a datacenter IP that Meta blocks. Use **residential HTTP proxies** or **mobile proxies** for Meta/Muse instead.

## Notes

- Requests are pinned to a single proxy per call. The next call rotates based on success rate and health.
- Failed proxies are not removed, but they are de-prioritized automatically.
- All core Termigo SSRF protections still apply. This MCP only chooses the egress path; it does not bypass private-IP or metadata-IP filters.
- For Meta/Muse specifically, choose proxies with Indonesian egress IPs if Meta blocks datacenter ranges.
