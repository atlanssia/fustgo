package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Logger provides structured logging capabilities
type Logger struct {
	level      Level
	output     *os.File
	prefix     string
	enableFile bool
	filePath   string
}

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of the log level
func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// NewLogger creates a new logger instance
func NewLogger(prefix string, enableFile bool, filePath string) (*Logger, error) {
	logger := &Logger{
		level:      LevelInfo,
		prefix:     prefix,
		enableFile: enableFile,
		filePath:   filePath,
		output:     os.Stdout,
	}

	if enableFile && filePath != "" {
		if err := logger.initFileOutput(); err != nil {
			return nil, err
		}
	}

	return logger, nil
}

// initFileOutput initializes file-based logging
func (l *Logger) initFileOutput() error {
	dir := filepath.Dir(l.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.output = file
	return nil
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// log writes a log message
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] [%s] [%s] %s\n", timestamp, level, l.prefix, message)

	if l.enableFile && l.output != os.Stdout {
		l.output.WriteString(logLine)
	} else {
		fmt.Print(logLine)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LevelFatal, format, args...)
	os.Exit(1)
}

// Close closes the logger and releases resources
func (l *Logger) Close() error {
	if l.output != nil && l.output != os.Stdout {
		return l.output.Close()
	}
	return nil
}

// Global logger instance
var defaultLogger *Logger

func init() {
	var err error
	defaultLogger, err = NewLogger("fustgo", false, "")
	if err != nil {
		log.Fatal("Failed to initialize default logger:", err)
	}
}

// SetDefaultLogger sets the default logger
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// Convenience functions for default logger
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}
