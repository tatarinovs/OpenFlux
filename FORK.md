# OpenFlux Fork Differences & Enhancements

This repository is an official evolution and feature-complete fork of **[p1neappleXpress/OpenFlux](https://github.com/p1neappleXpress/OpenFlux)**.

---

## Comparison Summary

| Feature / Component | Upstream OpenFlux (`p1neappleXpress`) | OpenFlux Fork (`tatarinovs`) |
| :--- | :--- | :--- |
| **End-to-End Encryption** | ❌ None (plaintext IP packets exposed to Yandex/OnlyOffice) | ✅ **ChaCha20-Poly1305 AEAD** (256-bit key, individual 12-byte nonce per packet, 16-byte MAC) |
| **Android Client** | ❌ None (CLI utility only) | ✅ **Native Android App** with Material 3 UI, Dark/Light themes, Quick Settings Tile |
| **Android Operating Modes** | ❌ None | ✅ **Full VPN Mode** (gVisor `tun2socks`) and **Proxy Mode** (local SOCKS5 `127.0.0.1:1080`) |
| **Split Tunneling** | ❌ None | ✅ **Per-App Routing** (Whitelist & Blacklist to route only selected apps) |
| **Document Failover Pool** | ❌ Single URL (tunnel breaks on errors or captchas) | ✅ **Multi-URL Pool** with auto-rotation and instant seamless failover |
| **Docker Containerization** | ❌ None | ✅ **Multi-stage Dockerfile & Docker Compose** with `NET_ADMIN` / `NET_RAW` capabilities |
| **Server Network Isolation** | ❌ Global host `DROP RST` rule (breaks co-existing services) | ✅ **Linux Network Namespace (`openflux`)** isolation with zero host interference |
| **Health Monitoring** | ❌ None | ✅ **Systemd Watchdog (`sd_notify`)**: heartbeat signals every 15s with auto-restart |
| **DNS-over-TCP** | ❌ Basic handling | ✅ **RFC 1035 TCP tunneling** of DNS queries (eliminates UDP 53 leaks & throttling) |
| **Decompression Security** | ⚠️ Unbounded LZ4 allocation | ✅ **Decompression Bomb Protection**: strict buffer limits and header validation |
| **Connection Stability** | ⚠️ Dual-stack (IPv6) connection hangs | ✅ Forced `tcp4`, race-free reconnection generations (`reconnectGen`), desktop User-Agent |

---

## Key Enhancements in Detail

### 1. Zero-Knowledge End-to-End Encryption (ChaCha20-Poly1305)
- **Upstream Limitation:** Raw IP frames were encapsulated in plaintext inside OnlyOffice binary messages. The cloud provider could perform Deep Packet Inspection (DPI), log target domains via TLS SNI, and inspect unencrypted payloads.
- **Enhancement:** Implemented in `transport/encrypted.go`. All payloads undergo authenticated encryption with associated data (AEAD) using **ChaCha20-Poly1305**. Every packet receives a fresh random 12-byte Nonce and a 16-byte Poly1305 authentication tag. Cloud intermediaries observe only cryptographically random bytes.

### 2. Full-Featured Android Client (`android/`)
- Built from the ground up in Kotlin with Jetpack libraries and Material 3 design.
- **Full VPN Mode:** Uses Android `VpnService` to capture system-wide traffic into a virtual `tun0` interface, routed via the Go core gVisor `tun2socks` stack.
- **Proxy Mode:** Runs a lightweight background daemon hosting a local SOCKS5 proxy (`127.0.0.1:1080`) for browser-only or custom proxy setups.
- **Split Tunneling:** Route only specific applications through the tunnel or exclude bandwidth-heavy apps (games, video streaming).
- **Quick Settings Tile:** Toggle the tunnel directly from the Android system notification panel.
- **Real-Time Monitoring:** Live ping measurement, upload/download speed counters, and an in-app log viewer (`LogActivity`).
- **Release Optimization:** Automated 1-click build pipeline with R8 code minification, resource shrinking, and APK Signature Scheme v2 (~17 MB).

### 3. Multi-URL Pool with Automatic Failover
- Supports comma-separated or multiline document URLs in Android settings and server `.env`.
- Thread-safe round-robin selection (`atomic.Int32`) with automatic instant failover when encountering anti-bot captchas (`showcaptcha`), HTTP 404 errors, or WebSocket connection drops.

### 4. Server Architecture: Docker Compose & Network Namespace
- **Docker Compose:** Run the exit node anywhere in 1 command without installing the Go compiler on the host VPS.
- **Linux Network Namespace Isolation (`deploy/ofx-netns.sh`):** Isolate the exit node into a dedicated `openflux` netns. The required `iptables DROP RST` rule applies strictly inside this namespace and never interferes with host SSH, Nginx, or co-hosted VPN protocols (Xray, Sing-box, Outline, WireGuard).
- **Systemd Watchdog:** Native Go `sd_notify` socket client (`utils/watchdog.go`). Sends `WATCHDOG=1` heartbeats every 15 seconds; systemd automatically recovers the service if any freeze occurs.

### 5. Robustness & DoS Hardening
- **Decompression Bomb Protection:** LZ4 decompression validates uncompressed length headers (64 KB ceiling per frame), preventing memory exhaustion attacks.
- **Socket Leak & Race Guard:** `reconnectGen` generation tracking ensures stale asynchronous callbacks cannot corrupt newly established sessions.
- **Forced IPv4 (`tcp4`):** Eliminates multi-second connection timeouts on dual-stack hosts with misconfigured IPv6 routing.
