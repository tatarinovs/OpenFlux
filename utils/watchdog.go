package utils

import (
	"net"
	"os"
	"strconv"
	"time"
)

// NotifySystemd sends a state string to systemd via $NOTIFY_SOCKET.
func NotifySystemd(state string) error {
	socketAddr := os.Getenv("NOTIFY_SOCKET")
	if socketAddr == "" {
		return nil
	}

	conn, err := net.Dial("unixgram", socketAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(state))
	return err
}

// StartSystemdWatchdog checks if systemd watchdog is requested ($WATCHDOG_USEC).
// If active, it sends READY=1 and periodically sends WATCHDOG=1 as long as healthy() returns true.
// Returns a stop function.
func StartSystemdWatchdog(healthy func() bool) func() {
	notifySocket := os.Getenv("NOTIFY_SOCKET")
	watchdogUsecStr := os.Getenv("WATCHDOG_USEC")

	if notifySocket == "" {
		return func() {}
	}

	// Send initial READY=1
	_ = NotifySystemd("READY=1\n")
	Debugf("[SYSTEMD] Sent READY=1 to %s", notifySocket)

	if watchdogUsecStr == "" {
		return func() {
			_ = NotifySystemd("STOPPING=1\n")
		}
	}

	usec, err := strconv.ParseInt(watchdogUsecStr, 10, 64)
	if err != nil || usec <= 0 {
		return func() {
			_ = NotifySystemd("STOPPING=1\n")
		}
	}

	interval := time.Duration(usec/2) * time.Microsecond
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}

	Debugf("[SYSTEMD] Watchdog active with keep-alive interval %v (WATCHDOG_USEC=%s)", interval, watchdogUsecStr)

	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				if healthy == nil || healthy() {
					_ = NotifySystemd("WATCHDOG=1\n")
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		close(done)
		_ = NotifySystemd("STOPPING=1\n")
	}
}
