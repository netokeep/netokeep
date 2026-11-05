package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/natefinch/lumberjack"
)

var homeDir, _ = os.UserHomeDir()
var logDir = filepath.Join(homeDir, ".local", "state", "netokeep", "logs")

func InitLogging(name string) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logPath := LogPath(name)
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    1,    // Size of one log file (MB)
		MaxBackups: 1,    // Max number of old log files to keep
		MaxAge:     28,   // Max age of old log files (days)
		Compress:   true, // Compress old log files
	}
	multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
	log.SetOutput(multiWriter)
}

func ClearLogs(name string) error {
	pattern := filepath.Join(logDir, name+"*.log*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, match := range matches {
		_ = os.Remove(match)
	}
	return nil
}

// LogPath returns the current log file of name.
func LogPath(name string) string {
	return filepath.Join(logDir, name+".log")
}

// Tail returns the last n lines of the current log file of name.
// The file is rotated at 1 MB, so reading it whole is cheap.
func Tail(name string, n int) ([]string, error) {
	data, err := os.ReadFile(LogPath(name))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}
