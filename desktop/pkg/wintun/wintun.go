package wintun

import (
	_ "embed"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
	tunlog "github.com/xjasonlyu/tun2socks/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows"
	"universal-bypass-tool/utils"
)

func init() {
	logger := zap.New(zapcore.NewNopCore(), zap.WithFatalHook(zapcore.WriteThenPanic))
	tunlog.SetLogger(logger)
}

func IsElevated() bool {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()
	return token.IsElevated()
}

//go:embed embed/wintun.dll
var embeddedWintunDLL []byte

const (
	AdapterName = "OpenFlux"
	TunnelIP    = "10.0.85.2"
	TunnelMask  = "255.255.255.0"
	TunnelGW    = "10.0.85.1"
)

type WintunManager struct {
	mu           sync.Mutex
	active       bool
	origGateway  string
	bypassIPs    []string
	adapterName  string
}

var globalWintun = &WintunManager{
	adapterName: AdapterName,
}

func GetManager() *WintunManager {
	return globalWintun
}

func EnsureDLL() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)
	targetDLL := filepath.Join(exeDir, "wintun.dll")

	if _, err := os.Stat(targetDLL); err == nil {
		return nil
	}

	utils.Log("[Wintun] Extracting wintun.dll to %s", targetDLL)
	return os.WriteFile(targetDLL, embeddedWintunDLL, 0644)
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func findDefaultGateway() (string, error) {
	out, err := runCmd("route", "print", "0.0.0.0")
	if err != nil {
		return "", err
	}
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 5 && fields[0] == "0.0.0.0" && fields[1] == "0.0.0.0" {
			gw := fields[2]
			if gw != TunnelGW && !strings.HasPrefix(gw, "10.0.85.") && gw != "On-link" {
				return gw, nil
			}
		}
	}
	return "", fmt.Errorf("default gateway not found")
}

func resolveHostIPs(rawURLs string) []string {
	var ips []string
	seen := make(map[string]bool)

	urls := strings.FieldsFunc(rawURLs, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';' || r == ' '
	})

	for _, uStr := range urls {
		uStr = strings.TrimSpace(uStr)
		if uStr == "" {
			continue
		}
		if !strings.HasPrefix(uStr, "http://") && !strings.HasPrefix(uStr, "https://") && !strings.HasPrefix(uStr, "ws://") && !strings.HasPrefix(uStr, "wss://") {
			uStr = "https://" + uStr
		}
		parsed, err := url.Parse(uStr)
		if err != nil {
			continue
		}
		host := parsed.Hostname()
		if host == "" || host == "#" {
			continue
		}

		if ip := net.ParseIP(host); ip != nil {
			if !seen[host] {
				seen[host] = true
				ips = append(ips, host)
			}
			continue
		}

		addrs, err := net.LookupHost(host)
		if err == nil {
			for _, addr := range addrs {
				if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
					if !seen[addr] {
						seen[addr] = true
						ips = append(ips, addr)
					}
				}
			}
		}
	}
	return ips
}

func (w *WintunManager) Start(socksPort int, docURLs string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.active {
		return nil
	}

	if !IsElevated() {
		return fmt.Errorf("для режима Wintun (полный системный VPN) требуются права администратора Windows. Запустите OpenFlux от имени администратора, либо переключитесь на режим 'Системный прокси'")
	}

	if err := EnsureDLL(); err != nil {
		return fmt.Errorf("failed to ensure wintun.dll: %w", err)
	}

	origGW, err := findDefaultGateway()
	if err != nil {
		utils.Log("[Wintun] Warning: could not find default gateway: %v", err)
	} else {
		w.origGateway = origGW
		utils.Log("[Wintun] Original default gateway: %s", origGW)
	}

	utils.Log("[Wintun] Initializing Wintun adapter '%s' with tun2socks...", w.adapterName)

	key := &engine.Key{
		Device:   fmt.Sprintf("tun://%s", w.adapterName),
		Proxy:    fmt.Sprintf("socks5://127.0.0.1:%d", socksPort),
		MTU:      1500,
		LogLevel: "silent",
	}

	var startErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				startErr = fmt.Errorf("tun2socks initialization failed: %v", r)
			}
		}()
		engine.Insert(key)
		engine.Start()
	}()

	if startErr != nil {
		return fmt.Errorf("ошибка запуска Wintun: %w (требуются права администратора)", startErr)
	}

	// Wait for adapter creation in Windows
	time.Sleep(1 * time.Second)

	// Configure IP and Gateway on Wintun adapter
	utils.Log("[Wintun] Configuring IP %s on interface '%s'...", TunnelIP, w.adapterName)
	_, _ = runCmd("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", w.adapterName),
		"static", TunnelIP, TunnelMask, TunnelGW, "1")

	// Set DNS servers
	_, _ = runCmd("netsh", "interface", "ip", "set", "dns",
		fmt.Sprintf("name=%s", w.adapterName), "static", "1.1.1.1", "primary")
	_, _ = runCmd("netsh", "interface", "ip", "add", "dns",
		fmt.Sprintf("name=%s", w.adapterName), "8.8.8.8", "index=2")

	// Add direct host routes for Yandex / Exit Node servers so they bypass the VPN
	bypassIPs := resolveHostIPs(docURLs)
	w.bypassIPs = bypassIPs
	if w.origGateway != "" {
		for _, ip := range bypassIPs {
			utils.Log("[Wintun] Adding bypass route for server IP %s via %s", ip, w.origGateway)
			_, _ = runCmd("route", "add", ip, "mask", "255.255.255.255", w.origGateway, "metric", "1")
		}
	}

	// Add dual /1 routes covering all IPv4
	utils.Log("[Wintun] Adding global IPv4 routes via %s...", TunnelGW)
	_, err1 := runCmd("route", "add", "0.0.0.0", "mask", "128.0.0.0", TunnelGW, "metric", "5")
	_, err2 := runCmd("route", "add", "128.0.0.0", "mask", "128.0.0.0", TunnelGW, "metric", "5")

	if err1 != nil || err2 != nil {
		utils.Log("[Wintun] Warning setting routes: %v, %v (run as administrator if routing failed)", err1, err2)
	}

	w.active = true
	utils.Log("[Wintun] System VPN successfully activated")
	return nil
}

func (w *WintunManager) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.active {
		return nil
	}

	utils.Log("[Wintun] Stopping System VPN and restoring routing...")

	// Remove dual /1 routes
	_, _ = runCmd("route", "delete", "0.0.0.0", "mask", "128.0.0.0")
	_, _ = runCmd("route", "delete", "128.0.0.0", "mask", "128.0.0.0")

	// Remove bypass IP routes
	for _, ip := range w.bypassIPs {
		_, _ = runCmd("route", "delete", ip)
	}
	w.bypassIPs = nil

	// Stop tun2socks engine
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Log("[Wintun] Recovered from engine.Stop: %v", r)
			}
			close(done)
		}()
		engine.Stop()
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		utils.Log("[Wintun] tun2socks engine stop timeout")
	}

	w.active = false
	utils.Log("[Wintun] System VPN stopped")
	return nil
}

func (w *WintunManager) IsActive() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active
}
