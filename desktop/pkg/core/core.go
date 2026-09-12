package core

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"OpenFlux/pkg/config"
	"OpenFlux/pkg/sysproxy"
	"OpenFlux/pkg/wintun"

	"universal-bypass-tool/socks5"
	"universal-bypass-tool/transport"
	"universal-bypass-tool/transport/oneme"
	"universal-bypass-tool/transport/yandex"
	"universal-bypass-tool/tunnel"
	"universal-bypass-tool/utils"
)

type ConnectionStatus struct {
	Connected      bool    `json:"connected"`
	Mode           string  `json:"mode"` // "wintun", "sysproxy", "socks"
	Transport      string  `json:"transport"` // "yandex", "vyandex", "oneme"
	Uptime         string  `json:"uptime"`
	UploadSpeed    string  `json:"upload_speed"`
	DownloadSpeed  string  `json:"download_speed"`
	TotalUpload    string  `json:"total_upload"`
	TotalDownload  string  `json:"total_download"`
	PingMs         int     `json:"ping_ms"`
	RecentLogs     string  `json:"recent_logs"`
	CurrentDocURL  string  `json:"current_doc_url"`
}

type CoreManager struct {
	mu            sync.Mutex
	logMu         sync.Mutex
	running       bool
	mode          string
	transport     string
	trans         transport.Transport
	socksServer   *socks5.SOCKS5Server
	startTime     time.Time
	recentLogs    []string
	maxLogLines   int
	lastSentBytes uint64
	lastRecvBytes uint64
	lastStatsTime time.Time
	lastUpSpeed   string
	lastDownSpeed string
	lastPingMs    int
	currentPort   int
	docURLs       string
	stopPing      chan struct{}
}

var globalCore = &CoreManager{
	maxLogLines: 300,
	lastPingMs:  -1,
}

func init() {
	utils.SetLogCallback(func(msg string) {
		globalCore.addLog(msg)
	})
}

func Get() *CoreManager {
	return globalCore
}

func (c *CoreManager) addLog(msg string) {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", timestamp, msg)
	c.recentLogs = append(c.recentLogs, line)
	if len(c.recentLogs) > c.maxLogLines {
		c.recentLogs = c.recentLogs[len(c.recentLogs)-c.maxLogLines:]
	}
}

func (c *CoreManager) GetLogs() string {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	return strings.Join(c.recentLogs, "\n")
}

func (c *CoreManager) Start(cfg config.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("already connected")
	}

	if cfg.Debug {
		utils.EnableDebug()
	}

	docURL := strings.TrimSpace(cfg.DocURLs)
	if cfg.Transport != "oneme" && docURL == "" {
		return fmt.Errorf("URL Яндекс.Документа не указан")
	}

	if cfg.Mode == "wintun" && !wintun.IsElevated() {
		return fmt.Errorf("для режима Wintun (полный системный VPN) требуются права администратора Windows. Запустите OpenFlux от имени администратора, либо переключитесь на режим 'Системный прокси'")
	}

	utils.Log("[Core] Starting OpenFlux (Mode: %s, Port: %d, Transport: %s)...", cfg.Mode, cfg.SocksPort, cfg.Transport)

	transConfig := transport.DefaultConfig()
	var rawTrans transport.Transport

	switch cfg.Transport {
	case "vyandex":
		utils.Log("[Core] Using Volga Yandex Transport (vyandex)...")
		rawTrans = yandex.NewYandexVolgaTransport(docURL, transConfig)
	case "oneme":
		token := strings.TrimSpace(cfg.MaxToken)
		if token == "" {
			return fmt.Errorf("MAX Web Token не указан в настройках")
		}
		uidint, _ := strconv.ParseInt(strings.TrimSpace(cfg.MaxUid), 10, 64)
		utils.Log("[Core] Using MAX Messenger Transport (oneme)...")
		rawTrans = oneme.NewOneMeTransport(false, token, uidint, transConfig)
	default:
		if strings.Contains(docURL, "volga.yandex") {
			utils.Log("[Core] Auto-detected Volga Yandex Transport (vyandex)...")
			rawTrans = yandex.NewYandexVolgaTransport(docURL, transConfig)
		} else {
			rawTrans = yandex.NewYandexDocsTransport(docURL, transConfig)
		}
	}

	effectiveKey := strings.TrimSpace(cfg.SecretKey)
	if effectiveKey == "none" || effectiveKey == "off" {
		effectiveKey = ""
	}

	encTrans, err := transport.NewEncryptedTransport(rawTrans, effectiveKey)
	if err != nil {
		utils.Log("[Core] Encryption init error: %v", err)
		return err
	}
	trans := transport.NewCompressedTransport(encTrans)

	if err := trans.Start(); err != nil {
		utils.Log("[Core] Transport start error: %v", err)
		return err
	}

	tun := tunnel.NewTCPTunnel(trans, false)
	socksAddr := fmt.Sprintf("127.0.0.1:%d", cfg.SocksPort)
	srv := socks5.NewSOCKS5Server(socksAddr, tun)

	go func() {
		if err := srv.Start(); err != nil {
			utils.Log("[Core] SOCKS5 server stopped: %v", err)
		}
	}()

	c.trans = trans
	c.socksServer = srv
	c.running = true
	c.mode = cfg.Mode
	c.transport = cfg.Transport
	if c.transport == "" {
		c.transport = "yandex"
	}
	c.startTime = time.Now()
	c.lastStatsTime = time.Now()
	c.lastSentBytes = 0
	c.lastRecvBytes = 0
	c.currentPort = cfg.SocksPort
	c.docURLs = docURL
	c.lastUpSpeed = "0 KB/s"
	c.lastDownSpeed = "0 KB/s"
	c.lastPingMs = -1

	// Apply mode-specific network integration
	switch cfg.Mode {
	case "wintun":
		utils.Log("[Core] Initializing Wintun Virtual VPN adapter...")
		if err := wintun.GetManager().Start(cfg.SocksPort, docURL); err != nil {
			utils.Log("[Core] Wintun start failed: %v", err)
			_ = c.stopLocked()
			return err
		}
	case "sysproxy":
		utils.Log("[Core] Setting Windows System Proxy...")
		if err := sysproxy.Enable(socksAddr, cfg.Bypass); err != nil {
			utils.Log("[Core] Failed to enable system proxy: %v", err)
			_ = c.stopLocked()
			return err
		}
	case "socks":
		utils.Log("[Core] Running in SOCKS5 only mode on %s", socksAddr)
	}

	// Start periodic end-to-end ping loop
	c.stopPing = make(chan struct{})
	go c.pingLoop(c.stopPing)

	return nil
}

func (c *CoreManager) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopLocked()
}

func (c *CoreManager) stopLocked() error {
	if !c.running {
		return nil
	}

	utils.Log("[Core] Disconnecting...")

	if c.stopPing != nil {
		close(c.stopPing)
		c.stopPing = nil
	}

	if c.mode == "wintun" {
		_ = wintun.GetManager().Stop()
	} else if c.mode == "sysproxy" {
		_ = sysproxy.Disable()
	}

	if c.socksServer != nil {
		c.socksServer.Stop()
		c.socksServer = nil
	}

	if c.trans != nil {
		c.trans.Stop()
		c.trans = nil
	}

	c.running = false
	c.mode = ""
	c.transport = ""
	c.lastPingMs = -1
	utils.Log("[Core] Disconnected successfully")
	return nil
}

func (c *CoreManager) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *CoreManager) pingLoop(stopChan <-chan struct{}) {
	// First ping after 1 second
	select {
	case <-time.After(1 * time.Second):
		c.executePing()
	case <-stopChan:
		return
	}

	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.executePing()
		case <-stopChan:
			return
		}
	}
}

func (c *CoreManager) executePing() {
	c.mu.Lock()
	port := c.currentPort
	running := c.running
	c.mu.Unlock()

	if !running || port <= 0 {
		return
	}

	targets := []struct {
		ip   []byte
		port uint16
	}{
		{[]byte{1, 1, 1, 1}, 80},
		{[]byte{1, 0, 0, 1}, 80},
		{[]byte{8, 8, 8, 8}, 53},
	}

	for _, tgt := range targets {
		rtt, err := c.tryPingTarget(port, tgt.ip, tgt.port)
		if err == nil && rtt > 0 {
			c.mu.Lock()
			c.lastPingMs = rtt
			c.mu.Unlock()
			return
		}
	}

	c.mu.Lock()
	c.lastPingMs = -1
	c.mu.Unlock()
}

func (c *CoreManager) tryPingTarget(socksPort int, targetIP []byte, targetPort uint16) (int, error) {
	d := net.Dialer{Timeout: 2500 * time.Millisecond}
	conn, err := d.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort))
	if err != nil {
		return -1, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

	// SOCKS5 Greeting: 0x05, 0x01 (1 method), 0x00 (no auth)
	if _, err := conn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		return -1, err
	}
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil || buf[0] != 0x05 || buf[1] != 0x00 {
		return -1, fmt.Errorf("invalid auth reply")
	}

	// SOCKS5 CONNECT request
	req := []byte{
		0x05, 0x01, 0x00, 0x01,
		targetIP[0], targetIP[1], targetIP[2], targetIP[3],
		byte(targetPort >> 8), byte(targetPort & 0xFF),
	}

	start := time.Now()
	if _, err := conn.Write(req); err != nil {
		return -1, err
	}

	resp := make([]byte, 10)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return -1, err
	}
	if resp[0] != 0x05 || resp[1] != 0x00 {
		return -1, fmt.Errorf("connect failed with status %d", resp[1])
	}

	rtt := int(time.Since(start).Milliseconds())
	return rtt, nil
}

func (c *CoreManager) GetStatus() ConnectionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.trans == nil {
		return ConnectionStatus{
			Connected:     false,
			Mode:          "",
			Transport:     "",
			Uptime:        "00:00:00",
			UploadSpeed:   "0 KB/s",
			DownloadSpeed: "0 KB/s",
			TotalUpload:   "0 B",
			TotalDownload: "0 B",
			PingMs:        -1,
			RecentLogs:    c.GetLogs(),
			CurrentDocURL: "",
		}
	}

	uptimeDur := time.Since(c.startTime).Truncate(time.Second)
	uptime := fmt.Sprintf("%02d:%02d:%02d", int(uptimeDur.Hours()), int(uptimeDur.Minutes())%60, int(uptimeDur.Seconds())%60)

	st := c.trans.Stats()
	now := time.Now()

	var upSpeed, downSpeed float64
	if !c.lastStatsTime.IsZero() {
		dur := now.Sub(c.lastStatsTime).Seconds()
		if dur > 0.4 {
			if st.BytesSent >= c.lastSentBytes {
				upSpeed = float64(st.BytesSent-c.lastSentBytes) / dur
			}
			if st.BytesReceived >= c.lastRecvBytes {
				downSpeed = float64(st.BytesReceived-c.lastRecvBytes) / dur
			}
			c.lastSentBytes = st.BytesSent
			c.lastRecvBytes = st.BytesReceived
			c.lastStatsTime = now
			c.lastUpSpeed = formatSpeed(upSpeed)
			c.lastDownSpeed = formatSpeed(downSpeed)
		}
	}

	return ConnectionStatus{
		Connected:     true,
		Mode:          c.mode,
		Transport:     c.transport,
		Uptime:        uptime,
		UploadSpeed:   c.lastUpSpeed,
		DownloadSpeed: c.lastDownSpeed,
		TotalUpload:   formatBytes(st.BytesSent),
		TotalDownload: formatBytes(st.BytesReceived),
		PingMs:        c.lastPingMs,
		RecentLogs:    c.GetLogs(),
		CurrentDocURL: c.docURLs,
	}
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.2f MB/s", bytesPerSec/(1024*1024))
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
