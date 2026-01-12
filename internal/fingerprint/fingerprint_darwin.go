package fingerprint

import (
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
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

	deviceModel, serialNumber := getDeviceModelAndSerialNumber()

	return &Fingerprint{
		Username:      username,
		Hostname:      hostname,
		Platform:      "macos",
		OSVersion:     osVersion,
		KernelVersion: kernelVersion,
		Architecture:  architecture,
		DeviceModel:   deviceModel,
		SerialNumber:  serialNumber,
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

func getDeviceModelAndSerialNumber() (string, string) {
	type spHardwareOutput struct {
		Output struct {
			SerialNumber string `json:"serial_number"`
			MachineModel string `json:"machine_model"`
		} `json:"SPHardwareDataType"`
	}

	cmd := exec.Command("system_profiler", "SPHardwareDataType", "-json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", ""
	}

	var jsonOutput spHardwareOutput
	if err := json.Unmarshal(output, &jsonOutput); err != nil {
		return "", ""
	}

	return jsonOutput.Output.MachineModel, jsonOutput.Output.SerialNumber
}
