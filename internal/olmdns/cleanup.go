package olmdns

import (
	override "github.com/fosrl/olm/dns/override"
)

const DefaultInterfaceName = "pangolin"

// CleanupStaleState removes DNS configuration left from an unclean shutdown
// (for example, killing the process while the tunnel was active).
func CleanupStaleState(interfaceName string) error {
	if interfaceName == "" {
		interfaceName = DefaultInterfaceName
	}
	return override.CleanupStaleState(interfaceName)
}
