package yandex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"universal-bypass-tool/transport"
	"universal-bypass-tool/utils"
)

// YandexCookie is an optional Cookie header value (name=value; ...) sent with
// the document fetch request and WebSocket connection to bypass Yandex antibot (showcaptcha).
var YandexCookie string

const DocUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

func resolveYandexCookie(cliCookie string) string {
	if cliCookie != "" {
		return cliCookie
	}
	if envCookie := os.Getenv("OPENFLUX_YCOOKIE"); envCookie != "" {
		return envCookie
	}
	return YandexCookie
}

type YandexDocsInfo struct {
	CookieStr   string
	Token       string
	DocID       string
	CallbackURL string
	UserID      string
	Origin      string
	Host        string
	WsURL       string
	Permissions map[string]interface{}
	OpenCmd     map[string]interface{}
}

type DocSession struct {
	Info       YandexDocsInfo
	Conn       *websocket.Conn
	WriteQueue chan []byte
	UserID     string
	writeMu    sync.Mutex
}

func (s *DocSession) safeWrite(messageType int, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.Conn == nil {
		return fmt.Errorf("connection closed")
	}
	s.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.Conn.WriteMessage(messageType, data)
}

func parseDocURLs(raw string) []string {
	delims := func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	}
	parts := strings.FieldsFunc(raw, delims)
	var urls []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && (strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://")) {
			urls = append(urls, p)
		}
	}
	if len(urls) == 0 && strings.TrimSpace(raw) != "" {
		urls = append(urls, strings.TrimSpace(raw))
	}
	return urls
}

type YandexDocsTransport struct {
	*transport.BaseTransport

	urls          []string
	currentUrlIdx atomic.Int32
	session       *DocSession

	userCounter  atomic.Int32
	baseUserID   string
	reconnectGen atomic.Uint32
}

func NewYandexDocsTransport(url string, config transport.TransportConfig) *YandexDocsTransport {
	parsedUrls := parseDocURLs(url)
	t := &YandexDocsTransport{
		BaseTransport: transport.NewBaseTransport(config),
		urls:          parsedUrls,
	}
	t.baseUserID = randUserID()
	return t
}

func (t *YandexDocsTransport) Start() error {
	if err := t.BaseTransport.Start(); err != nil {
		return err
	}

	t.baseUserID = randUserID()
	go t.keepAliveLoop()
	t.connectToDoc(0)

	return nil
}

func (t *YandexDocsTransport) Stop() error {
	t.BaseTransport.Stop()
	t.Mu.Lock()
	session := t.session
	t.session = nil
	t.Mu.Unlock()

	if session != nil && session.Conn != nil {
		session.Conn.Close()
	}
	return nil
}

func (t *YandexDocsTransport) Send(data []byte) error {
	if !t.IsConnected() {
		return fmt.Errorf("transport not connected")
	}

	t.Mu.RLock()
	session := t.session
	t.Mu.RUnlock()

	if session == nil {
		return fmt.Errorf("no active session")
	}

	select {
	case session.WriteQueue <- data:
		t.RecordSend(len(data))
		return nil
	default:
		return fmt.Errorf("write queue full")
	}
}

func (t *YandexDocsTransport) connectToDoc(attempt int) {
	if !t.IsRunning() {
		return
	}

	if len(t.urls) == 0 {
		utils.Debugf("[YDOCS] No valid document URLs configured")
		return
	}

	currentIdx := int(t.currentUrlIdx.Load()) % len(t.urls)
	targetUrl := t.urls[currentIdx]
	utils.Debugf("[YDOCS] connectToDoc attempt %d using pool [%d/%d]: %s", attempt+1, currentIdx+1, len(t.urls), targetUrl)

	go func(activeUrl string, activeIdx int) {
		t.Mu.Lock()
		existingSession := t.session
		t.Mu.Unlock()

		var userID string
		if existingSession != nil {
			userID = existingSession.UserID
		} else {
			suffix := fmt.Sprintf("%03d", t.userCounter.Add(1)%1000)
			userID = t.baseUserID + suffix
		}

		info, err := t.fetchDocInfo(activeUrl, userID)
		if err != nil {
			utils.Debugf("[YDOCS] fetchDocInfo failed on [%s]: %v", activeUrl, err)
			if len(t.urls) > 1 {
				nextIdx := int(t.currentUrlIdx.Add(1)) % len(t.urls)
				utils.Debugf("[YDOCS] Switched to fallback URL [%d/%d]: %s", nextIdx+1, len(t.urls), t.urls[nextIdx])
			}
			t.scheduleReconnect(attempt)
			return
		}

		netDialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		dialer := websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
			NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return netDialer.DialContext(ctx, "tcp4", addr)
			},
		}
		headers := http.Header{}
		headers.Set("User-Agent", DocUserAgent)
		headers.Set("Origin", info.Origin)
		cookie := info.CookieStr
		if extraCookie := resolveYandexCookie(YandexCookie); extraCookie != "" {
			if cookie != "" {
				cookie = cookie + "; " + extraCookie
			} else {
				cookie = extraCookie
			}
		}
		headers.Set("Cookie", cookie)
		headers.Set("Host", info.Host)

		t.Mu.Lock()
		if existingSession != nil && existingSession.Conn != nil {
			existingSession.Conn.Close()
		}
		t.Mu.Unlock()

		conn, _, err := dialer.Dial(info.WsURL, headers)
		if err != nil {
			utils.Debugf("[YDOCS] WebSocket dial failed on [%s]: %v", activeUrl, err)
			if len(t.urls) > 1 {
				nextIdx := int(t.currentUrlIdx.Add(1)) % len(t.urls)
				utils.Debugf("[YDOCS] Switched to fallback URL [%d/%d]: %s", nextIdx+1, len(t.urls), t.urls[nextIdx])
			}
			t.scheduleReconnect(attempt)
			return
		}

		conn.SetReadLimit(2 << 20)
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		})

		writeQueue := make(chan []byte, t.GetConfig().MaxQueueSize)
		if existingSession != nil {
			writeQueue = existingSession.WriteQueue
		}

		session := &DocSession{
			Info:       info,
			Conn:       conn,
			WriteQueue: writeQueue,
			UserID:     userID,
		}

		t.Mu.Lock()
		t.session = session
		t.SetConnected(true)
		t.Mu.Unlock()

		if existingSession == nil {
			go t.writerLoop()
		}

		// Auth - use safeWrite
		auth1 := fmt.Sprintf(`40{"token":"%s"}`, info.Token)
		session.safeWrite(websocket.TextMessage, []byte(auth1))

		authData := map[string]interface{}{
			"type": "auth", "docid": info.DocID, "token": "fghhfgsjdgfjs",
			"user": map[string]interface{}{"id": userID}, "editorType": 0,
			"lastOtherSaveTime": -1, "permissions": info.Permissions,
			"openCmd": info.OpenCmd, "coEditingMode": "fast", "jwtOpen": info.Token,
		}
		messagePart, _ := json.Marshal([]interface{}{"message", authData})
		session.safeWrite(websocket.TextMessage, []byte(fmt.Sprintf("42%s", string(messagePart))))

		for t.IsRunning() {
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				utils.Debugf("[YDOCS] Read error on [%s]: %v", activeUrl, err)
				t.SetConnected(false)
				conn.Close()
				if len(t.urls) > 1 {
					nextIdx := int(t.currentUrlIdx.Add(1)) % len(t.urls)
					utils.Debugf("[YDOCS] Reconnect will try URL [%d/%d]: %s", nextIdx+1, len(t.urls), t.urls[nextIdx])
				}
				t.scheduleReconnect(attempt)
				return
			}
			t.handleMessage(session, message)
		}
	}(targetUrl, currentIdx)
}

var cursorRegex = regexp.MustCompile(`"cursor":"[^;]+;([^"]+)"`)

func (t *YandexDocsTransport) writerLoop() {
	for t.IsRunning() {
		t.Mu.Lock()
		session := t.session
		t.Mu.Unlock()

		if session == nil || session.Conn == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}

		select {
		case packet, ok := <-session.WriteQueue:
			if !ok {
				return
			}
			payload := base64.StdEncoding.EncodeToString(packet)
			msg := fmt.Sprintf(`42["message",{"type":"cursor","cursor":"18;%s"}]`, payload)

			if err := session.safeWrite(websocket.TextMessage, []byte(msg)); err != nil {
				utils.Debugf("[YDOCS] Write error: %v", err)
			}
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func (t *YandexDocsTransport) keepAliveLoop() {
	ticker := time.NewTicker(t.GetConfig().KeepAliveInterval)
	defer ticker.Stop()
	keepAliveMsg := `42["message",{"type":"cursor","cursor":"18;---KA---"}]`

	for t.IsRunning() {
		<-ticker.C
		t.Mu.Lock()
		session := t.session
		t.Mu.Unlock()

		if session != nil && session.Conn != nil {
			if err := session.safeWrite(websocket.TextMessage, []byte(keepAliveMsg)); err != nil {
				utils.Debugf("[YDOCS] Keep-alive failed: %v", err)
				t.SetConnected(false)
			}
		}
	}
}

func (t *YandexDocsTransport) handleMessage(session *DocSession, data []byte) {
	text := string(data)

	if strings.Contains(text, "---KA---") {
		return
	}

	// Socket.IO ping - respond with pong (use safeWrite)
	if text == "2" {
		if session != nil && session.Conn != nil {
			session.safeWrite(websocket.TextMessage, []byte("3"))
		}
		return
	}
	if text == "3" {
		return
	}

	if strings.Contains(text, "saveChanges") || strings.Contains(text, "cursor") {
		base64Str := t.extractBase64String(text)
		if base64Str == "" {
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(base64Str)
		if err != nil {
			utils.Debugf("[YDOCS] Base64 decode error: %v", err)
			return
		}

		t.RecordReceive(len(decoded))
		t.CallReceive(decoded)
	}
}

func (t *YandexDocsTransport) extractBase64String(response string) string {
	if strings.Contains(response, "saveChanges") {
		marker := `"excelAdditionalInfo":"`
		left := strings.Index(response, marker) + len(marker)
		if left < len(marker) {
			return ""
		}
		right := strings.Index(response[left:], `"`)
		if right == -1 {
			return ""
		}
		return response[left : left+right]
	}

	matches := cursorRegex.FindStringSubmatch(response)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (t *YandexDocsTransport) scheduleReconnect(attempt int) {
	if !t.IsRunning() || attempt >= t.GetConfig().MaxReconnectAttempts {
		return
	}

	gen := t.reconnectGen.Add(1)
	t.RecordReconnect()
	delay := time.Duration(1+attempt) * time.Second
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}
	utils.Debugf("[YDOCS] Reconnecting in %v (attempt %d)...", delay, attempt+1)
	time.Sleep(delay)
	if t.reconnectGen.Load() != gen || !t.IsRunning() {
		return
	}
	t.connectToDoc(attempt + 1)
}

func (t *YandexDocsTransport) fetchDocInfo(url, userID string) (YandexDocsInfo, error) {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return YandexDocsInfo{}, err
	}
	req.Header.Set("User-Agent", DocUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru,en-US;q=0.7,en;q=0.3")
	if cookie := resolveYandexCookie(YandexCookie); cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	resp, err := client.Do(req)
	if err != nil {
		return YandexDocsInfo{}, err
	}
	defer resp.Body.Close()

	if strings.Contains(resp.Request.URL.String(), "showcaptcha") {
		return YandexDocsInfo{}, fmt.Errorf("yandex captcha encountered for this IP")
	}

	htmlBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return YandexDocsInfo{}, err
	}
	html := string(htmlBytes)

	var cookies []string
	if extraCookie := resolveYandexCookie(YandexCookie); extraCookie != "" {
		cookies = append(cookies, extraCookie)
	}
	for _, c := range resp.Cookies() {
		cookies = append(cookies, fmt.Sprintf("%s=%s", c.Name, c.Value))
	}

	re := regexp.MustCompile(`<script[^>]*id="client-config"[^>]*>(.*?)</script>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return YandexDocsInfo{}, fmt.Errorf("client-config not found in page (check URL and access permissions)")
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &config); err != nil {
		return YandexDocsInfo{}, fmt.Errorf("invalid json in client-config: %w", err)
	}

	officeAction, ok := config["officeActionData"].(map[string]interface{})
	if !ok || officeAction == nil {
		return YandexDocsInfo{}, fmt.Errorf("officeActionData not found (open document in browser and enable classic editor)")
	}

	editorConfigRaw, ok := officeAction["editor_config"].(map[string]interface{})
	if !ok || editorConfigRaw == nil {
		return YandexDocsInfo{}, fmt.Errorf("editor_config not found (classic editor required)")
	}

	balancerURL, _ := officeAction["balancer_url"].(string)
	if balancerURL == "" {
		balancerURL = "https://docs.yandex.net"
	}
	host := strings.TrimPrefix(balancerURL, "https://")
	host = strings.TrimPrefix(host, "http://")

	document, ok := editorConfigRaw["document"].(map[string]interface{})
	if !ok || document == nil {
		return YandexDocsInfo{}, fmt.Errorf("document metadata nil")
	}

	perms, _ := document["permissions"].(map[string]interface{})
	if perms == nil {
		perms = make(map[string]interface{})
	}

	token, _ := editorConfigRaw["token"].(string)
	docKey, _ := document["key"].(string)

	return YandexDocsInfo{
		CookieStr:   strings.Join(cookies, "; "),
		Token:       token,
		DocID:       docKey,
		Origin:      balancerURL,
		Host:        host,
		WsURL:       fmt.Sprintf("wss://%s/2024.1.1-375/doc/%s/c/?EIO=4&transport=websocket", host, docKey),
		Permissions: perms,
		OpenCmd: map[string]interface{}{
			"c":      "open",
			"id":     docKey,
			"userid": userID,
			"format": document["fileType"],
			"url":    document["url"],
			"title":  document["title"],
			"lcid":   25,
		},
	}, nil
}

func randUserID() string {
	return fmt.Sprintf("%010d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1000000000))
}
