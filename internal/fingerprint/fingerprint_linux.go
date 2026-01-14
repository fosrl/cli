package fingerprint

import (
	"os"
	"os/exec"
	"os/user"
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
		Username:      username,
		Hostname:      hostname,
		Platform:      "linux",
		OSVersion:     detectOSVersion(),
		KernelVersion: kernelVersion,
		Architecture:  architecture,
		DeviceModel:   deviceModel,
		SerialNumber:  serialNumber,
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

	if output, err := exec.Command("uname", "-srv").CombinedOutput(); err == nil {
		return strings.TrimSpace(string(output))
	}

	return ""
}

func detectReleaseFromOSRelease(data []byte) string {
	var name, version string

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		if after, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(after, `"`)
		}
		if after, ok := strings.CutPrefix(line, "VERSION="); ok {
			version = strings.Trim(after, `"`)
		}
	}

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
