package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fosrl/cli/internal/config"
	"github.com/spf13/viper"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
)

var (
	// Icons
	iconInfo    = "ℹ"
	iconDebug   = "⚙"
	iconSuccess = "✓"
	iconWarning = "⚠"
	iconError   = "✗"

	// Color styles using lipgloss
	colorInfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo))
	colorDebugStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDebug))
	colorSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	colorWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
	colorErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
)

// Logger is a simple logger wrapper around fmt.Printf
type Logger struct {
	level LogLevel
}

var globalLogger *Logger

// InitLogger initializes the global logger with log level from viper config
func InitLogger() {
	levelStr := viper.GetString("log_level")
	if levelStr == "" {
		levelStr = string(LogLevelInfo)
	}

	levelStr = strings.ToLower(strings.TrimSpace(levelStr))
	level := LogLevel(levelStr)

	// Validate log level, default to info if invalid
	if level != LogLevelDebug && level != LogLevelInfo {
		level = LogLevelInfo
	}

	globalLogger = &Logger{
		level: level,
	}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		InitLogger()
	}
	return globalLogger
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Debug logs a debug message (only if log level is debug)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level != LogLevelDebug {
		return
	}
	message := fmt.Sprintf(format, args...)
	icon := colorDebugStyle.Render(iconDebug)
	fmt.Printf("%s %s", icon, message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Success logs a success message
func (l *Logger) Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	// icon := colorSuccessStyle.Render(iconSuccess)
	fmt.Printf("%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	// icon := colorErrorStyle.Render(iconError)
	fmt.Fprintf(os.Stderr, "%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Fprintln(os.Stderr)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	// icon := colorWarningStyle.Render(iconWarning)
	fmt.Printf("%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Package-level convenience functions that use the global logger

// Info logs an info message using the global logger
func Info(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

// Debug logs a debug message using the global logger
func Debug(format string, args ...interface{}) {
	GetLogger().Debug(format, args...)
}

// Success logs a success message using the global logger
func Success(format string, args ...interface{}) {
	GetLogger().Success(format, args...)
}

// Error logs an error message using the global logger
func Error(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

// Warning logs a warning message using the global logger
func Warning(format string, args ...interface{}) {
	GetLogger().Warning(format, args...)
}

// GetDefaultLogPath returns the default log file path for client logs
func GetDefaultLogPath() string {
	pangolinDir, err := config.GetPangolinConfigDir()
	if err != nil {
		return "/tmp/olm.log"
	}
	// Ensure logs subdirectory exists
	logsDir := filepath.Join(pangolinDir, "logs")
	os.MkdirAll(logsDir, 0o755)
	return filepath.Join(logsDir, "client.log")
}
