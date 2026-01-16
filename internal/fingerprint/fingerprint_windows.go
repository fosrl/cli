package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

func GatherFingerprintInfo() *Fingerprint {
	var username string
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	hostname, _ := os.Hostname()

	osVersion, kernelVersion := getWindowsVersion()

	deviceModel, serialNumber := getWindowsModelAndSerial()

	return &Fingerprint{
		Username:            username,
		Hostname:            hostname,
		Platform:            "windows",
		OSVersion:           osVersion,
		KernelVersion:       kernelVersion,
		Architecture:        runtime.GOARCH,
		DeviceModel:         deviceModel,
		SerialNumber:        serialNumber,
		PlatformFingerprint: computePlatformFingerprint(),
	}
}

func GatherPostureChecks() *PostureChecks {
	return &PostureChecks{
		BiometricsEnabled:  windowsBiometricsEnabled(),
		DiskEncrypted:      windowsDiskEncrypted(),
		FirewallEnabled:    windowsFirewallEnabled(),
		AutoUpdatesEnabled: windowsAutoUpdatesEnabled(),
		TpmAvailable:       windowsTPMAvailable(),

		WindowsDefenderEnabled: windowsDefenderEnabled(),
	}
}

type rtlOsVersionInfoEx struct {
	dwOSVersionInfoSize uint32
	dwMajorVersion      uint32
	dwMinorVersion      uint32
	dwBuildNumber       uint32
	dwPlatformId        uint32
	szCSDVersion        [128]uint16
}

func getWindowsVersion() (string, string) {
	ntdll := syscall.NewLazyDLL("ntdll.dll")
	proc := ntdll.NewProc("RtlGetVersion")

	var info rtlOsVersionInfoEx
	info.dwOSVersionInfoSize = uint32(unsafe.Sizeof(info))

	_, _, _ = proc.Call(uintptr(unsafe.Pointer(&info)))

	osVersion := strings.TrimSpace(
		strings.Join([]string{
			"Windows",
			strconv.FormatUint(uint64(info.dwMajorVersion), 10),
			strconv.FormatUint(uint64(info.dwMinorVersion), 10),
			"Build",
			strconv.FormatUint(uint64(info.dwBuildNumber), 10),
		}, " "),
	)

	return osVersion, osVersion
}

func getWindowsModelAndSerial() (string, string) {
	model, _ := runPowerShell(`
		(Get-CimInstance Win32_ComputerSystem).Model
	`)

	serial, _ := runPowerShell(`
		(Get-CimInstance Win32_BIOS).SerialNumber
	`)

	return model, serial
}

func windowsDiskEncrypted() bool {
	out, err := runPowerShell(`
		(Get-BitLockerVolume -MountPoint $env:SystemDrive).VolumeStatus
	`)
	if err != nil {
		return false
	}

	return strings.EqualFold(out, "FullyEncrypted")
}

func windowsFirewallEnabled() bool {
	out, err := runPowerShell(`
		(Get-NetFirewallProfile | Where-Object {$_.Enabled -eq "True"}).Count
	`)
	if err != nil {
		return false
	}

	return out != "" && out != "0"
}

func windowsDefenderEnabled() bool {
	out, err := runPowerShell(`
		(Get-Service WinDefend).Status
	`)
	if err != nil {
		return false
	}

	return strings.EqualFold(out, "Running")
}

func windowsAutoUpdatesEnabled() bool {
	out, err := runPowerShell(`
		Get-ItemProperty -Path "HKLM:\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU" ` +
		`-Name NoAutoUpdate -ErrorAction SilentlyContinue ` +
		`| Select-Object -ExpandProperty NoAutoUpdate
	`)
	if err != nil || out == "" {
		// Key missing → updates enabled
		return true
	}

	return out != "1"
}

func windowsTPMAvailable() bool {
	out, err := runPowerShell(`
		(Get-CimInstance -Namespace root\cimv2\security\microsofttpm ` +
		`-ClassName Win32_Tpm).IsEnabled_InitialValue
	`)
	if err != nil {
		return false
	}

	return strings.EqualFold(out, "True")
}

func windowsBiometricsEnabled() bool {
	out, err := runPowerShell(`
		Get-ItemProperty -Path "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Biometrics" ` +
		`-Name Enabled -ErrorAction SilentlyContinue ` +
		`| Select-Object -ExpandProperty Enabled
	`)
	if err != nil || out == "" {
		return false
	}

	return out == "1"
}

func runPowerShell(cmd string) (string, error) {
	out, err := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-Command",
		cmd,
	).CombinedOutput()

	return strings.TrimSpace(string(out)), err
}

func computePlatformFingerprint() string {
	parts := []string{
		runtime.GOOS,
		runtime.GOARCH,
		cpuFingerprint(),
		dmiFingerprint(),
	}

	fmt.Println("parts")

	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}

	raw := strings.Join(out, "|")
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func cpuFingerprint() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()

	var parts []string

	if v, _, err := k.GetStringValue("VendorIdentifier"); err == nil {
		parts = append(parts, "vendor="+normalize(v))
	}
	if v, _, err := k.GetStringValue("ProcessorNameString"); err == nil {
		parts = append(parts, "model_name="+normalize(v))
	}
	if v, _, err := k.GetStringValue("Identifier"); err == nil {
		parts = append(parts, "identifier="+normalize(v))
	}

	return strings.Join(parts, "|")
}

func dmiFingerprint() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\SystemInformation`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()

	var parts []string

	read := func(name, key string) {
		if v, _, err := k.GetStringValue(name); err == nil && v != "" {
			parts = append(parts, key+"="+normalize(v))
		}
	}

	read("SystemManufacturer", "sys_vendor")
	read("SystemProductName", "product_name")
	read("SystemSKU", "sku")
	read("BaseBoardManufacturer", "board_vendor")
	read("BaseBoardProduct", "board_name")

	return strings.Join(parts, "|")
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(s), " ")
}

func GetDeviceName() string {
	// TODO: implement laptop/desktop detection
	return "Windows"
}
