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

**Configure Squid** (edit `/etc/squid/squid.conf` or `/usr/local/etc/squid/squid.conf` on macOS):

```
auth_param basic program /usr/lib/squid/basic_ncsa_auth /etc/squid/passwd
auth_param basic realm Proxy Authentication
acl authenticated proxy_auth REQUIRED
http_access allow authenticated
http_port 8080
```

**Start Squid:**

```bash
# Linux (systemd)
sudo systemctl enable --now squid

# macOS (launchd)
sudo brew services start squid
```

**Firewall:**

```bash
# Linux (ufw)
sudo ufw allow 8080/tcp

# Linux (iptables)
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# macOS
echo "allow 8080" | sudo pfctl -f -
```

**Proxy URL format:**

```
http://youruser:yourpassword@your-vps-hostname:8080
```

---

### Option B: 3proxy (HTTP + SOCKS5, lightweight)

3proxy is lightweight and supports both HTTP and SOCKS5 on the same port or different ports.

**Install (Linux):**

```bash
# Debian / Ubuntu
sudo apt install 3proxy -y

# Or compile from source
git clone https://github.com/z3APA3A/3proxy.git
cd 3proxy
make -f Makefile.Linux
sudo cp 3proxy /usr/local/bin/
```

**Configure** (`/etc/3proxy/3proxy.cfg` or `~/3proxy.cfg`):

```
# Define users: username:CL:password
users youruser:CL:yourpassword

# HTTP proxy on 8080
proxy -p8080 -i0.0.0.0 -e0.0.0.0

# SOCKS5 proxy on 9050
socks -p9050 -i0.0.0.0 -e0.0.0.0

# Optional: require auth for SOCKS5
socks -p9050 -i0.0.0.0 -e0.0.0.0 -u youruser:CL:yourpassword
```

**Start 3proxy:**

```bash
# Linux (systemd)
sudo systemctl enable --now 3proxy

# Or run directly
sudo 3proxy /etc/3proxy/3proxy.cfg
```

**Proxy URL formats:**

```
http://youruser:yourpassword@your-vps-hostname:8080
socks5://youruser:yourpassword@your-vps-hostname:9050
```

---

### Option C: Dante (SOCKS5 only)

Dante is a mature SOCKS5 server.

**Install (Linux):**

```bash
sudo apt install dante-server -y
```

**Configure** (`/etc/danted.conf`):

```
# Listen on all interfaces
internal: 0.0.0.0 port = 9050
external: 0.0.0.0

# Authentication
method: username
user.privileged: root
user.notprivileged: nobody

# ACLs
client pass {
    from: 0.0.0.0/0 to: 0.0.0.0/0
    log: error
}

# Allow authenticated users
client pass {
    from: 0.0.0.0/0 to: 0.0.0./0
    method: username
}

# Pass all traffic
pass {
    from: 0.0.0.0/0 to: 0.0.0.0/0
    method: username
}

# Logging
logoutput: stderr
```

**Start Dante:**

```bash
sudo systemctl enable --now danted
```

**Proxy URL format:**

```
socks5://youruser:yourpassword@your-vps-hostname:9050
```

---

### Getting the Proxy URL

After setting up your proxy server, construct the URL:

```
protocol://username:password@host:port
```

| Component | Example | Description |
|-----------|---------|-------------|
| `protocol` | `http` or `socks5` | Depends on your proxy software |
| `username` | `youruser` | The username you configured |
| `password` | `yourpassword` | The password you configured |
| `host` | `your-vps-hostname` | IP address or domain name of your VPS |
| `port` | `8080` or `9050` | Port your proxy listens on |

**Examples:**

```json
[
  {
    "url": "http://youruser:yourpassword@your-vps-hostname:8080",
    "country": "ID",
    "weight": 10,
    "timeout": "15s"
  },
  {
    "url": "socks5://youruser:yourpassword@your-vps-hostname:9050",
    "country": "ID",
    "weight": 5,
    "timeout": "30s"
  }
]
```

---

### Testing Your Proxy

Test from your local machine before adding to `proxies.json`:

```bash
# Linux / macOS
curl -x http://youruser:yourpassword@your-vps-hostname:8080 https://api.meta.com/v1/models

# Windows PowerShell
curl -x http://youruser:yourpassword@your-vps-hostname:8080 https://api.meta.com/v1/models
```

If you get a response from Meta, the proxy is working.

---

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

**Start service:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now proxy-mcp
sudo systemctl status proxy-mcp
```

**View logs:**

```bash
journalctl -u proxy-mcp -f
```

**macOS (launchd):**

Create `~/Library/LaunchAgents/com.99apps.proxymcp.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.99apps.proxymcp</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/proxy-mcp</string>
        <string>-config</string>
        <string>/Users/youruser/.config/proxy-mcp/proxies.json</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load it:

```bash
launchctl load ~/Library/LaunchAgents/com.99apps.proxymcp.plist
```

## Notes

- Requests are pinned to a single proxy per call. The next call rotates based on success rate and health.
- Failed proxies are not removed, but they are de-prioritized automatically.
- All core Termigo SSRF protections still apply. This MCP only chooses the egress path; it does not bypass private-IP or metadata-IP filters.
- For Meta/Muse specifically, choose proxies with Indonesian egress IPs if Meta blocks datacenter ranges.
