package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
)

func logDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "netokeep", "logs")
}

/*
LogPath returns the full path to the log file for a given name.
*/
func LogPath(name string) string {
	return filepath.Join(logDir(), name+".log")
}

func InitLogging(name string) {
	if err := os.MkdirAll(logDir(), 0755); err != nil {
		panic("Failed to create log directory: " + err.Error())
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
