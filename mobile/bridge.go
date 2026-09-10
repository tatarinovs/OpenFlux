package main

/*
#include <jni.h>
#include <stdlib.h>

static const char* get_string_utf(JNIEnv* env, jstring str) {
    if (!str) return NULL;
    return (*env)->GetStringUTFChars(env, str, NULL);
}

static void release_string_utf(JNIEnv* env, jstring str, const char* chars) {
    if (str && chars) {
        (*env)->ReleaseStringUTFChars(env, str, chars);
    }
}

static jstring new_string_utf(JNIEnv* env, const char* str) {
    return (*env)->NewStringUTF(env, str);
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/xjasonlyu/tun2socks/v2/engine"
	tunlog "github.com/xjasonlyu/tun2socks/v2/log"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"universal-bypass-tool/socks5"
	"universal-bypass-tool/transport"
	"universal-bypass-tool/transport/oneme"
	"universal-bypass-tool/transport/yandex"
	"universal-bypass-tool/tunnel"
	"universal-bypass-tool/utils"
)

func init() {
	logger := zap.New(zapcore.NewNopCore(), zap.WithFatalHook(zapcore.WriteThenPanic))
	tunlog.SetLogger(logger)
}

type AppCore struct {
	mu           sync.Mutex
	logMu        sync.Mutex
	running      bool
	vpnActive    bool
	trans        transport.Transport
	socksServer  *socks5.SOCKS5Server
	tunEngineOn  bool
	startTime    time.Time
	recentLogs   []string
	maxLogLines  int
	currentPort   int
	currentTrans  string
	lastSentBytes uint64
	lastRecvBytes uint64
	lastStatsTime time.Time
	lastSpeedStr  string
}

var core = &AppCore{
	maxLogLines: 300,
}

func init() {
	utils.SetLogCallback(func(msg string) {
		core.addLog(msg)
	})
}

var (
	localZoneMu sync.RWMutex
	localZone   *time.Location = time.Local
)

func setTimezoneOffset(offsetSec int) {
	localZoneMu.Lock()
	loc := time.FixedZone("Local", offsetSec)
	localZone = loc
	time.Local = loc
	localZoneMu.Unlock()
}

func getLocalTime() time.Time {
	localZoneMu.RLock()
	loc := localZone
	localZoneMu.RUnlock()
	return time.Now().In(loc)
}

func (c *AppCore) addLog(msg string) {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	timestamp := getLocalTime().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", timestamp, msg)
	c.recentLogs = append(c.recentLogs, line)
	if len(c.recentLogs) > c.maxLogLines {
		c.recentLogs = c.recentLogs[len(c.recentLogs)-c.maxLogLines:]
	}
}

func (c *AppCore) getLogs() string {
	c.logMu.Lock()
	defer c.logMu.Unlock()
	return strings.Join(c.recentLogs, "\n")
}

var DefaultSecretKey = ""

func (c *AppCore) startProxy(transType, docUrl, maxToken, maxUid, secretKey string, port int, listenAll, debug bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return fmt.Errorf("already running")
	}

	if debug {
		utils.EnableDebug()
	}

	utils.Log("Starting OpenFlux Core (Transport: %s, Port: %d)", transType, port)

	cfg := transport.DefaultConfig()
	var rawTrans transport.Transport

	switch transType {
	case "yandex":
		if docUrl == "" {
			return fmt.Errorf("yandex docs URL is required")
		}
		rawTrans = yandex.NewYandexDocsTransport(docUrl, cfg)
	case "oneme":
		uidint, _ := strconv.ParseInt(maxUid, 10, 64)
		rawTrans = oneme.NewOneMeTransport(false, maxToken, uidint, cfg)
	default:
		return fmt.Errorf("unknown transport type: %s", transType)
	}

	effectiveKey := secretKey
	if effectiveKey == "" {
		effectiveKey = DefaultSecretKey
	} else if effectiveKey == "none" || effectiveKey == "off" {
		effectiveKey = ""
	}

	encTrans, err := transport.NewEncryptedTransport(rawTrans, effectiveKey)
	if err != nil {
		utils.Log("Failed to initialize encrypted transport: %v", err)
		return err
	}
	trans := transport.NewCompressedTransport(encTrans)

	if err := trans.Start(); err != nil {
		utils.Log("Failed to start transport: %v", err)
		return err
	}

	tun := tunnel.NewTCPTunnel(trans, false)

	listenHost := "127.0.0.1"
	if listenAll {
		listenHost = "0.0.0.0"
	}
	socksAddr := fmt.Sprintf("%s:%d", listenHost, port)

	srv := socks5.NewSOCKS5Server(socksAddr, tun)
	go func() {
		if err := srv.Start(); err != nil {
			utils.Log("SOCKS5 server stopped: %v", err)
		}
	}()

	c.trans = trans
	c.socksServer = srv
	c.running = true
	c.vpnActive = false
	c.startTime = time.Now()
	c.lastStatsTime = time.Now()
	c.lastSentBytes = 0
	c.lastRecvBytes = 0
	c.currentPort = port
	c.currentTrans = transType

	utils.Log("OpenFlux SOCKS5 listening on %s", socksAddr)
	return nil
}

func (c *AppCore) startVpn(tunFd int, transType, docUrl, maxToken, maxUid, secretKey string, port int, debug bool) error {
	// First start the local proxy on 127.0.0.1
	if err := c.startProxy(transType, docUrl, maxToken, maxUid, secretKey, port, false, debug); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	utils.Log("Attaching tun2socks to VPN interface (FD: %d)...", tunFd)

	key := &engine.Key{
		Device:   fmt.Sprintf("fd://%d", tunFd),
		Proxy:    fmt.Sprintf("socks5://127.0.0.1:%d", port),
		MTU:      1500,
		LogLevel: "info",
	}

	engine.Insert(key)
	engine.Start()

	c.tunEngineOn = true
	c.vpnActive = true
	utils.Log("VPN tunnel active and routed to OpenFlux SOCKS5")
	return nil
}

func (c *AppCore) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return
	}

	utils.Log("Stopping OpenFlux Core...")

	if c.tunEngineOn {
		c.tunEngineOn = false
		done := make(chan struct{})
		go func() {
			defer func() {
				if r := recover(); r != nil {
					utils.Log("Recovered from engine.Stop: %v", r)
				}
				close(done)
			}()
			engine.Stop()
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			utils.Log("tun2socks stop timed out")
		}
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
	c.vpnActive = false
	c.lastStatsTime = time.Time{}
	c.lastSentBytes = 0
	c.lastRecvBytes = 0
	c.lastSpeedStr = ""
	utils.Log("OpenFlux Core stopped")
}

func (c *AppCore) stats() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return "Stopped"
	}

	uptime := time.Since(c.startTime).Truncate(time.Second)
	mode := "Только прокси"
	if c.vpnActive {
		mode = "VPN (Туннель)"
	}
	transName := "Yandex Docs"
	if c.currentTrans != "yandex" {
		transName = "MAX Messenger"
	}

	var sentBytes, recvBytes, sentPkts, recvPkts uint64
	if c.trans != nil {
		st := c.trans.Stats()
		sentBytes = st.BytesSent
		recvBytes = st.BytesReceived
		sentPkts = st.PacketsSent
		recvPkts = st.PacketsRecv
	}

	speed := c.lastSpeedStr
	if speed == "" {
		speed = "↑ 0 B/s  ↓ 0 B/s"
	}

	return fmt.Sprintf("Режим: %s\nТранспорт: %s\nСкорость: %s\nВремя: %s\nОтправлено: %s (%d пак.)\nПринято: %s (%d пак.)",
		mode, transName, speed, uptime,
		formatBytes(sentBytes), sentPkts,
		formatBytes(recvBytes), recvPkts)
}

func (c *AppCore) trafficStats() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running || c.trans == nil {
		return ""
	}

	st := c.trans.Stats()
	now := time.Now()

	var upSpeed, downSpeed float64
	if !c.lastStatsTime.IsZero() {
		dur := now.Sub(c.lastStatsTime).Seconds()
		if dur > 0.3 {
			if st.BytesSent >= c.lastSentBytes {
				upSpeed = float64(st.BytesSent-c.lastSentBytes) / dur
			}
			if st.BytesReceived >= c.lastRecvBytes {
				downSpeed = float64(st.BytesReceived-c.lastRecvBytes) / dur
			}
			c.lastSentBytes = st.BytesSent
			c.lastRecvBytes = st.BytesReceived
			c.lastStatsTime = now
			c.lastSpeedStr = fmt.Sprintf("↑ %s/s  ↓ %s/s", formatSpeed(upSpeed), formatSpeed(downSpeed))
		}
	} else {
		c.lastSentBytes = st.BytesSent
		c.lastRecvBytes = st.BytesReceived
		c.lastStatsTime = now
		c.lastSpeedStr = "↑ 0 B/s  ↓ 0 B/s"
	}

	return fmt.Sprintf("↑ %s/s (%s)  ↓ %s/s (%s)",
		formatSpeed(upSpeed), formatBytes(st.BytesSent),
		formatSpeed(downSpeed), formatBytes(st.BytesReceived))
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.2f MB", bytesPerSec/(1024*1024))
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func jstringToGo(env *C.JNIEnv, str C.jstring) string {
	cStr := C.get_string_utf(env, str)
	if cStr == nil {
		return ""
	}
	defer C.release_string_utf(env, str, cStr)
	return C.GoString(cStr)
}

//export Java_com_openflux_client_core_OpenFluxCore_startProxy
func Java_com_openflux_client_core_OpenFluxCore_startProxy(
	env *C.JNIEnv,
	thiz C.jobject,
	transportType C.jstring,
	url C.jstring,
	maxToken C.jstring,
	maxUid C.jstring,
	secretKey C.jstring,
	port C.jint,
	listenAll C.jboolean,
	debug C.jboolean,
) C.jint {
	tType := jstringToGo(env, transportType)
	u := jstringToGo(env, url)
	mToken := jstringToGo(env, maxToken)
	mUid := jstringToGo(env, maxUid)
	sKey := jstringToGo(env, secretKey)
	p := int(port)
	lAll := bool(listenAll != 0)
	dbg := bool(debug != 0)

	err := core.startProxy(tType, u, mToken, mUid, sKey, p, lAll, dbg)
	if err != nil {
		utils.Log("startProxy error: %v", err)
		return -1
	}
	return 0
}

//export Java_com_openflux_client_core_OpenFluxCore_startVpn
func Java_com_openflux_client_core_OpenFluxCore_startVpn(
	env *C.JNIEnv,
	thiz C.jobject,
	tunFd C.jint,
	transportType C.jstring,
	url C.jstring,
	maxToken C.jstring,
	maxUid C.jstring,
	secretKey C.jstring,
	port C.jint,
	debug C.jboolean,
) C.jint {
	fd := int(tunFd)
	tType := jstringToGo(env, transportType)
	u := jstringToGo(env, url)
	mToken := jstringToGo(env, maxToken)
	mUid := jstringToGo(env, maxUid)
	sKey := jstringToGo(env, secretKey)
	p := int(port)
	dbg := bool(debug != 0)

	err := core.startVpn(fd, tType, u, mToken, mUid, sKey, p, dbg)
	if err != nil {
		utils.Log("startVpn error: %v", err)
		return -1
	}
	return 0
}

//export Java_com_openflux_client_core_OpenFluxCore_stop
func Java_com_openflux_client_core_OpenFluxCore_stop(env *C.JNIEnv, thiz C.jobject) C.jint {
	core.stop()
	return 0
}

//export Java_com_openflux_client_core_OpenFluxCore_isRunning
func Java_com_openflux_client_core_OpenFluxCore_isRunning(env *C.JNIEnv, thiz C.jobject) C.jboolean {
	core.mu.Lock()
	r := core.running
	core.mu.Unlock()
	if r {
		return 1
	}
	return 0
}

//export Java_com_openflux_client_core_OpenFluxCore_getStats
func Java_com_openflux_client_core_OpenFluxCore_getStats(env *C.JNIEnv, thiz C.jobject) C.jstring {
	s := core.stats()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	return C.new_string_utf(env, cs)
}

//export Java_com_openflux_client_core_OpenFluxCore_getRecentLogs
func Java_com_openflux_client_core_OpenFluxCore_getRecentLogs(env *C.JNIEnv, thiz C.jobject) C.jstring {
	l := core.getLogs()
	cs := C.CString(l)
	defer C.free(unsafe.Pointer(cs))
	return C.new_string_utf(env, cs)
}

//export Java_com_openflux_client_core_OpenFluxCore_getTrafficStats
func Java_com_openflux_client_core_OpenFluxCore_getTrafficStats(env *C.JNIEnv, thiz C.jobject) C.jstring {
	s := core.trafficStats()
	cs := C.CString(s)
	defer C.free(unsafe.Pointer(cs))
	return C.new_string_utf(env, cs)
}

//export Java_com_openflux_client_core_OpenFluxCore_setTimezoneOffset
func Java_com_openflux_client_core_OpenFluxCore_setTimezoneOffset(env *C.JNIEnv, thiz C.jobject, offsetSeconds C.jint) {
	setTimezoneOffset(int(offsetSeconds))
}

func main() {}
