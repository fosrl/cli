package utils

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

// GetDeviceName returns a human-readable device name
func GetDeviceName() string {
	isLaptop := isLaptop()

	switch runtime.GOOS {
	case "linux":
		osName := getLinuxOSName()
		return formatDeviceName(osName, isLaptop)

	case "windows":
		return formatDeviceName("Windows", isLaptop)

	case "darwin":
		hw := RunMacOSSystemProfiler()
		if hw == nil {
			return "macOS"
		}
		return hw.MachineName

	default:
		return "Unknown Device"
	}
}

func formatDeviceName(osName string, isLaptop bool) string {
	if isLaptop {
		return osName + " Laptop"
	}
	return osName + " Desktop"
}

func getLinuxOSName() string {
	osRelease, err := ParseOSRelease()
	if err != nil {
		return "Linux"
	}

	if name, ok := osRelease["NAME"]; ok {
		return name
	}

	return "Linux"
}

func isLaptop() bool {
	switch runtime.GOOS {
	case "linux":
		matches, err := filepath.Glob("/sys/class/power_supply/BAT*")
		if err != nil {
			return false
		}
		return len(matches) > 0
	case "windows":
		// TODO: implement
		return false
	case "darwin":
		out, err := exec.Command("ioreg", "-r", "-c", "AppleSmartBattery").Output()
		if err != nil {
			return false
		}
		return len(out) > 0
	default:
		return false
	}
}
