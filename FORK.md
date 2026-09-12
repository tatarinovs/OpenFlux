# OpenFlux Fork Differences & Enhancements

This repository is an official evolution and feature-complete fork of **[p1neappleXpress/OpenFlux](https://github.com/p1neappleXpress/OpenFlux)**.

# Disclaimer

The project authors **do not encourage** the use of this tool to violate network policies or platform rules, and **are not responsible** for third-party actions. Development is conducted strictly for educational and scientific research purposes in computer networking. Code is provided "as is", without any warranties.

---

## Key Enhancements in Detail

### 1. Zero-Knowledge End-to-End Encryption (ChaCha20-Poly1305)
- **Data Privacy in Original:** Raw IP frames were encapsulated in plaintext inside OnlyOffice binary messages. The platform servers could inspect unencrypted headers, domain names via TLS SNI, and plaintext DNS queries.
- **Enhancement:** Implemented in `transport/encrypted.go`. All payloads undergo authenticated encryption with associated data (AEAD) using **ChaCha20-Poly1305**. Every packet receives a fresh random 12-byte Nonce and a 16-byte Poly1305 authentication tag. Intermediary servers observe only cryptographically random bytes.

### 2. Full-Featured Android Client (`android/`)
- Built from the ground up in Kotlin with Jetpack libraries and Material 3 design.
- **Full VPN Mode:** Uses Android `VpnService` to capture system-wide traffic into a virtual `tun0` interface, routed via the Go core gVisor `tun2socks` stack.
- **Proxy Mode:** Runs a lightweight background daemon hosting a local SOCKS5 proxy (`127.0.0.1:1080`) for browser-only or custom proxy setups.
- **Split Tunneling:** Route only specific applications through the tunnel or exclude bandwidth-heavy apps (games, video streaming).
- **Quick Settings Tile:** Toggle the tunnel directly from the Android system notification panel.
- **Real-Time Monitoring:** Live ping measurement, upload/download speed counters, and an in-app log viewer (`LogActivity`).

### 3. Multi-URL Pool with Automatic Failover
- Supports comma-separated or multiline document URLs in Android settings, Windows client, and server `.env`.
- Thread-safe round-robin selection (`atomic.Int32`) with automatic instant failover when encountering anti-bot captchas (`showcaptcha`), HTTP 404 errors, or WebSocket connection drops.

### 4. Server Architecture: Docker Compose & Network Namespace
- **Docker Compose:** Run the exit node anywhere in 1 command without installing the Go compiler on the host VPS.
- **Linux Network Namespace Isolation (`deploy/ofx-netns.sh`):** Isolate the exit node into a dedicated `openflux` netns. The required `iptables DROP RST` rule applies strictly inside this namespace and never interferes with host SSH, Nginx, or co-hosted VPN protocols (Xray, Sing-box, Outline, WireGuard).
- **Systemd Watchdog:** Native Go `sd_notify` socket client (`utils/watchdog.go`). Sends `WATCHDOG=1` heartbeats every 15 seconds; systemd automatically recovers the service if any freeze occurs.

### 5. Robustness & DoS Hardening
- **Decompression Bomb Protection:** LZ4 decompression validates uncompressed length headers (64 KB ceiling per frame), preventing memory exhaustion attacks.
- **Socket Leak & Race Guard:** `reconnectGen` generation tracking ensures stale asynchronous callbacks cannot corrupt newly established sessions.
- **Forced IPv4 (`tcp4`):** Eliminates multi-second connection timeouts on dual-stack hosts with misconfigured IPv6 routing.

### 6. Desktop Windows GUI Client (`desktop/`)
- **Technology Stack:** Wails v2 + Vue 3, compiled into a single self-contained executable `OpenFlux.exe` (~18 MB).
- **Wintun Mode (System-wide VPN):** Captures all Windows network traffic (including DNS requests, games, messengers, and browsers) via the high-performance Wintun virtual adapter.
- **System Proxy Mode:** Instantly toggles the Windows system proxy in the registry (WinINET) without requiring UAC administrator privileges.
- **System Tray Integration:** Native Windows system tray icon with connection status, quick toggle menu, and background execution when closing the window.

### 7. Version 1.0.1 Enhancements
- **Yandex Volga Transport (`vyandex`):** Volga transport implementation from upstream (PR #33).
- **MAX Messenger (`oneme`) in Windows GUI:** The desktop client now natively supports tunneling via MAX Web WebRTC DataChannels in addition to Yandex Docs and Volga.
- **Light & Dark Theme Support:** Sleek light theme alongside the dark aesthetic with one-click instant toggling.
- **Dropdown Transport Selectors:** Unified dropdown selection interface across Windows Desktop and Android clients.
- **Parallel Multi-Listen on Exit Node:** The exit node connects to all pool documents simultaneously, allowing seamless zero-downtime routing when clients fail over.
