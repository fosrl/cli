package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/fosrl/cli/internal/utils"
)

func GatherFingerprintInfo() *Fingerprint {
	var username string
	if user, err := user.Current(); err != nil {
		username = user.Username
	} else if u := os.Getenv("USER"); u != "" {
		username = u
	}

	hostname, _ := os.Hostname()

	var osVersion string
	if output, err := exec.Command("sw_vers", "-productVersion").CombinedOutput(); err == nil {
		osVersion = strings.TrimSpace(string(output))
	}

	var kernelVersion string
	if output, err := exec.Command("uname", "-r").CombinedOutput(); err == nil {
		kernelVersion = strings.TrimSpace(string(output))
	}

	var architecture string
	if output, err := exec.Command("uname", "-a").CombinedOutput(); err == nil {
		architecture = strings.TrimSpace(string(output))
	}

	var deviceModel, serialNumber string

	systemProfilerOutput := utils.RunMacOSSystemProfiler()
	if systemProfilerOutput != nil {
		deviceModel = systemProfilerOutput.MachineModel
		serialNumber = systemProfilerOutput.SerialNumber
	}

	platformFingerprint := computePlatformFingerprint(systemProfilerOutput)

	return &Fingerprint{
		Username:            username,
		Hostname:            hostname,
		Platform:            "macos",
		OSVersion:           osVersion,
		KernelVersion:       kernelVersion,
		Architecture:        architecture,
		DeviceModel:         deviceModel,
		SerialNumber:        serialNumber,
		PlatformFingerprint: platformFingerprint,
	}
}

func GatherPostureChecks() *PostureChecks {
	var biometricsEnabled bool
	if output, err := exec.Command("bioutil", "-r").CombinedOutput(); err == nil {
		matches := biometricsRegex.FindStringSubmatch(string(output))
		if len(matches) > 0 {
			if n, _ := strconv.ParseInt(matches[0], 10, 64); n > 0 {
				biometricsEnabled = true
			}
		}
	}

	var diskEncrypted bool
	if output, err := exec.Command("fdesetup", "status").CombinedOutput(); err == nil {
		statusStr := strings.ToLower(string(output))
		diskEncrypted = strings.Contains(statusStr, "filevault is on")
	}

	var firewallEnabled bool
	firewallQueryCmd := exec.Command("/usr/bin/defaults", "read", "/Library/Preferences/com.apple.alf", "globalstate")
	if output, err := firewallQueryCmd.CombinedOutput(); err == nil {
		valueStr := strings.TrimSpace(strings.ToLower(string(output)))
		if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
			// 0 = off, 1 = on for specific services, 2 = on for essential services
			firewallEnabled = value != 0
		}
	}

	var autoUpdatesEnabled bool
	if output, err := exec.Command("softwareupdate", "--schedule").CombinedOutput(); err == nil {
		statusStr := strings.ToLower(string(output))
		autoUpdatesEnabled = strings.Contains(statusStr, "on")
	}

	var sipEnabled bool
	if output, err := exec.Command("csrutil", "status").CombinedOutput(); err == nil {
		statusStr := strings.ToLower(string(output))
		sipEnabled = strings.Contains(statusStr, "enabled")
	}

	var gatekeeperEnabled bool
	if output, err := exec.Command("spctl", "--status").CombinedOutput(); err == nil {
		statusStr := strings.ToLower(string(output))
		gatekeeperEnabled = strings.Contains(statusStr, "enabled")
	}

	var firewallStealthMode bool
	firewallStealthModeCmd := exec.Command("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getstealthmode")
	if output, err := firewallStealthModeCmd.CombinedOutput(); err == nil {
		statusStr := strings.ToLower(string(output))
		firewallStealthMode = strings.Contains(statusStr, "is on")
	}

	return &PostureChecks{
		BiometricsEnabled:  biometricsEnabled,
		DiskEncrypted:      diskEncrypted,
		FirewallEnabled:    firewallEnabled,
		AutoUpdatesEnabled: autoUpdatesEnabled,
		// T2 and secure facilities are always available on macOS
		TpmAvailable: true,

		MacOSSIPEnabled:          sipEnabled,
		MacOSGatekeeperEnabled:   gatekeeperEnabled,
		MacOSFirewallStealthMode: firewallStealthMode,
	}
}

var biometricsRegex = regexp.MustCompile(`Biometrics for unlock:\s*(\d+)`)

func computePlatformFingerprint(hw *utils.SPHardwareOutput) string {
	if hw == nil {
		return ""
	}

	var parts []string

	parts = append(parts, runtime.GOOS, runtime.GOARCH)

	if hw.MachineModel != "" {
		parts = append(parts, normalize(hw.MachineModel))
	}

	if hw.SerialNumber != "" {
		parts = append(parts, normalize(hw.SerialNumber))
	}

	if hw.PlatformUUID != "" {
		parts = append(parts, normalize(hw.PlatformUUID))
	}

	raw := strings.Join(parts, "|")
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}
