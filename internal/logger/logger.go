package logger

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color represents a lipgloss color ID
type Color string

const (
	// Standard colors
	ColorInfo    Color = "6"   // Cyan (ANSI 36)
	ColorDebug   Color = "248" // Light gray (ANSI 90)
	ColorSuccess Color = "46"  // Bright green (ANSI 32)
	ColorWarning Color = "220" // Yellow/Orange (ANSI 33)
	ColorError   Color = "1"   // Red (ANSI 31)

	// Gray scale
	ColorDarkGray  Color = "240" // Dark gray
	ColorLightGray Color = "248" // Light gray
)

// String returns the color ID as a string
func (c Color) String() string {
	return string(c)
}

// LogLevel represents the logging level
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
)

var (
	// Icons
	// iconInfo    = "ℹ"
	iconDebug = "⚙"
	// iconSuccess = "✓"
	// iconWarning = "⚠"
	// iconError   = "✗"

	// Color styles using lipgloss
	// colorInfoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorInfo))
	colorDebugStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDebug))
	// colorSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess))
	// colorWarningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorWarning))
	// colorErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
)

// Logger is a simple logger wrapper around fmt.Printf
type Logger struct {
	level LogLevel
}

var globalLogger *Logger

// InitLogger initializes the global logger with log level from viper config
func InitLogger(level LogLevel) {
	globalLogger = &Logger{
		level: level,
	}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	if globalLogger == nil {
		InitLogger(LogLevelInfo)
	}
	return globalLogger
}

// Info logs an info message
func (l *Logger) Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Debug logs a debug message (only if log level is debug)
func (l *Logger) Debug(format string, args ...any) {
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
func (l *Logger) Success(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// icon := colorSuccessStyle.Render(iconSuccess)
	fmt.Printf("%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Error logs an error message
func (l *Logger) Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// icon := colorErrorStyle.Render(iconError)
	fmt.Fprintf(os.Stderr, "%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Fprintln(os.Stderr)
	}
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// icon := colorWarningStyle.Render(iconWarning)
	fmt.Printf("%s", message)
	if !strings.HasSuffix(message, "\n") {
		fmt.Println()
	}
}

// Package-level convenience functions that use the global logger

// Info logs an info message using the global logger
func Info(format string, args ...any) {
	GetLogger().Info(format, args...)
}

// Debug logs a debug message using the global logger
func Debug(format string, args ...any) {
	GetLogger().Debug(format, args...)
}

// Success logs a success message using the global logger
func Success(format string, args ...any) {
	GetLogger().Success(format, args...)
}

// Error logs an error message using the global logger
func Error(format string, args ...any) {
	GetLogger().Error(format, args...)
}

// Warning logs a warning message using the global logger
func Warning(format string, args ...any) {
	GetLogger().Warning(format, args...)
}
