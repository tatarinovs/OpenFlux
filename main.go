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

func resolveSecretKey(cliKey string) string {
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

	exitNode := flag.Bool("exit-node", false, "Run as exit node (needs root)")
	client := flag.Bool("client", false, "Run as client")
	debug := flag.Bool("debug", false, "Enable verbose debug logging")
	socksAddr := flag.String("socks5", ":1080", "SOCKS5 address")
	transportType := flag.String("transport", "yandex", "Transport type (yandex, vyandex, oneme)")
	flag.StringVar(&globalDocUrl, "url", "http://#", "Document URL(s), comma or space separated. If u use Yandex.Docs transport")
	flag.StringVar(&maxToken, "maxToken", "", "MAX Web token. If u use MAX transport")
	flag.StringVar(&maxUid, "maxUid", "", "MAX call user id. If u use MAX transport")
	flag.StringVar(&localIP, "local-ip", "", "Egress IP for exit node (scoped RST drop)")
	var secretKey string
	flag.StringVar(&secretKey, "key", "", "End-to-End encryption key (or set OPENFLUX_KEY env / secret_key.txt)")
	var yandexCookie string
	flag.StringVar(&yandexCookie, "ycookie", "", "Yandex session cookies (name=value; ...) to bypass showcaptcha")
	multiListen := flag.Bool("multi-listen", false, "Listen to all pool URLs simultaneously (auto-enabled for exit node)")
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

	secretKey = resolveSecretKey(secretKey)

	if (globalDocUrl == "http://#" || globalDocUrl == "") && os.Getenv("OPENFLUX_URL") != "" {
		globalDocUrl = os.Getenv("OPENFLUX_URL")
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
		yTrans := yandex.NewYandexDocsTransport(globalDocUrl, config)
		if *exitNode || *multiListen {
			yTrans.SetMultiListen(true)
		}
		rawTrans = yTrans
	case "oneme":
		uidint, _ := strconv.ParseInt(maxUid, 10, 64)
		rawTrans = oneme.NewOneMeTransport(*exitNode, maxToken, uidint, config)
	default:
		log.Fatalf("Unknown transport type: %s", *transportType)
	}

	encTrans, err := transport.NewEncryptedTransport(rawTrans, secretKey)
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

	tun := tunnel.NewTCPTunnel(trans, *exitNode)

	if *exitNode {
		log.Printf("Running as EXIT NODE (needs root for raw socket)")
		if localIP != "" {
			log.Printf("! Run: sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -s %s -j DROP", localIP)
		} else {
			log.Printf("! Kernel RSTs would tear down tunnel connections. Prefer a scoped rule:")
			log.Printf("!   assign a dedicated alias IP, run with --local-ip <ip>, then:")
			log.Printf("!   sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -s <ip> -j DROP")
			log.Printf("! Host-wide fallback (drops ALL outbound RST; makes closed ports look filtered):")
			log.Printf("!   sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -j DROP")
		}
		select {}
	} else {
		log.Printf("Running as CLIENT (SOCKS5 on %s)", *socksAddr)
		socks5Server := socks5.NewSOCKS5Server(*socksAddr, tun)
		log.Fatal(socks5Server.Start())
	}
}
