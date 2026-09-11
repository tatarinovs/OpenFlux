package sysproxy

import (
	"fmt"
	"strings"
	"sync"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

var (
	modwininet            = syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption = modwininet.NewProc("InternetSetOptionW")
)

const (
	internetOptionSettingsChanged = 39
	internetOptionRefresh         = 37
)

type SysProxy struct {
	mu           sync.Mutex
	active       bool
	prevEnabled  uint64
	prevServer   string
	prevOverride string
}

var globalProxy = &SysProxy{}

func refreshInternetSettings() {
	procInternetSetOption.Call(0, uintptr(internetOptionSettingsChanged), 0, 0)
	procInternetSetOption.Call(0, uintptr(internetOptionRefresh), 0, 0)
}

func Enable(socksAddr string, bypass string) error {
	globalProxy.mu.Lock()
	defer globalProxy.mu.Unlock()

	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open internet settings registry: %w", err)
	}
	defer k.Close()

	if !globalProxy.active {
		val, _, err := k.GetIntegerValue("ProxyEnable")
		if err == nil {
			globalProxy.prevEnabled = val
		}
		str, _, err := k.GetStringValue("ProxyServer")
		if err == nil {
			globalProxy.prevServer = str
		}
		ovr, _, err := k.GetStringValue("ProxyOverride")
		if err == nil {
			globalProxy.prevOverride = ovr
		}
	}

	hostPort := socksAddr
	if strings.HasPrefix(hostPort, ":") {
		hostPort = "127.0.0.1" + hostPort
	}
	proxyServerVal := fmt.Sprintf("socks=%s;http=%s;https=%s", hostPort, hostPort, hostPort)

	overrideVal := "<local>;localhost;127.*;192.168.*;10.*"
	if bypass != "" {
		overrideVal = bypass
	}

	if err := k.SetDWordValue("ProxyEnable", 1); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyServer", proxyServerVal); err != nil {
		return err
	}
	if err := k.SetStringValue("ProxyOverride", overrideVal); err != nil {
		return err
	}

	refreshInternetSettings()
	globalProxy.active = true
	return nil
}

func Disable() error {
	globalProxy.mu.Lock()
	defer globalProxy.mu.Unlock()

	if !globalProxy.active {
		return nil
	}

	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if globalProxy.prevEnabled == 0 {
		_ = k.SetDWordValue("ProxyEnable", 0)
		_ = k.SetStringValue("ProxyServer", "")
	} else {
		_ = k.SetDWordValue("ProxyEnable", uint32(globalProxy.prevEnabled))
		_ = k.SetStringValue("ProxyServer", globalProxy.prevServer)
		_ = k.SetStringValue("ProxyOverride", globalProxy.prevOverride)
	}

	refreshInternetSettings()
	globalProxy.active = false
	return nil
}

func IsActive() bool {
	globalProxy.mu.Lock()
	defer globalProxy.mu.Unlock()
	return globalProxy.active
}
