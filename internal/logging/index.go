package logging

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// lumberjack names rotated backups "<name>-<time>.log", plus ".gz" when compressed.
const backupTimeFormat = "2006-01-02T15-04-05.000"

// ClearLogs removes the log file of name and its rotated backups.
// Other clients whose names share the prefix (e.g. "cpu-2" for "cpu") are left alone.
func ClearLogs(name string) error {
	if err := os.Remove(LogPath(name)); err != nil && !os.IsNotExist(err) {
		return err
	}
	backups, err := filepath.Glob(filepath.Join(logDir, name+"-*.log*"))
	if err != nil {
		return err
	}
	for _, b := range backups {
		ts := strings.TrimPrefix(filepath.Base(b), name+"-")
		ts, _, _ = strings.Cut(ts, ".log")
		if _, err := time.Parse(backupTimeFormat, ts); err != nil {
			continue // not a backup of name, e.g. "cpu-2.log"
		}
		if err := os.Remove(b); err != nil && !os.IsNotExist(err) {
			return err
		}
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
