//go:build linux

package svcmgr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var ErrRootRequired = errors.New("this command must be run as root (use sudo)")

const (
	unitDir = "/etc/systemd/system"
	envDir  = "/etc/pangolin"
)

func unitPath(name string) string {
	return filepath.Join(unitDir, name+".service")
}

func envPath(name string) string {
	return filepath.Join(envDir, name+".env")
}

// CheckSupported returns an error if the process lacks the permissions
// needed to manage system services, or systemd isn't available.
func CheckSupported() error {
	if os.Geteuid() != 0 {
		return ErrRootRequired
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return errors.New("systemctl was not found; is systemd installed?")
	}
	return nil
}

// Install writes the unit (and environment file, if any), reloads systemd,
// and enables + starts the service immediately.
func Install(spec Spec) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	if len(spec.EnvVars) > 0 {
		if err := os.MkdirAll(envDir, 0o755); err != nil {
			return fmt.Errorf("failed to create %s: %w", envDir, err)
		}

		keys := make([]string, 0, len(spec.EnvVars))
		for k := range spec.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var b strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&b, "%s=%s\n", k, spec.EnvVars[k])
		}

		if err := os.WriteFile(envPath(spec.Name), []byte(b.String()), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", envPath(spec.Name), err)
		}
	}

	if err := os.WriteFile(unitPath(spec.Name), []byte(buildUnitFile(spec, executable)), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", unitPath(spec.Name), err)
	}

	if err := runSystemctl("daemon-reload"); err != nil {
		return err
	}

	return runSystemctl("enable", "--now", spec.Name)
}

func buildUnitFile(spec Spec, executable string) string {
	execStart := systemdQuote(executable)
	for _, a := range spec.Args {
		execStart += " " + systemdQuote(a)
	}

	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s\n", spec.Description)
	b.WriteString("Wants=network-online.target\n")
	b.WriteString("After=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("User=root\n")
	b.WriteString("Group=root\n")
	if len(spec.EnvVars) > 0 {
		fmt.Fprintf(&b, "EnvironmentFile=%s\n", envPath(spec.Name))
	}
	fmt.Fprintf(&b, "ExecStart=%s\n", execStart)
	b.WriteString("Restart=always\n")
	b.WriteString("RestartSec=2\n")
	b.WriteString("UMask=0077\n")
	b.WriteString("PrivateTmp=true\n\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String()
}

// systemdQuote quotes a token for use in a unit file's ExecStart= line if
// it contains characters that would otherwise be treated as word
// separators (systemd.service(5) uses C-style quoting).
func systemdQuote(s string) string {
	if strings.ContainsAny(s, " \t\"'") {
		return strconv.Quote(s)
	}
	return s
}

// Uninstall stops and disables the service, then removes its unit and
// environment file. It does not fail if the service was never installed.
func Uninstall(name string) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	// Best-effort: the unit may not exist or may already be stopped.
	_ = runSystemctl("disable", "--now", name)

	if err := os.Remove(unitPath(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove unit file: %w", err)
	}
	if err := os.Remove(envPath(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove environment file: %w", err)
	}

	return runSystemctl("daemon-reload")
}

// Status returns a human-readable summary of the service's current state,
// including recent log output.
func Status(name string) (string, error) {
	if err := CheckSupported(); err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Active:  %s\n", systemctlOutput("is-active", name))
	fmt.Fprintf(&b, "Enabled: %s\n", systemctlOutput("is-enabled", name))

	if out, _ := exec.Command("systemctl", "status", name, "--no-pager", "-l").CombinedOutput(); len(out) > 0 {
		b.WriteString("\n")
		b.Write(out)
	}

	return b.String(), nil
}

// Follow streams the service's journal to stdout until interrupted
// (equivalent to `journalctl -u <name> -f`), optionally showing `lines` of
// prior output first.
func Follow(name string, lines int) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	args := []string{"-u", name, "-f"}
	if lines > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", lines))
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runSystemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func systemctlOutput(args ...string) string {
	out, _ := exec.Command("systemctl", args...).CombinedOutput()
	return strings.TrimSpace(string(out))
}
