package fingerprint

import "encoding/json"

type Fingerprint struct {
	Username            string `json:"username"`
	Hostname            string `json:"hostname"`
	Platform            string `json:"platform"`
	OSVersion           string `json:"osVersion"`
	KernelVersion       string `json:"kernelVersion"`
	Architecture        string `json:"arch"`
	DeviceModel         string `json:"deviceModel"`
	SerialNumber        string `json:"serialNumber"`
	PlatformFingerprint string `json:"platformFingerprint"`
}

func (p *Fingerprint) ToMap() map[string]any {
	b, err := json.Marshal(p)
	if err != nil {
		return map[string]any{}
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}

	return m
}

type PostureChecks struct {
	// Platform-agnostic checks

	BiometricsEnabled  bool `json:"biometricsEnabled"`
	DiskEncrypted      bool `json:"diskEncrypted"`
	FirewallEnabled    bool `json:"firewallEnabled"`
	AutoUpdatesEnabled bool `json:"autoUpdatesEnabled"`
	TpmAvailable       bool `json:"tpmAvailable"`

	// Windows-specific posture check information

	WindowsDefenderEnabled bool `json:"windowsDefenderEnabled"`

	// macOS-specific posture check information

	MacOSSIPEnabled          bool `json:"macosSipEnabled"`
	MacOSGatekeeperEnabled   bool `json:"macosGatekeeperEnabled"`
	MacOSFirewallStealthMode bool `json:"macosFirewallStealthMode"`

	// Linux-specific posture check information

	LinuxAppArmorEnabled bool `json:"linuxAppArmorEnabled"`
	LinuxSELinuxEnabled  bool `json:"linuxSELinuxEnabled"`
}

func (p *PostureChecks) ToMap() map[string]any {
	b, err := json.Marshal(p)
	if err != nil {
		return map[string]any{}
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}

	return m
}
