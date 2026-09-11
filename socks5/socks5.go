package socks5

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"universal-bypass-tool/utils"
)

type Dialer interface {
	DialTCP(address string) (net.Conn, error)
}

type SOCKS5Server struct {
	listenAddr string
	dialer     Dialer
	listener   net.Listener
	mu         sync.Mutex
	closed     bool
}

func NewSOCKS5Server(addr string, dialer Dialer) *SOCKS5Server {
	return &SOCKS5Server{listenAddr: addr, dialer: dialer}
}

func (s *SOCKS5Server) Start() error {
	listener, err := net.Listen("tcp", s.listenAddr)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.listener = listener
	s.closed = false
	s.mu.Unlock()
	defer listener.Close()

	utils.Debugf("[SOCKS5] Listening on %s", s.listenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return nil
			}
			utils.Debugf("[SOCKS5] Accept error: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *SOCKS5Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *SOCKS5Server) handleConnection(clientConn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			utils.Debugf("[SOCKS5] Recovered from panic in handleConnection: %v", r)
		}
	}()
	defer clientConn.Close()

	buf := make([]byte, 4096)
	n, err := clientConn.Read(buf)
	if err != nil || n < 2 {
		return
	}

	if buf[0] != 0x05 {
		s.handleHTTPRequest(clientConn, buf[:n])
		return
	}

	clientConn.Write([]byte{0x05, 0x00})

	n, err = clientConn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	cmd := buf[1]
	if cmd == 0x03 {
		// SOCKS5 UDP ASSOCIATE (used by tun2socks for UDP/DNS traffic)
		s.handleUDPAssociate(clientConn)
		return
	}

	if cmd != 0x01 || n < 10 {
		return
	}

	var targetAddr string
	switch buf[3] {
	case 0x01:
		targetAddr = fmt.Sprintf("%d.%d.%d.%d:%d",
			buf[4], buf[5], buf[6], buf[7],
			uint16(buf[8])<<8|uint16(buf[9]))
	case 0x03:
		domainLen := int(buf[4])
		if n < 5+domainLen+2 {
			return
		}
		targetAddr = fmt.Sprintf("%s:%d",
			string(buf[5:5+domainLen]),
			uint16(buf[5+domainLen])<<8|uint16(buf[5+domainLen+1]))
	default:
		return
	}

	utils.Debugf("[SOCKS5] CONNECT %s", targetAddr)

	targetConn, err := s.dialer.DialTCP(targetAddr)
	if err != nil {
		utils.Debugf("[SOCKS5] Dial failed: %v", err)
		clientConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return
	}
	defer targetConn.Close()

	clientConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer targetConn.Close()
		io.Copy(targetConn, clientConn)
	}()

	go func() {
		defer wg.Done()
		defer clientConn.Close()
		io.Copy(clientConn, targetConn)
	}()

	wg.Wait()
}

func (s *SOCKS5Server) handleHTTPRequest(clientConn net.Conn, initialData []byte) {
	reqStr := string(initialData)
	lines := strings.Split(reqStr, "\r\n")
	if len(lines) == 0 {
		return
	}
	fields := strings.Fields(lines[0])
	if len(fields) < 2 {
		return
	}

	method := strings.ToUpper(fields[0])
	target := fields[1]

	var targetAddr string
	if method == "CONNECT" {
		targetAddr = target
		if !strings.Contains(targetAddr, ":") {
			targetAddr += ":443"
		}
	} else {
		host := ""
		for _, line := range lines[1:] {
			if strings.HasPrefix(strings.ToLower(line), "host:") {
				host = strings.TrimSpace(line[5:])
				break
			}
		}
		if host == "" {
			if strings.HasPrefix(target, "http://") {
				if u, err := url.Parse(target); err == nil {
					host = u.Host
				}
			}
		}
		if host == "" {
			return
		}
		targetAddr = host
		if !strings.Contains(targetAddr, ":") {
			targetAddr += ":80"
		}
	}

	utils.Debugf("[HTTP-Proxy] %s %s", method, targetAddr)

	targetConn, err := s.dialer.DialTCP(targetAddr)
	if err != nil {
		utils.Debugf("[HTTP-Proxy] Dial failed: %v", err)
		if method == "CONNECT" {
			clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		}
		return
	}
	defer targetConn.Close()

	if method == "CONNECT" {
		if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
	} else {
		if _, err := targetConn.Write(initialData); err != nil {
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer targetConn.Close()
		io.Copy(targetConn, clientConn)
	}()

	go func() {
		defer wg.Done()
		defer clientConn.Close()
		io.Copy(clientConn, targetConn)
	}()

	wg.Wait()
}

func (s *SOCKS5Server) handleUDPAssociate(clientConn net.Conn) {
	udpListener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		clientConn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer udpListener.Close()

	lAddr := udpListener.LocalAddr().(*net.UDPAddr)
	port := uint16(lAddr.Port)

	// Send success reply to client
	reply := []byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, byte(port >> 8), byte(port & 0xFF)}
	if _, err := clientConn.Write(reply); err != nil {
		return
	}

	// Read UDP packets in background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				utils.Debugf("[SOCKS5] Recovered from panic in UDP reader: %v", r)
			}
		}()
		buf := make([]byte, 4096)
		for {
			n, srcAddr, err := udpListener.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if n < 10 {
				continue
			}

			// Parse SOCKS5 UDP request: RSV(2) + FRAG(1) + ATYP(1) + ADDR + PORT + DATA
			if buf[2] != 0 {
				continue // Fragmentation not supported
			}
			atyp := buf[3]
			var headerLen int
			var dstIP net.IP
			var dstPort uint16

			switch atyp {
			case 0x01: // IPv4
				headerLen = 10
				dstIP = net.IP(buf[4:8])
				dstPort = binary.BigEndian.Uint16(buf[8:10])
			case 0x03: // Domain
				if n < 5 {
					continue
				}
				domainLen := int(buf[4])
				headerLen = 5 + domainLen + 2
				if n < headerLen {
					continue
				}
				dstPort = binary.BigEndian.Uint16(buf[5+domainLen : headerLen])
			case 0x04: // IPv6
				headerLen = 22
				if n < headerLen {
					continue
				}
				dstPort = binary.BigEndian.Uint16(buf[20:22])
			default:
				continue
			}

			if n <= headerLen {
				continue
			}

			payload := buf[headerLen:n]

			// If it's DNS query (port 53/5353), forward it over TCP via tunnel
			if dstPort == 53 || dstPort == 5353 {
				go func(queryPayload []byte, hdr []byte, clientSrc *net.UDPAddr) {
					defer func() {
						if r := recover(); r != nil {
							utils.Debugf("[SOCKS5] Recovered from panic in DNS forwarder: %v", r)
						}
					}()
					dnsServer := "8.8.8.8:53"
					if dstIP != nil && !dstIP.IsUnspecified() {
						dnsServer = fmt.Sprintf("%s:53", dstIP.String())
					}
					respPayload, err := s.queryDNSOverTCP(dnsServer, queryPayload)
					if err != nil {
						return
					}
					resp := make([]byte, len(hdr)+len(respPayload))
					copy(resp, hdr)
					copy(resp[len(hdr):], respPayload)
					udpListener.WriteToUDP(resp, clientSrc)
				}(append([]byte{}, payload...), append([]byte{}, buf[:headerLen]...), srcAddr)
			}
		}
	}()

	// Hold TCP connection open. When client disconnects, close UDP listener.
	dummy := make([]byte, 1)
	for {
		if _, err := clientConn.Read(dummy); err != nil {
			return
		}
	}
}

func (s *SOCKS5Server) queryDNSOverTCP(dnsServer string, query []byte) ([]byte, error) {
	conn, err := s.dialer.DialTCP(dnsServer)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// DNS over TCP prefixes query with 2-byte length
	tcpMsg := make([]byte, 2+len(query))
	binary.BigEndian.PutUint16(tcpMsg[:2], uint16(len(query)))
	copy(tcpMsg[2:], query)

	if _, err := conn.Write(tcpMsg); err != nil {
		return nil, err
	}

	var lenBuf [2]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return nil, err
	}
	respLen := binary.BigEndian.Uint16(lenBuf[:])

	resp := make([]byte, respLen)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return nil, err
	}

	return resp, nil
}
