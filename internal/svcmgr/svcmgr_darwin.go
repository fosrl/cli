//go:build darwin

package svcmgr

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var ErrRootRequired = errors.New("this command must be run as root (use sudo)")

const (
	launchDaemonDir = "/Library/LaunchDaemons"
	logDir          = "/var/log/pangolin"
)

// label returns the reverse-DNS launchd label for a service, e.g.
// "pangolin-site" -> "net.pangolin.site".
func label(name string) string {
	return "net.pangolin." + strings.TrimPrefix(name, "pangolin-")
}

func plistPath(name string) string {
	return filepath.Join(launchDaemonDir, label(name)+".plist")
}

func logPath(name string) string {
	return filepath.Join(logDir, name+".log")
}

// CheckSupported returns an error if the process lacks the permissions
// needed to manage system daemons, or launchctl isn't available.
func CheckSupported() error {
	if os.Geteuid() != 0 {
		return ErrRootRequired
	}
	if _, err := exec.LookPath("launchctl"); err != nil {
		return errors.New("launchctl was not found")
	}
	return nil
}

// Install writes a LaunchDaemon plist and loads it immediately (it will
// also load automatically on every boot, via RunAtLoad).
func Install(spec Spec) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", logDir, err)
	}

	plist := buildPlist(spec, executable, logPath(spec.Name))
	path := plistPath(spec.Name)

	// 0600: EnvironmentVariables may hold credentials; only root needs to
	// read this (launchd itself runs as root).
	if err := os.WriteFile(path, []byte(plist), 0o600); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	// Unload any stale copy first (e.g. left over from a crashed prior
	// install) so bootstrap doesn't fail with "already bootstrapped".
	_ = runLaunchctl("bootout", "system/"+label(spec.Name))

	if err := runLaunchctl("bootstrap", "system", path); err != nil {
		return err
	}

	return nil
}

func buildPlist(spec Spec, executable, logFile string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")

	fmt.Fprintf(&b, "  <key>Label</key>\n  <string>%s</string>\n", xmlEscape(label(spec.Name)))

	b.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	fmt.Fprintf(&b, "    <string>%s</string>\n", xmlEscape(executable))
	for _, a := range spec.Args {
		fmt.Fprintf(&b, "    <string>%s</string>\n", xmlEscape(a))
	}
	b.WriteString("  </array>\n")

	if len(spec.EnvVars) > 0 {
		keys := make([]string, 0, len(spec.EnvVars))
		for k := range spec.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		b.WriteString("  <key>EnvironmentVariables</key>\n  <dict>\n")
		for _, k := range keys {
			fmt.Fprintf(&b, "    <key>%s</key>\n    <string>%s</string>\n", xmlEscape(k), xmlEscape(spec.EnvVars[k]))
		}
		b.WriteString("  </dict>\n")
	}

	b.WriteString("  <key>RunAtLoad</key>\n  <true/>\n")
	b.WriteString("  <key>KeepAlive</key>\n  <true/>\n")
	fmt.Fprintf(&b, "  <key>StandardOutPath</key>\n  <string>%s</string>\n", xmlEscape(logFile))
	fmt.Fprintf(&b, "  <key>StandardErrorPath</key>\n  <string>%s</string>\n", xmlEscape(logFile))

	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func xmlEscape(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

// Uninstall unloads and removes the LaunchDaemon plist. It does not fail if
// the service was never installed.
func Uninstall(name string) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	// Best-effort: the service may not be loaded.
	_ = runLaunchctl("bootout", "system/"+label(name))

	path := plistPath(name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove %s: %w", path, err)
	}

	return nil
}

// Status returns a human-readable summary of the service's current state.
func Status(name string) (string, error) {
	if err := CheckSupported(); err != nil {
		return "", err
	}

	var b strings.Builder

	if out, err := exec.Command("launchctl", "print", "system/"+label(name)).CombinedOutput(); err == nil {
		b.Write(out)
	} else if out, err := exec.Command("launchctl", "list", label(name)).CombinedOutput(); err == nil {
		b.Write(out)
	} else {
		fmt.Fprintf(&b, "Not loaded (%v)\n", err)
	}

	return b.String(), nil
}

// Follow streams the service's log file to stdout until interrupted.
func Follow(name string, lines int) error {
	if err := CheckSupported(); err != nil {
		return err
	}
	return TailFile(logPath(name), lines)
}

func runLaunchctl(args ...string) error {
	out, err := exec.Command("launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
