package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	godebug "runtime/debug"
	"strconv"
	"strings"

	_ "github.com/wlynxg/anet"
	"universal-bypass-tool/socks5"
	"universal-bypass-tool/transport"
	"universal-bypass-tool/transport/cupsonline"
	"universal-bypass-tool/transport/oneme"
	"universal-bypass-tool/transport/yandex"
	"universal-bypass-tool/tunnel"
	"universal-bypass-tool/utils"
)

var (
	globalDocUrl     string
	maxToken         string
	maxUid           string
	localIP          string
	defaultSecretKey string
)

func resolveSecretKey(cliKey, keyFile string) string {
	if keyFile != "" {
		if data, err := os.ReadFile(keyFile); err == nil {
			return strings.TrimSpace(string(data))
		} else {
			log.Fatalf("Failed to read encryption key file: %v", err)
		}
	}
	if cliKey == "none" || cliKey == "off" {
		return ""
	}
	if cliKey != "" {
		return cliKey
	}
	if envKey := os.Getenv("OPENFLUX_KEY"); envKey != "" {
		if envKey == "none" || envKey == "off" {
			return ""
		}
		return envKey
	}
	// Try loading from secret_key.txt in working directory
	if data, err := os.ReadFile("secret_key.txt"); err == nil {
		k := strings.TrimSpace(string(data))
		if k != "" {
			return k
		}
	}
	// Try loading from secret_key.txt next to executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		if data, err := os.ReadFile(filepath.Join(exeDir, "secret_key.txt")); err == nil {
			k := strings.TrimSpace(string(data))
			if k != "" {
				return k
			}
		}
	}
	return defaultSecretKey
}

func main() {
	//os.Setenv("GODEBUG", "netdns=go")
	fmt.Print("written by p1neappleXpress\n")

	exitNode := flag.Bool("exit-node", false, "Run as exit node")
	client := flag.Bool("client", false, "Run as client")
	debug := flag.Bool("debug", false, "Enable verbose debug logging")
	socksAddr := flag.String("socks5", ":1080", "SOCKS5 address")
	transportType := flag.String("transport", "yandex", "Transport type (yandex, vyandex, oneme, cupsonline)")
	modeStr := flag.String("mode", "proxy", "Exit node mode: proxy (default, userspace net.Dial, no root) or raw")
	flag.StringVar(&globalDocUrl, "url", "http://#", "Document URL(s) or Cupsonline base64 rooms")
	flag.StringVar(&maxToken, "maxToken", "", "MAX Web token. If u use MAX transport")
	flag.StringVar(&maxUid, "maxUid", "", "MAX call user id. If u use MAX transport")
	flag.StringVar(&localIP, "local-ip", "", "Egress IP for exit node (scoped RST drop)")
	var secretKey string
	flag.StringVar(&secretKey, "key", "", "End-to-End encryption passphrase (or set OPENFLUX_KEY env / secret_key.txt)")
	var keyFile string
	flag.StringVar(&keyFile, "encryption-key-file", "", "File containing shared encryption secret")
	var yandexCookie string
	flag.StringVar(&yandexCookie, "ycookie", "", "Yandex session cookies (name=value; ...) to bypass showcaptcha")
	flag.Parse()

	if localIP != "" {
		tunnel.SetLocalIP(localIP)
	}

	// The exit node often runs on a tiny VPS; keep the heap tight under load (GC aggressively)
	if *exitNode {
		godebug.SetGCPercent(20)
	}

	if yandexCookie != "" {
		yandex.YandexCookie = yandexCookie
	}

	if !*exitNode && !*client {
		flag.Usage()
		os.Exit(1)
	}

	if *debug {
		utils.EnableDebug()
	}

	secretKey = resolveSecretKey(secretKey, keyFile)

	if (globalDocUrl == "http://#" || globalDocUrl == "") && os.Getenv("OPENFLUX_URL") != "" {
		globalDocUrl = os.Getenv("OPENFLUX_URL")
	}

	exitMode, err := tunnel.ParseExitMode(*modeStr)
	if err != nil {
		log.Fatalf("Invalid exit mode: %v", err)
	}

	log.Printf("=== Universal Bypass Tool ===")
	log.Printf("Mode: %s", map[bool]string{true: "EXIT NODE", false: "CLIENT"}[*exitNode])
	log.Printf("Transport: %s", *transportType)

	config := transport.DefaultConfig()
	var rawTrans transport.Transport

	switch *transportType {
	case "vyandex":
		rawTrans = yandex.NewYandexVolgaTransport(globalDocUrl, config)
	case "yandex":
		rawTrans = yandex.NewYandexDocsTransport(globalDocUrl, config)
	case "oneme":
		uidint, _ := strconv.ParseInt(maxUid, 10, 64)
		rawTrans = oneme.NewOneMeTransport(*exitNode, maxToken, uidint, config)
	case "cupsonline":
		rawTrans = cupsonline.NewCupsonlineTransport(globalDocUrl, config, *client)
	default:
		log.Fatalf("Unknown transport type: %s", *transportType)
	}

	context := *transportType
	if globalDocUrl != "" && globalDocUrl != "http://#" {
		context = globalDocUrl
	}

	encTrans, err := transport.NewEncryptedTransport(rawTrans, secretKey, context, *exitNode)
	if err != nil {
		log.Fatalf("Failed to initialize encrypted transport: %v", err)
	}
	trans := transport.NewCompressedTransport(encTrans)

	if err := trans.Start(); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}

	stopWatchdog := utils.StartSystemdWatchdog(func() bool {
		return trans.IsRunning()
	})
	defer stopWatchdog()

	tun := tunnel.NewTCPTunnelMode(trans, *exitNode, exitMode)

	if *exitNode {
		log.Printf("Running as EXIT NODE (mode: %s, userspace proxy, no raw sockets needed)", exitMode)
		if exitMode == tunnel.ExitModeRaw {
			if localIP != "" {
				log.Printf("! Run: sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -s %s -j DROP", localIP)
			} else {
				log.Printf("! Run: sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -j DROP")
			}
		}
		select {}
	} else {
		log.Printf("Running as CLIENT with SOCKS5 on %s", *socksAddr)
		server := socks5.NewSOCKS5Server(*socksAddr, tun)
		if err := server.Start(); err != nil {
			log.Fatalf("SOCKS5 server error: %v", err)
		}
	}
}
