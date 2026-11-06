package version

// Version is the current version of the Pangolin CLI. It is set at build time
// via -ldflags (see Makefile, Dockerfile and flake.nix):
//
//	go build -ldflags "-X github.com/fosrl/cli/internal/version.Version=<version>"
//
// A plain `go build` leaves it as "dev".
var Version = "dev"
