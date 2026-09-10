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

