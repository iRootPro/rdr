// Package rlog provides a simple file logger for rdr.
// All components write to a single <home>/rdr.log file.
package rlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu      sync.Mutex
	logDir  string
	logFile *os.File
)

// Init sets the log directory and opens the log file.
// Call once from main/ui init.
func Init(dir string) {
	mu.Lock()
	defer mu.Unlock()
	logDir = dir
	path := filepath.Join(dir, "rdr.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	logFile = f
}

// Close flushes and closes the log file.
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// Log writes a timestamped message to the log file.
func Log(component, msg string) {
	mu.Lock()
	defer mu.Unlock()
	if logFile == nil {
		return
	}
	ts := time.Now().Format("15:04:05")
	fmt.Fprintf(logFile, "[%s] %s: %s\n", ts, component, msg)
}

// Logf writes a formatted timestamped message.
func Logf(component, format string, args ...any) {
	Log(component, fmt.Sprintf(format, args...))
}

// Error writes an error entry.
func Error(component string, err error) {
	if err != nil {
		Log(component, "ERROR: "+err.Error())
	}
}

// LogPath returns the path to the log file.
func LogPath() string {
	return filepath.Join(logDir, "rdr.log")
}
