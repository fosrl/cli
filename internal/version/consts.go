package version

// Version is the current version of the Pangolin CLI.
// This value can be overridden at build time using ldflags:
//
//	go build -ldflags "-X github.com/fosrl/cli/internal/version.Version=<version>"
var Version = "0.15.0"
