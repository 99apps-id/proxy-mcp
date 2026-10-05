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
- `url`: required. `http://`, `https://`, or `socks5://`. Supports `user:pass@host:port`.
- `country`: optional hint for proxy geolocation (e.g. `"ID"`, `"US"`, `"tor"`).
- `weight`: optional traffic weight (higher = more requests routed here).
- `timeout`: optional per-proxy timeout (e.g. `"10s"`, `"30s"`).
- `concurrency`: optional max concurrent connections per proxy.

## Tools Provided

| Tool | Description |
|------|-------------|
| `proxy_request` | Send HTTP(S) request through the proxy pool |
| `proxy_health` | Health-check all proxies and return status |
| `proxy_list` | List proxies with success rate and status |

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

- **Port-level blocking**: Some ISPs or networks block specific ports (e.g. 9050 for Tor) but not others.
- **Rate limiting per port**: Distributing traffic across multiple ports avoids per-port rate limits.

### When multi-port does NOT help

- **DPI / SNI filtering**: Deep packet inspection can still identify and block the traffic regardless of port.
- **IP-level blocking**: If the proxy server's IP is blocked, changing ports won't help.

For IP-level blocking, use exit-node proxies from different networks or countries.

## Setting Up a Proxy Server

If you have your own VPS or server, you can run your own proxy. Below are common setups for HTTP and SOCKS5 proxies.

### Option A: Squid (HTTP/HTTPS)

Squid is the most common HTTP proxy server.

**Install (Linux):**

```bash
# Debian / Ubuntu
sudo apt update
sudo apt install squid apache2-utils -y

# RHEL / CentOS / Fedora
sudo dnf install squid httpd-tools -y
```

**Install (macOS):**

```bash
brew install squid
```

**Configure credentials:**

```bash
sudo htpasswd -c /etc/squid/passwd youruser
# Enter password twice when prompted
```

**Configure Squid** (edit `/etc/squid/squid.conf` or `/usr/local/etc/squid/squid.conf`):

```
auth_param basic program /usr/lib/squid/basic_ncsa_auth /etc/squid/passwd
auth_param basic realm proxy
acl authenticated proxy_auth REQUIRED
http_access allow authenticated
http_access deny all
http_port 3128
```

**Start Squid:**

```bash
# Linux (systemd)
sudo systemctl enable --now squid

# macOS
sudo brew services start squid
```

**Proxy URL format:**
```
http://youruser:yourpassword@your-vps-hostname:3128
```

### Option B: 3proxy (HTTP + SOCKS5, lightweight)

3proxy is a tiny proxy server suitable for low-resource VPS.

**Install (Linux):**

```bash
# Debian / Ubuntu
sudo apt install 3proxy -y

# Or build from source
git clone https://github.com/z3APA3A/3proxy.git
cd 3proxy
make -f Makefile.Linux
sudo cp 3proxy /usr/local/bin/
```

**Configure** (`/etc/3proxy/3proxy.cfg`):

```
auth strong
users youruser:CL:yourpassword
proxy -p3128 -n
socks -p9050 -n
```

**Start:**

```bash
sudo 3proxy /etc/3proxy/3proxy.cfg
```

**Proxy URLs:**
```
http://youruser:yourpassword@your-vps-hostname:3128
socks5://youruser:yourpassword@your-vps-hostname:9050
```

### Option C: Dante (SOCKS5 only)

Dante is a mature SOCKS5 server.

**Install (Linux):**

```bash
# Debian / Ubuntu
sudo apt install dante-server -y
```

**Configure** (`/etc/danted.conf`):

```
logoutput: stderr
internal: 0.0.0.0 port = 1080
external: eth0
clientmethod: none
socksmethod: username
user.privileged: root
user.unprivileged: nobody

client pass {
    from: 0.0.0.0/0 to: 0.0.0.0/0
    log: error
}

socks pass {
    from: 0.0.0.0/0 to: 0.0.0.0/0
    command: connect
    log: error
    socksmethod: username
}
```

**Add user** (edit `/etc/pam.d/sockd` or use system auth):

```bash
sudo useradd -r -s /usr/sbin/nologin proxyuser
echo "proxyuser:yourpassword" | sudo chpasswd
```

**Start:**

```bash
sudo systemctl enable --now danted
```

**Proxy URL format:**
```
socks5://proxyuser:yourpassword@your-vps-hostname:1080
```

## Running as a Service on Linux

For VPS deployments, you can run `proxy-mcp` as a systemd service so it starts on boot and restarts on failure.

**Install binary:**

```bash
# Build or download binary to a system path
sudo cp proxy-mcp /usr/local/bin/
sudo chmod +x /usr/local/bin/proxy-mcp
```

**Create config directory:**

```bash
sudo mkdir -p /etc/proxy-mcp
sudo cp proxies.json /etc/proxy-mcp/proxies.json
```

**Create systemd service** (`/etc/systemd/system/proxy-mcp.service`):

```ini
[Unit]
Description=Proxy MCP Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/proxy-mcp -config /etc/proxy-mcp/proxies.json
Restart=on-failure
RestartSec=5
Environment=HOME=/root

[Install]
WantedBy=multi-user.target
```

**Enable and start:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now proxy-mcp
```

**Check status:**

```bash
sudo systemctl status proxy-mcp
journalctl -u proxy-mcp -f
```

## Running as a Service on macOS

**Install binary:**

```bash
# Build or copy binary
mkdir -p ~/.local/bin
cp proxy-mcp ~/.local/bin/
chmod +x ~/.local/bin/proxy-mcp
```

**Create config directory:**

```bash
mkdir -p ~/.config/proxy-mcp
cp proxies.json ~/.config/proxy-mcp/proxies.json
```

**Create launchd plist** (`~/Library/LaunchAgents/com.99apps.proxy-mcp.plist`):

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.99apps.proxy-mcp</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/proxy-mcp</string>
        <string>-config</string>
        <string>/Users/your-user/.config/proxy-mcp/proxies.json</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/proxy-mcp.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/proxy-mcp.err</string>
</dict>
</plist>
```

**Load and start:**

```bash
launchctl load ~/Library/LaunchAgents/com.99apps.proxy-mcp.plist
launchctl start com.99apps.proxy-mcp
```

## Notes

- Requests are pinned to a single proxy per call. The next call rotates based on success rate and health.
- Failed proxies are not removed, but they are de-prioritized automatically.
- All core Termigo SSRF protections still apply. This MCP only chooses the egress path; it does not bypass private-IP or metadata-IP filters.
- For Meta/Muse specifically, choose proxies with Indonesian egress IPs if Meta blocks datacenter ranges.

## Editor and Agent Compatibility

This MCP server is designed to work with any MCP client, including:

- **Termigo**
- **Termixgo**
- **VS Code** (with MCP extension)
- **Claude Code** (Anthropic's CLI)
- **Codex** (OpenAI's coding agent)
- **OpenCode**
- **OpenClaw**
- **Hermes**
- **9router**
- Any other editor or agent that supports the Model Context Protocol (MCP) over stdio

### Termigo

Add to `~/.termigo/mcp.json`:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

Replace `<PROXY_MCP_DIR>` with the directory containing the `proxy-mcp` binary and `<CONFIG_DIR>` with the path to your `proxies.json`.

### Termixgo

Add to `~/.termixgo/config.json`:

```json
{
  "mcpServers": [
    {
      "name": "proxy",
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  ]
}
```

Run `/mcp reload` in a session to start the server. Tools appear as `mcp_proxy__proxy_request`, `mcp_proxy__proxy_health`, `mcp_proxy__proxy_list`.

### VS Code

Add to `.vscode/mcp.json` in your workspace (requires MCP extension):

```json
{
  "servers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

Or add to user settings (`settings.json`):

```json
{
  "mcp.servers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

### Claude Code

Claude Code reads `mcpServers` from `~/.claude.json` (NOT `settings.json`).

**Manual edit:**

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

**Via 9router Dashboard:**

1. Open 9router Dashboard
2. Go to Claude Code CLI Tools
3. Use the "Add Custom MCP" feature

### Codex CLI

Codex CLI can use MCP servers via environment variables or config file.

**Environment variable:**

```bash
export OPENAI_MCP_SERVERS='{"proxy":{"command":"<PROXY_MCP_DIR>/proxy-mcp","args":["-config","<CONFIG_DIR>/proxies.json"]}}'
```

**Or add to Codex config file** (location varies by version, check Codex docs).

### Cursor

**Via Settings UI:**

1. Open Cursor Settings
2. Go to Features > MCP
3. Click "Add MCP Server"
4. Fill in:
   - Name: `proxy`
   - Command: `<PROXY_MCP_DIR>/proxy-mcp`
   - Args: `-config <CONFIG_DIR>/proxies.json`

**Or edit `~/.cursor/settings.json`:**

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

### OpenCode

Add to `.opencode/mcp.json` in your project root:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

### OpenClaw

Add to `~/.openclaw/openclaw.json`:

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

### Hermes

Add to Hermes config file (check Hermes docs for exact path, typically `~/.hermes/config.json` or project `.hermes/config.json`):

```json
{
  "mcpServers": {
    "proxy": {
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"]
    }
  }
}
```

### 9router

9router has a built-in MCP marketplace and supports custom MCP servers.

**Via Dashboard UI:**

1. Open 9router Dashboard (default: `http://localhost:20128`)
2. Go to MCP section
3. Click "Browse MCP Marketplace" or "Add Custom MCP"
4. Add a new stdio MCP server:
   - Name: `proxy`
   - Command: `<PROXY_MCP_DIR>/proxy-mcp`
   - Args: `-config <CONFIG_DIR>/proxies.json`

**Or edit config directly:**

9router supports both managed HTTP MCP servers and local stdio MCP servers. For `proxy-mcp`, use the local stdio format:

```json
{
  "managedMcpServers": [],
  "localStdioPlugins": [
    {
      "name": "proxy",
      "command": "<PROXY_MCP_DIR>/proxy-mcp",
      "args": ["-config", "<CONFIG_DIR>/proxies.json"],
      "toolNames": ["proxy_request", "proxy_health", "proxy_list"]
    }
  ]
}
```

After adding, restart 9router or reload the MCP servers.

## Path Placeholders

Replace these placeholders in the examples above:

| Placeholder | Linux | macOS | Windows |
|-------------|-------|-------|---------|
| `<PROXY_MCP_DIR>` | `/usr/local/bin` or `$HOME/.local/bin` | `/usr/local/bin` or `$HOME/.local/bin` | `C:\Users\<USER>\bin` or `C:\Program Files\proxy-mcp` |
| `<CONFIG_DIR>` | `/etc/proxy-mcp` or `$HOME/.config/proxy-mcp` | `$HOME/.config/proxy-mcp` | `C:\Users\<USER>\.config\proxy-mcp` |
| `<USER>` | your Linux username | your macOS username | your Windows username |

## Verification

After adding the MCP server, verify it works:

```bash
# List tools provided by the proxy MCP server
termixgo mcp
# or
~/.termigo/mcp.json
# or use the /mcp command in your editor
```

You should see `proxy_request`, `proxy_health`, and `proxy_list` in the tool list.

## Troubleshooting

- **"command not found"**: Ensure `proxy-mcp` is in your PATH or use the full absolute path.
- **"permission denied"**: Make the binary executable: `chmod +x proxy-mcp` (Linux/macOS).
- **"config file not found"**: Use absolute paths in the `-config` argument.
- **Proxy connection failed**: Check that the proxy server is reachable and credentials are correct. Test with `curl -x http://user:pass@host:port https://api.meta.com/v1/...`.
- **Firewall blocking**: Ensure outbound connections to the proxy port are allowed.

## Notes

- Requests are pinned to a single proxy per call. The next call rotates based on success rate and health.
- Failed proxies are not removed, but they are de-prioritized automatically.
- All core Termigo SSRF protections still apply. This MCP only chooses the egress path; it does not bypass private-IP or metadata-IP filters.
- For Meta/Muse specifically, choose proxies with Indonesian egress IPs if Meta blocks datacenter ranges.
