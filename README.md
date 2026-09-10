# OpenFlux — Covert Network Tunnel & Android Client

**English** | [Русский](README.ru.md)

**OpenFlux** is an advanced network tunneling framework designed to disguise and route TCP traffic through legitimate cloud services (such as Yandex Docs WebSocket collaboration sessions and MAX WebRTC DataChannels). It includes a high-performance Go exit node, a desktop SOCKS5 client, and a native **Android client application** with full-system VPN and Per-App Split Tunneling.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Client Side                         │
│  [Android App / Browser] ──> SOCKS5 Proxy (:1080)       │
│                                  │                      │
│                  LZ4 Payload Compression                │
│                                  │                      │
│            ChaCha20-Poly1305 AEAD Encryption            │
└──────────────────────────────┬──────────────────────────┘
                               │ (Masked WebSocket / WebRTC)
                               ▼
┌─────────────────────────────────────────────────────────┐
│               Cloud Intermediary Service                │
│       (Yandex Docs Collaborative WebSocket / MAX)       │
│      * Intermediary sees only encrypted noise *         │
└──────────────────────────────┬──────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────┐
│                   Linux Exit Node                       │
│            ChaCha20-Poly1305 Decryption                 │
│                                  │                      │
│                  LZ4 Payload Decompression              │
│                                  │                      │
│        Raw Sockets Router (tunnel/rawsocket_linux.go)   │
│                                  │                      │
│                         Target Internet                 │
└─────────────────────────────────────────────────────────┘
```

---

## Key Features

- **Covert Pluggable Transports:**
  - **Yandex Docs:** Emulates collaborative editing sessions via Socket.IO v4 over WebSocket. Uses cursor coordinates and document revisions to encapsulate network packets.
  - **MAX Messenger:** WebRTC DataChannel transport using signaling APIs.
- **End-to-End ChaCha20-Poly1305 Encryption:**
  - Zero-Knowledge transport: intermediary cloud servers (Yandex, MAX) cannot inspect headers, visited URLs, SNI, or DNS queries.
  - Unique 12-byte random nonce per packet + 16-byte Poly1305 authentication tag.
  - Configurable key via GUI or `-key` flag.
- **Full-featured Android Application (`android/`):**
  - **Material 3 Design:** Sleek UI with Dark/Light theme support.
  - **Proxy Mode:** Lightweight background service exposing a local SOCKS5 proxy (`127.0.0.1:1080` or `0.0.0.0:1080`).
  - **Full VPN Mode:** Android `VpnService` capturing device-wide traffic into a virtual `tun0` interface powered by `tun2socks` (gVisor netstack).
  - **Per-App Split Tunneling:** Whitelist / Blacklist apps from being routed through the tunnel.
  - **DNS-over-TCP:** Automatically handles Android UDP DNS requests (port 53) via RFC 1035 TCP tunneling.
  - **Live Traffic Monitoring:** Real-time speed and packet counters on the main dashboard and live status in the Android Notification Shade (`↑ / ↓`).
  - **Crash Resilience:** Process-level error trapping and integrated log viewer.
- **Enterprise-grade Linux Exit Node:**
  - Fast packet routing using AF_INET RAW sockets.
  - Automatic active port tracking and cleanup.
  - Forced IPv4 (`tcp4`) resolver preventing dual-stack IPv6 stalls.
- **Security Hardened:**
  - Fully hardened against DoS vulnerabilities, memory caps, zip-bomb defense, and thread-safe signaling across all modules.

---

## Repository Structure

```
OpenFlux/
├── main.go                     # Desktop client and Exit Node CLI entry point
├── transport/
│   ├── transport.go            # Base Transport interface
│   ├── encrypted.go            # ChaCha20-Poly1305 AEAD E2E encryption wrapper
│   ├── compressor.go           # LZ4 compression with decompression bomb limits
│   ├── yandex/                 # Yandex Docs WebSocket collaborative backend
│   └── oneme/                  # MAX WebRTC DataChannel backend
├── tunnel/
│   ├── tunnel.go               # gVisor TCP/IP network stack orchestrator
│   ├── endpoint.go             # Virtual NIC link endpoint
│   └── rawsocket_linux.go      # Linux raw socket routing & port tracking
├── socks5/
│   └── socks5.go               # SOCKS5 proxy server & UDP 53 DNS-over-TCP handler
├── network/
│   └── checksum.go             # IP and TCP checksum calculations
├── mobile/
│   └── bridge.go               # C-Shared JNI bridge for Android runtime
├── android/                    # Android Studio project (Kotlin + Jetpack)
│   ├── app/src/main/           # Android application source code
│   └── openflux-release.jks    # Release signing keystore
├── releases/                   # Output directory for optimized release APKs
├── scripts/
│   └── build_android_lib.ps1   # NDK Clang cross-compiler script for Go core
├── build_release.bat           # 1-Click Windows builder for optimized release APK
└── build_apk.ps1               # PowerShell debug builder
```

---

## Getting Started

### 1. Setting Up the Exit Node (Linux VPS)

The exit node requires a Linux server with root privileges (to use raw sockets).

```bash
# 1. Clone repository & build Linux binary
go build -ldflags="-s -w" -o universal-bypass-tool .

# 2. Prevent kernel RST packets from interrupting tunnel TCP sessions
sudo iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP

# 3. Start exit node
sudo ./universal-bypass-tool -exit-node \
  -transport yandex \
  -url "https://disk.yandex.ru/i/YOUR_DOCUMENT_KEY" \
  -key "YOUR_SECRET_KEY_HEX" \
  -debug
```

#### Running as a Systemd Service

Create `/etc/systemd/system/openflux-exit.service`:
```ini
[Unit]
Description=OpenFlux Exit Node
After=network.target network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/openflux
ExecStartPre=/bin/sh -c '/sbin/iptables -C OUTPUT -p tcp --tcp-flags RST RST -j DROP 2>/dev/null || /sbin/iptables -I OUTPUT 1 -p tcp --tcp-flags RST RST -j DROP'
ExecStart=/opt/openflux/universal-bypass-tool -exit-node -transport yandex -url "https://disk.yandex.ru/i/YOUR_DOC_ID" -debug
ExecStopPost=/sbin/iptables -D OUTPUT -p tcp --tcp-flags RST RST -j DROP
Restart=always
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```
Enable and run:
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now openflux-exit.service
```

---

### 2. Building the Android Client

#### Automated 1-Click Release Build (Windows)
Run [build_release.bat](build_release.bat):
```cmd
build_release.bat
```
The script will:
1. Compile `libopenflux.so` with `-trimpath`, `-ldflags="-s -w"`, and NDK Clang `-O3`.
2. Run Android Gradle `assembleRelease` with **R8 code minification** and **Resource Shrinking**.
3. Sign the APK using **APK Signature Scheme v2**.
4. Output the ready-to-install package to `releases/OpenFlux-release.apk` (~17 MB).

#### Manual CLI Build:
```powershell
# Build Go native library for arm64-v8a
$env:GOOS = "android"; $env:GOARCH = "arm64"; $env:CGO_ENABLED = "1"
$env:CC = "C:\Users\<user>\AppData\Local\Android\Sdk\ndk\<version>\toolchains\llvm\prebuilt\windows-x86_64\bin\aarch64-linux-android24-clang.cmd"
go build -trimpath -buildmode=c-shared -ldflags="-s -w -checklinkname=0" -o android/app/src/main/jniLibs/arm64-v8a/libopenflux.so ./mobile

# Build APK
cd android
.\gradlew.bat assembleRelease
```

---

### 3. Desktop Client (CLI)

To run as a desktop client forwarding a local browser/system through the tunnel:

```bash
./universal-bypass-tool -client \
  -transport yandex \
  -url "https://disk.yandex.ru/i/YOUR_DOCUMENT_KEY" \
  -socks5 127.0.0.1:1080 \
  -key "YOUR_SECRET_KEY_HEX" \
  -debug
```
Configure your browser or system proxy to SOCKS5 `127.0.0.1:1080`.

---

## Command-Line Flags

| Flag | Default | Description |
|---|---|---|
| `-client` | `false` | Run as desktop client (starts SOCKS5 proxy) |
| `-exit-node` | `false` | Run as server exit node (requires root) |
| `-transport` | `yandex` | Transport backend: `yandex` or `oneme` |
| `-url` | `http://#` | Yandex Docs document sharing URL |
| `-socks5` | `:1080` | SOCKS5 listen address for client mode |
| `-key` | *(built-in)* | ChaCha20-Poly1305 encryption key (or `OPENFLUX_KEY` env) |
| `-maxToken` | `""` | MAX Messenger auth token |
| `-maxUid` | `""` | MAX Messenger User ID |
| `-debug` | `false` | Enable verbose diagnostic logging |

---

## Security & Privacy

1. **Zero-Knowledge Transport:** The intermediary document server stores opaque base64 cursor updates. Even with TLS termination at Yandex/MAX balancers, payloads remain securely encrypted with AEAD ChaCha20-Poly1305.
2. **Decompression Bomb Protection:** LZ4 decompression is bounded to `MaxDecompressedSize = 10 MB` per frame to prevent memory exhaustion attacks.
3. **Hardened Architecture:** Robust memory allocation limits, panic recovery across goroutines, and strict buffer boundary checks.

---

## License & Disclaimer

This software is developed for research and educational purposes regarding network protocols, covert tunneling, and resilient communication stacks. Use only on systems and networks where you have explicit authorization.
