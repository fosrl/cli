package fingerprint

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func GatherFingerprintInfo() *Fingerprint {
	var username string

	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		username = sudoUser
	} else if u, err := user.Current(); err == nil {
		username = u.Username
	} else if u := os.Getenv("USER"); u != "" {
		username = u
	}

	hostname, _ := os.Hostname()

	var kernelVersion string
	if output, err := exec.Command("uname", "-r").CombinedOutput(); err == nil {
		kernelVersion = strings.TrimSpace(string(output))
	}

	var architecture string
	if output, err := exec.Command("uname", "-m").CombinedOutput(); err == nil {
		architecture = strings.TrimSpace(string(output))
	}

	deviceModel, serialNumber := getLinuxDeviceModelAndSerialNumber()

	return &Fingerprint{
		Username:            username,
		Hostname:            hostname,
		Platform:            "linux",
		OSVersion:           detectOSVersion(),
		KernelVersion:       kernelVersion,
		Architecture:        architecture,
		DeviceModel:         deviceModel,
		SerialNumber:        serialNumber,
		PlatformFingerprint: computeHwFingerprint(),
	}
}

func GatherPostureChecks() *PostureChecks {
	// Check for LUKS devices. This can be improved later on
	// to intelligently look for the mounted root device and
	// see if it is indeed encrypted.
	var diskEncrypted bool
	if output, err := exec.Command("lsblk", "-o", "NAME,TYPE").CombinedOutput(); err == nil {
		for line := range strings.SplitSeq(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[1] == "crypt" {
				diskEncrypted = true
				break
			}
		}
	}

	var appArmorEnabled bool
	if output, err := os.ReadFile("/sys/module/apparmor/parameters/enabled"); err == nil {
		appArmorEnabled = strings.Contains(strings.ToLower(string(output)), "y")
	}

	var selinuxEnabled bool
	if output, err := exec.Command("getenforce").CombinedOutput(); err == nil {
		selinuxEnabled = strings.TrimSpace(strings.ToLower(string(output))) == "enforcing"
	}

	return &PostureChecks{
		// TODO: implement heuristic for checking for biometrics on Linux
		BiometricsEnabled: false,
		DiskEncrypted:     diskEncrypted,
		FirewallEnabled:   isFirewallEnabled(),
		// TODO: implement heuristic for checking for auto-updates
		AutoUpdatesEnabled: false,
		TpmAvailable:       tpmAvailable(),

		LinuxAppArmorEnabled: appArmorEnabled,
		LinuxSELinuxEnabled:  selinuxEnabled,
	}
}

func detectOSVersion() string {
	if _, err := exec.LookPath("lsb_release"); err == nil {
		if output, err := exec.Command("lsb_release", "-ds").CombinedOutput(); err == nil {
			return strings.Trim(strings.TrimSpace(string(output)), `"`)
		}
	}

	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		if pretty := detectReleaseFromOSRelease(data); pretty != "" {
			return pretty
		}
	}

	if output, err := exec.Command("uname", "-sr").CombinedOutput(); err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}

func detectReleaseFromOSRelease(data []byte) string {
	osRelease, err := ParseOSRelease()
	if err != nil {
		return ""
	}

	name, _ := osRelease["NAME"]
	version, _ := osRelease["VERSION"]

	if name != "" && version != "" {
		return name + " " + version
	}

	return ""
}

func isFirewallEnabled() bool {
	if _, err := exec.LookPath("ufw"); err == nil {
		if output, err := exec.Command("ufw", "status").CombinedOutput(); err == nil {
			if strings.Contains(strings.ToLower(string(output)), "status: active") {
				return true
			}
		}
	}

	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		if output, err := exec.Command("firewall-cmd", "--state").CombinedOutput(); err == nil {
			if strings.TrimSpace(string(output)) == "running" {
				return true
			}
		}
	}

	if _, err := exec.LookPath("nft"); err == nil {
		if output, err := exec.Command("nft", "list", "ruleset").CombinedOutput(); err == nil {
			if strings.TrimSpace(string(output)) != "" {
				return true
			}
		}
	}

	if _, err := exec.LookPath("iptables"); err == nil {
		if output, err := exec.Command("iptables", "-S").CombinedOutput(); err == nil {
			lines := strings.SplitSeq(string(output), "\n")
			for line := range lines {
				if strings.HasPrefix(line, "-A") {
					return true
				}
			}
		}
	}

	return false
}

func getLinuxDeviceModelAndSerialNumber() (string, string) {
	var model, serial string

	if output, err := os.ReadFile("/sys/devices/virtual/dmi/id/product_name"); err == nil {
		model = strings.TrimSpace(string(output))
	}

	if output, err := os.ReadFile("/sys/devices/virtual/dmi/id/product_serial"); err == nil {
		serial = strings.TrimSpace(string(output))
	}

	return model, serial
}

func tpmAvailable() bool {
	if _, err := os.Stat("/dev/tpm0"); err == nil {
		return true
	}

	return false
}

func readFileAndTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func computeHwFingerprint() string {
	var parts []string

	parts = append(parts, runtime.GOARCH, runtime.GOOS)

	parts = append(parts, cpuFingerprint())

	dmiPaths := []string{
		"/sys/devices/virtual/dmi/id/product_uuid",
		"/sys/devices/virtual/dmi/id/board_serial",
		"/sys/devices/virtual/dmi/id/product_name",
		"/sys/devices/virtual/dmi/id/sys_vendor",
	}
	for _, p := range dmiPaths {
		parts = append(parts, readFileAndTrim(p))
	}

	// Normalize
	var cleaned []string
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}

	sort.Strings(cleaned)

	joined := strings.Join(cleaned, "|")

	hash := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(hash[:])
}

// Extracts stable, per-CPU fields from /proc/cpuinfo.
// Returns a deterministic, normalized string.
func cpuFingerprint() string {
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")

	allowed := map[string]string{
		"vendor_id":       "vendor",
		"model name":      "model_name",
		"cpu family":      "family",
		"model":           "model",
		"stepping":        "stepping",
		"cpu cores":       "cores",
		"siblings":        "siblings",
		"cpu implementer": "implementer",
		"cpu part":        "part",
		"cpu revision":    "revision",
	}

	values := make(map[string]string)

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.ToLower(strings.TrimSpace(parts[1]))

		if normKey, ok := allowed[key]; ok {
			// Take the first occurrence only (per CPU differences are noise)
			if _, exists := values[normKey]; !exists && val != "" {
				values[normKey] = val
			}
		}
	}

	order := []string{
		"vendor",
		"model_name",
		"family",
		"model",
		"stepping",
		"cores",
		"siblings",
		"implementer",
		"part",
		"revision",
	}

	var out []string
	for _, k := range order {
		if v, ok := values[k]; ok {
			out = append(out, k+"="+v)
		}
	}

	return strings.Join(out, "|")
}

func GetDeviceName() string {
	var osName string = "Linux"

	if osRelease, err := ParseOSRelease(); err == nil {
		if name, ok := osRelease["NAME"]; ok {
			osName = name
		}
	}

	var isLaptop bool
	if matches, err := filepath.Glob("/sys/class/power_supply/BAT*"); err == nil {
		isLaptop = len(matches) > 0
	}

	return formatDeviceName(osName, isLaptop)
}

func ParseOSRelease() (map[string]string, error) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := strings.Trim(parts[1], `"`)
		values[key] = value
	}

	return values, nil
}
