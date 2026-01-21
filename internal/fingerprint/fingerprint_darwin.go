package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"runtime"
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
	if output, err := exec.Command("uname", "-m").CombinedOutput(); err == nil {
		architecture = strings.TrimSpace(string(output))
	}

	var deviceModel, serialNumber string

	systemProfilerOutput := RunMacOSSystemProfiler()
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
		if len(matches) > 1 {
			// matches[0] is the full match, matches[1] is the captured group (the number)
			if n, _ := strconv.ParseInt(matches[1], 10, 64); n > 0 {
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
	if output, err := exec.Command("/usr/libexec/ApplicationFirewall/socketfilterfw", "--getglobalstate").CombinedOutput(); err == nil {
		statusStr := strings.ToLower(string(output))
		firewallEnabled = strings.Contains(statusStr, "enabled")
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

func computePlatformFingerprint(hw *SPHardwareOutput) string {
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

type SPHardwareOutput struct {
	MachineName  string `json:"machine_name"`
	SerialNumber string `json:"serial_number"`
	MachineModel string `json:"machine_model"`
	PlatformUUID string `json:"platform_UUID"`
}

// Run the system_profiler command on macOS and
// get the SPHardwareDataType.
// Returns nil if unsuccessful or on an unsupported
// platform.
func RunMacOSSystemProfiler() *SPHardwareOutput {
	type outerType struct {
		Output []SPHardwareOutput `json:"SPHardwareDataType"`
	}

	cmd := exec.Command("system_profiler", "SPHardwareDataType", "-json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil
	}

	var jsonOutput outerType
	if err := json.Unmarshal(output, &jsonOutput); err != nil {
		return nil
	}

	if len(jsonOutput.Output) == 0 {
		return nil
	}
	return &jsonOutput.Output[0]
}

func GetDeviceName() string {
	hw := RunMacOSSystemProfiler()
	if hw == nil {
		return "macOS"
	}

	return hw.MachineName
}
