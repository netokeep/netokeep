package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
)

var homeDir, _ = os.UserHomeDir()
var logDir = filepath.Join(homeDir, ".local", "state", "netokeep", "logs")

func InitLogging(name string) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logPath := filepath.Join(logDir, name+".log")
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
