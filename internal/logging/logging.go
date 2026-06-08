package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	appconfig "main/internal/config"
)

var (
	mu       sync.Mutex
	logPath  string
	logFile  *os.File
	initialized bool
)

func Init() error {
	mu.Lock()
	defer mu.Unlock()
	if initialized {
		return nil
	}
	if err := appconfig.EnsureAppDataDir(); err != nil {
		return err
	}
	dir := filepath.Join(appconfig.AppDataDir(), "logs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	logPath = filepath.Join(dir, "app.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	logFile = f
	initialized = true
	writeLocked("INFO", "Logging initialized")
	return nil
}

func Path() string {
	if logPath != "" {
		return logPath
	}
	return filepath.Join(appconfig.AppDataDir(), "logs", "app.log")
}

func Info(format string, args ...interface{}) {
	write("INFO", format, args...)
}

func Error(format string, args ...interface{}) {
	write("ERROR", format, args...)
}

func write(level, format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	writeLocked(level, fmt.Sprintf(format, args...))
}

func writeLocked(level, message string) {
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), level, message)
	if logFile != nil {
		_, _ = logFile.WriteString(line)
	}
}

func LogPanic(source string, recovered interface{}) string {
	stack := string(debug.Stack())
	msg := fmt.Sprintf("PANIC in %s: %v\n%s", source, recovered, stack)
	mu.Lock()
	defer mu.Unlock()
	writeLocked("PANIC", msg)
	return msg
}

func InstallGlobalPanicHandler() {
	_ = Init()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				LogPanic("global goroutine", r)
			}
		}()
	}()
}
