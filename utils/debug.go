package utils

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

var (
	debugLog    *log.Logger = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	verbose     atomic.Bool
	logMu       sync.Mutex
	logCallback func(string)
)

func SetLogCallback(cb func(string)) {
	logMu.Lock()
	defer logMu.Unlock()
	logCallback = cb
}

func EnableDebug() {
	verbose.Store(true)
	logMu.Lock()
	debugLog = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
	logMu.Unlock()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
}

func Log(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logMu.Lock()
	l := debugLog
	cb := logCallback
	logMu.Unlock()

	if l != nil {
		l.Output(2, msg)
	}
	if cb != nil {
		cb(msg)
	}
}

func Debugf(format string, args ...interface{}) {
	if verbose.Load() {
		Log(format, args...)
	}
}

func IsVerbose() bool {
	return verbose.Load()
}

// SetDebug toggles verbose logging at runtime (off = Debugf becomes a no-op).
func SetDebug(on bool) {
	if on {
		EnableDebug()
		return
	}
	verbose.Store(false)
}

// SafeGo runs fn in a new goroutine, recovering from any panic so a crash in
// one worker cannot take down the whole process (critical when this code runs
// embedded as a library inside a mobile app or desktop GUI).
func SafeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				Debugf("[PANIC] recovered in %s: %v", name, r)
			}
		}()
		fn()
	}()
}
