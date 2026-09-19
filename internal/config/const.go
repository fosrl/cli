package config

// ManagedBy identifies the package manager managing this binary (e.g., "nix", "brew", "apt").
// Defaults to "none", indicating the CLI manages its own updates internally.
//
// This value can be overridden at build time using ldflags:
//
//	go build -ldflags "-X github.com/fosrl/cli/internal/config.ManagedBy=<manager>"
var ManagedBy = "none"

func IsExternallyManaged() bool {
	return ManagedBy != "none"
}
