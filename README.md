# OpenFlux — Network Tunnel, Windows Desktop GUI & Android Client

**English** | [Русский](README.ru.md)

OpenFlux is a high-performance network stack research tool: a TCP/UDP tunnel featuring pluggable transports and end-to-end (E2E) encryption.

Fork of the original repository: [p1neappleXpress/OpenFlux](https://github.com/p1neappleXpress/OpenFlux).

> [!NOTE]
> **Disclaimer**: This project is a non-commercial educational and research tool provided "as is" solely for studying network protocols and virtual network interfaces. The authors bear no responsibility for how users deploy or utilize this software.

---

## Key Features

- **Pluggable Transports (`-transport`):**
  - `yandex` — collaborative Yandex Docs editing session (Socket.IO v4 over WebSocket).
  - `vyandex` — high-speed streaming HTTP transport over Yandex Volga.
  - `cupsonline` — collaborative sessions over Cups.online (WebSocket).
  - `oneme` — traffic transport over MAX messenger signaling & WebRTC DataChannel.
- **End-to-End AEAD Encryption (Zero-Knowledge):**
  - AES-256-GCM with scrypt key derivation. Cloud intermediaries cannot inspect headers, URLs, SNI, or DNS.
  - Memory protection and packet sizing controls (10 MB LZ4 frame ceiling).
- **Android Client (`android/`):**
  - Modern Material 3 design (Light & Dark themes).
  - Built-in lightweight QR scanner.
  - Dual operation modes: Full **VPN** (`VpnService` + `tun2socks`/gVisor) or local **SOCKS5 proxy**.
  - Per-App Split Tunneling (whitelist & blacklist).
  - Real-time speed monitor, ping, uptime, and traffic statistics.
  - Quick Settings Tile for instant toggle from the notification shade.
- **Windows Desktop GUI (`desktop/`):**
  - Clean interface built with Wails v2 + Vue 3.
  - Modes: Wintun (full system network adapter), System Proxy (SysProxy), local SOCKS5, and integrated Exit Node.
- **Linux Exit Node Server:**
  - Dual routing modes: **Proxy** (userspace `net.Dial`, no root required) or **Raw Sockets** (kernel-level packet routing with stateful port tracking).
  - Ready-to-use `docker-compose.yml` and isolated `network namespace` scripts (preventing unwanted kernel RST packets).
  - Systemd watchdog support (`WatchdogSec=30s`).

---

## Screenshots

### Windows Desktop (GUI)
<p align="center">
  <img src="docs/screenshots/desktop.jpg" width="85%" alt="OpenFlux Windows Desktop GUI" />
</p>

### Android Client
<table width="100%">
  <tr>
    <th width="25%" align="center">Dashboard</th>
    <th width="25%" align="center">Settings</th>
    <th width="25%" align="center">Split Tunneling</th>
    <th width="25%" align="center">Event Logs</th>
  </tr>
  <tr>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_main.jpg" width="100%" alt="Dashboard" /></td>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_settings.jpg" width="100%" alt="Settings" /></td>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_split_tunnel.jpg" width="100%" alt="Split Tunneling" /></td>
    <td width="25%" align="center" valign="top"><img src="docs/screenshots/android_logs.jpg" width="100%" alt="Event Logs" /></td>
  </tr>
</table>

---

## Quick Start

### 1. Exit Node Server (Linux VPS)

The exit node makes an outbound connection to the cloud platform. No open inbound ports are needed in your firewall — it runs seamlessly behind NAT.

#### Option A: Docker Compose (Recommended)

1. Clone the repository and configure `.env`:
   ```bash
   git clone https://github.com/tatarinovs/OpenFlux.git
   cd OpenFlux
   cp deploy/openflux.env.example .env
   nano .env
   ```
2. Fill in your document URL and encryption passphrase:
   ```env
   OPENFLUX_URL=https://disk.yandex.ru/i/YOUR_DOCUMENT
   OPENFLUX_KEY=your_passphrase_or_hex_key
   OPENFLUX_TRANSPORT=yandex
   ```
3. Start the container:
   ```bash
   docker compose up -d --build
   docker compose logs -f
   ```

#### Option B: Direct CLI Execution

```bash
# Proxy mode (unprivileged, no root required):
./openflux -exit-node -mode proxy -transport yandex -url "https://disk.yandex.ru/i/..." -key "your_passphrase"

# Raw Sockets mode (high performance, requires root):
sudo iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP
sudo ./openflux -exit-node -mode raw -transport yandex -url "https://disk.yandex.ru/i/..." -key "your_passphrase"
```

---

### 2. Connecting Clients

1. **Android**:
   - Install the APK from [releases/](releases/) or GitHub Releases.
   - Enter the document URL (or scan the config QR code via Settings).
   - Enter the encryption passphrase (must match `OPENFLUX_KEY` on the server).
   - Choose your mode (VPN or Proxy) and tap Connect.
2. **Windows Desktop**:
   - Run `OpenFlux.exe` from the [releases/](releases/) directory.
   - Enter transport, document URL, and encryption key.
   - Choose mode (Wintun VPN, System Proxy, or SOCKS5) and connect.
3. **CLI Client (Cross-platform)**:
   ```bash
   ./openflux -client -transport yandex -url "https://disk.yandex.ru/i/..." -key "your_passphrase" -socks5 127.0.0.1:1080
   ```

---

## Command Line Flags

| Flag | Default | Description |
|---|---|---|
| `-client` | `false` | Run in client mode (starts local SOCKS5 proxy) |
| `-exit-node` | `false` | Run in exit node mode |
| `-mode` | `proxy` | Exit node routing mode: `proxy` (userspace, rootless) or `raw` |
| `-transport` | `yandex` | Transport protocol: `yandex`, `vyandex`, `cupsonline`, `oneme` |
| `-url` | `http://#` | Public URL for document or room |
| `-key` | `""` | End-to-End encryption passphrase (or `OPENFLUX_KEY` env) |
| `-socks5` | `:1080` | Local SOCKS5 listening address |
| `-ycookie` | `""` | Optional Yandex session cookies for session verification |
| `-maxToken` | `""` | MAX auth token (for `oneme` transport) |
| `-maxUid` | `""` | MAX call user ID (for `oneme` transport) |
| `-debug` | `false` | Enable verbose debug logging |

---

## Building from Source

- **Build all platforms with 1 click (Windows):**
  ```cmd
  build_all.bat
  ```
- **Android Release APK:**
  ```cmd
  build_release.bat
  ```
- **Windows Desktop (Wails):**
  ```cmd
  build_windows.bat
  ```
- **Linux Exit Node / CLI:**
  ```bash
  go build -trimpath -ldflags="-s -w" -o openflux .
  ```
