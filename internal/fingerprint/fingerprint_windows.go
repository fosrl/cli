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
	"sync"
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
	var wg sync.WaitGroup

	var biometrics, diskEncrypted, firewall, autoUpdates, tpm, defender bool

	wg.Go(func() {
		biometrics = windowsBiometricsEnabled()
	})

	wg.Go(func() {
		diskEncrypted = windowsDiskEncrypted()
	})

	wg.Go(func() {
		firewall = windowsFirewallEnabled()
	})

	wg.Go(func() {
		autoUpdates = windowsAutoUpdatesEnabled()
	})

	wg.Go(func() {
		tpm = windowsTPMAvailable()
	})

	wg.Go(func() {
		defender = windowsDefenderEnabled()
	})

	wg.Wait()

	return &PostureChecks{
		BiometricsEnabled:      biometrics,
		DiskEncrypted:          diskEncrypted,
		FirewallEnabled:        firewall,
		AutoUpdatesEnabled:     autoUpdates,
		TpmAvailable:           tpm,
		WindowsDefenderEnabled: defender,
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
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SYSTEM\CurrentControlSet\Control\SystemInformation`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return "", ""
	}
	defer k.Close()

	model, _, _ := k.GetStringValue("SystemProductName")
	serial, _, _ := k.GetStringValue("BIOSSerialNumber")

	return model, serial
}

func windowsDiskEncrypted() bool {
	cmd := exec.Command(`C:\Windows\System32\manage-bde.exe`, "-status", "C:")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()

	if err != nil && len(out) == 0 {
		return false
	}

	s := string(out)
	return strings.Contains(s, "Protection On")
}

func windowsFirewallEnabled() bool {
	profiles := []string{"DomainProfile", "StandardProfile", "PublicProfile"}

	for _, profile := range profiles {
		keyPath := fmt.Sprintf(`SYSTEM\CurrentControlSet\Services\SharedAccess\Parameters\FirewallPolicy\%s`, profile)
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		defer k.Close()

		val, _, err := k.GetIntegerValue("EnableFirewall")
		if err != nil {
			continue
		}

		if val != 0 {
			return true
		}
	}

	return false
}

func windowsDefenderEnabled() bool {
	cmd := exec.Command("sc", "query", "WinDefend")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "RUNNING")
}

func windowsAutoUpdatesEnabled() bool {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		// Key missing means updates are enabled
		return true
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue("NoAutoUpdate")
	if err != nil {
		return true // Value missing means updates are enabled
	}

	return val != 1
}

func windowsTPMAvailable() bool {
	cmd := exec.Command("sc", "query", "tpm")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(out), "RUNNING")
}

func windowsBiometricsEnabled() bool {
	key, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Biometrics`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer key.Close()

	val, _, err := key.GetIntegerValue("Enabled")
	if err != nil {
		return false
	}

	return val == 1
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
