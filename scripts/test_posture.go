package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fosrl/cli/internal/fingerprint"
)

func main() {
	fmt.Println("=== Posture Checks ===")
	fmt.Println()

	postures := fingerprint.GatherPostureChecks()

	// Print in a readable format
	fmt.Println("Platform-agnostic checks:")
	fmt.Printf("  Biometrics Enabled:  %v\n", postures.BiometricsEnabled)
	fmt.Printf("  Disk Encrypted:      %v\n", postures.DiskEncrypted)
	fmt.Printf("  Firewall Enabled:    %v\n", postures.FirewallEnabled)
	fmt.Printf("  Auto Updates:        %v\n", postures.AutoUpdatesEnabled)
	fmt.Printf("  TPM Available:       %v\n", postures.TpmAvailable)
	fmt.Println()

	fmt.Println("macOS-specific checks:")
	fmt.Printf("  SIP Enabled:         %v\n", postures.MacOSSIPEnabled)
	fmt.Printf("  Gatekeeper Enabled:  %v\n", postures.MacOSGatekeeperEnabled)
	fmt.Printf("  Firewall Stealth:    %v\n", postures.MacOSFirewallStealthMode)
	fmt.Println()

	// Also print as JSON for easy inspection
	fmt.Println("=== JSON Output ===")
	jsonData, err := json.MarshalIndent(postures, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}
