package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	godebug "runtime/debug"
	"strconv"
	"strings"
	"time"

	_ "github.com/wlynxg/anet"
	"openflux/socks5"
	"openflux/transport"
	"openflux/transport/cupsonline"
	"openflux/transport/mailru"
	"openflux/transport/oneme"
	"openflux/transport/yandex"
	"openflux/tunnel"
	"openflux/utils"
)

const (
	roleClient    = "client"
	roleExit      = "exit"
	roleBenchSend = "bench-send"
	roleBenchSink = "bench-sink"

	inboundTUN    = "tun"
	inboundSOCKS5 = "socks5"

	codecBatched = "batched"
	codecLegacy  = "legacy"
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

	role := flag.String("role", "", "Role: client | exit | bench-send | bench-sink")
	inbound := flag.String("inbound", "", "Client inbound: tun (recommended on macOS) | socks5 (everywhere)")
	mode := flag.String("mode", "proxy", "Exit node mode: proxy/l4 (userspace net.Dial, no root) or l3 (fast SNAT, Linux root)")
	codec := flag.String("codec", codecBatched, "Wire codec: batched (zstd + coalescing) or legacy (per-packet LZ4)")

	// Deprecated / compatibility flags
	depExit := flag.Bool("exit-node", false, "Deprecated: use --role=exit")
	depClient := flag.Bool("client", false, "Deprecated: use --role=client")
	depTun := flag.Bool("tun", false, "Deprecated: use --inbound=tun")
	depSocks5Mode := flag.Bool("socks5-mode", false, "Deprecated: use --inbound=socks5")
	depLegacy := flag.Bool("legacy", false, "Deprecated: use --codec=legacy")
	depBenchSend := flag.Int64("bench-send", 0, "Deprecated: use --role=bench-send --bench-bytes=N")
	depBenchSink := flag.Bool("bench-sink", false, "Deprecated: use --role=bench-sink")

	debug := flag.Bool("debug", false, "Enable verbose debug logging")
	socksAddr := flag.String("socks5", ":1080", "SOCKS5 address")
	transportType := flag.String("transport", "yandex", "Transport type (yandex, vyandex, oneme, cupsonline, mailru)")
	flag.StringVar(&globalDocUrl, "url", "http://#", "Document URL(s) or Cupsonline base64 rooms")
	flag.StringVar(&maxToken, "maxToken", "", "MAX Web token. If u use MAX transport")
	flag.StringVar(&maxUid, "maxUid", "", "MAX call user id. If u use MAX transport")
	flag.StringVar(&localIP, "local-ip", "", "Egress IP for exit node (scoped RST drop)")

	benchBytes := flag.Int64("bench-bytes", 100, "Megabytes of data to send in bench-send mode")
	benchCompressible := flag.Bool("bench-compressible", false, "Send compressible text rather than pseudo-random binary")

	var secretKey string
	flag.StringVar(&secretKey, "key", "", "End-to-End encryption passphrase (or set OPENFLUX_KEY env / secret_key.txt)")
	var keyFile string
	flag.StringVar(&keyFile, "encryption-key-file", "", "File containing shared encryption secret")
	var yandexCookie string
	flag.StringVar(&yandexCookie, "ycookie", "", "Yandex session cookies (name=value; ...) to bypass showcaptcha")
	flag.Parse()

	// Normalize deprecated flags to current ones
	roleSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "role" {
			roleSet = true
		}
	})
	if !roleSet {
		if *depClient {
			*role = roleClient
		}
		if *depExit {
			*role = roleExit
		}
	}
	if *depTun {
		*inbound = inboundTUN
	}
	if *depSocks5Mode {
		*inbound = inboundSOCKS5
	}
	if *depLegacy {
		*codec = codecLegacy
	}
	if *depBenchSend > 0 {
		*role = roleBenchSend
		*benchBytes = *depBenchSend
	}
	if *depBenchSink {
		*role = roleBenchSink
	}

	// Platform defaults
	if *inbound == "" {
		if runtime.GOOS == "darwin" {
			*inbound = inboundTUN
		} else {
			*inbound = inboundSOCKS5
		}
	}

	if *role == "" {
		flag.Usage()
		os.Exit(1)
	}

	if localIP != "" {
		tunnel.SetLocalIP(localIP)
	}

	// The exit node often runs on a tiny VPS; keep the heap tight under load (GC aggressively)
	if *role == roleExit {
		godebug.SetGCPercent(20)
	}

	if yandexCookie != "" {
		yandex.YandexCookie = yandexCookie
	}

	if *debug {
		utils.EnableDebug()
	}

	secretKey = resolveSecretKey(secretKey, keyFile)

	if (globalDocUrl == "http://#" || globalDocUrl == "") && os.Getenv("OPENFLUX_URL") != "" {
		globalDocUrl = os.Getenv("OPENFLUX_URL")
	}

	exitMode, err := tunnel.ParseExitMode(*mode)
	if err != nil {
		log.Fatalf("Invalid exit mode: %v", err)
	}

	log.Printf("=== OpenFlux ===")
	log.Printf("Role: %s", *role)
	log.Printf("Transport: %s", *transportType)
	if *role == roleClient {
		log.Printf("Inbound: %s", *inbound)
	}
	if *role == roleExit {
		log.Printf("Exit mode: %s", exitMode.String())
	}

	config := transport.DefaultConfig()
	var rawTrans transport.Transport

	switch *transportType {
	case "vyandex":
		rawTrans = yandex.NewYandexVolgaTransport(globalDocUrl, config)
	case "yandex":
		rawTrans = yandex.NewYandexDocsTransport(globalDocUrl, config)
	case "oneme":
		uidint, _ := strconv.ParseInt(maxUid, 10, 64)
		rawTrans = oneme.NewOneMeTransport(*role == roleExit, maxToken, uidint, config)
	case "cupsonline":
		rawTrans = cupsonline.NewCupsonlineTransport(globalDocUrl, config, *role != roleExit)
	case "mailru":
		rawTrans = mailru.NewMailruDocsTransport(globalDocUrl, config)
	default:
		log.Fatalf("Unknown transport type: %s", *transportType)
	}

	context := *transportType
	if globalDocUrl != "" && globalDocUrl != "http://#" {
		context = globalDocUrl
	}

	encTrans, err := transport.NewEncryptedTransport(rawTrans, secretKey, context, *role == roleExit)
	if err != nil {
		log.Fatalf("Failed to initialize encrypted transport: %v", err)
	}

	var trans transport.Transport
	switch *codec {
	case codecBatched:
		log.Printf("Codec: batched (zstd + coalescing)")
		trans = transport.NewBatchedTransport(encTrans)
	case codecLegacy:
		log.Printf("Codec: legacy (per-packet LZ4)")
		trans = transport.NewCompressedTransport(encTrans)
	default:
		log.Fatalf("Unknown codec: %s (want batched|legacy)", *codec)
	}

	if *role == roleBenchSend {
		if *benchBytes <= 0 {
			log.Fatalf("--role=bench-send requires --bench-bytes=<MB>")
		}
		if err := trans.Start(); err != nil {
			log.Fatalf("Failed to start transport: %v", err)
		}
		runBenchSend(trans, int(*benchBytes), *benchCompressible)
		return
	}
	if *role == roleBenchSink {
		if err := trans.Start(); err != nil {
			log.Fatalf("Failed to start transport: %v", err)
		}
		runBenchSink(trans)
		return
	}

	if err := trans.Start(); err != nil {
		log.Fatalf("Failed to start transport: %v", err)
	}

	stopWatchdog := utils.StartSystemdWatchdog(func() bool {
		return trans.IsRunning()
	})
	defer stopWatchdog()

	switch *role {
	case roleExit:
		runExit(trans, exitMode)
	case roleClient:
		runClient(trans, *inbound, *socksAddr, exitMode)
	default:
		log.Fatalf("unhandled role %q", *role)
	}
}

func runExit(trans transport.Transport, exitMode tunnel.ExitMode) {
	ex, err := tunnel.NewExitNode(trans, exitMode.String())
	if err != nil {
		log.Fatalf("Exit node error: %v", err)
	}
	log.Printf("Running as EXIT NODE (mode: %s)", ex.Mode())
	if err := ex.Start(); err != nil {
		log.Fatalf("Exit start error: %v", err)
	}

	if exitMode == tunnel.ExitModeL3 {
		if localIP != "" {
			log.Printf("! Run: sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -s %s -j DROP", localIP)
		} else {
			log.Printf("! Kernel RSTs would tear down tunnel connections. Run:")
			log.Printf("!   sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -j DROP")
		}
	}

	select {}
}

func runClient(trans transport.Transport, inbound, socksAddr string, exitMode tunnel.ExitMode) {
	switch inbound {
	case inboundTUN:
		runClientTUN(trans)
	case inboundSOCKS5:
		log.Printf("Running as CLIENT with SOCKS5 on %s", socksAddr)
		tun := tunnel.NewTCPTunnelMode(trans, false, exitMode)
		server := socks5.NewSOCKS5Server(socksAddr, tun)
		if err := server.Start(); err != nil {
			log.Fatalf("SOCKS5 server error: %v", err)
		}
	default:
		log.Fatalf("--inbound: unknown value %q (want tun|socks5)", inbound)
	}
}

func runClientTUN(trans transport.Transport) {
	tc, err := NewTUNClient(trans, 1280)
	if err != nil {
		log.Fatalf("utun: %v", err)
	}
	log.Printf("utun interface: %s", tc.Name())

	if err := tc.SaveDefault(); err != nil {
		log.Fatalf("save default route: %v", err)
	}
	if err := tc.SetupInterface(); err != nil {
		log.Fatalf("setup utun (need sudo): %v", err)
	}
	log.Printf("utun up; bypass gateway is %s", tc.Gateway())

	watcher := NewSocketWatcher(tc.Gateway(), func() {
		log.Printf("Socket set stable; taking default route into the tunnel")
		if err := tc.ConfigureDefault(); err != nil {
			log.Printf("FATAL: configure default: %v", err)
			return
		}
		tc.Start()
		log.Printf("Tunnel active")
	})
	watcher.Start(2 * time.Second)

	sigCh := make(chan os.Signal, 1)
	notifySignals(sigCh)
	<-sigCh
	watcher.Stop()
	log.Printf("Shutting down, restoring default route...")
	if err := tc.Close(); err != nil {
		log.Printf("cleanup warning: %v", err)
	}
	tc.RestoreDefault()
	log.Printf("Shutdown complete")
	os.Exit(0)
}
