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
	mu   sync.Mutex
	file *os.File
)

func Init() error {
	_ = appconfig.MigrateLegacyAppData()
	_ = appconfig.EnsureAppDataDir()
	logDir := filepath.Join(appconfig.AppDataDir(), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}
	path := filepath.Join(logDir, "app.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	mu.Lock()
	file = f
	mu.Unlock()
	return nil
}

func Path() string {
	return filepath.Join(appconfig.AppDataDir(), "logs", "app.log")
}

func write(level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format(time.RFC3339), level, msg)
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_, _ = file.WriteString(line)
	}
}

func Info(format string, args ...interface{})  { write("INFO", format, args...) }
func Error(format string, args ...interface{}) { write("ERROR", format, args...) }

func InstallGlobalPanicHandler() {}

func LogPanic(context string, r interface{}) string {
	msg := fmt.Sprintf("panic in %s: %v\n%s", context, r, debug.Stack())
	Error("%s", msg)
	return msg
}
