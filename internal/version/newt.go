package version

import "runtime/debug"

// newtModulePath is the module path of the embedded newt package whose
// version is reported to the server, since the CLI runs newt's own runtime
// code directly rather than shelling out to a standalone newt binary.
const newtModulePath = "github.com/fosrl/newt"

// NewtVersionOverride is the version of the github.com/fosrl/newt module
// this binary was built against (with any "v" prefix stripped), as read
// from go.mod by the build process.
//
// It's normally set via -ldflags, the same way Version is (see Makefile /
// Dockerfile), so the reported version is deterministic even if the binary
// is stripped or size-optimized in a way that drops Go build info.
var NewtVersionOverride string

// NewtVersion returns the version of the github.com/fosrl/newt module this
// binary was built against. It prefers NewtVersionOverride (set at build
// time via -ldflags); if that wasn't set - e.g. a plain `go build` or
// `go run` invocation that bypasses the Makefile - it falls back to reading
// the module version from the binary's embedded build info, and finally to
// the CLI's own Version if neither is available.
func NewtVersion() string {
	if NewtVersionOverride != "" {
		return NewtVersionOverride
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range bi.Deps {
			if dep.Path == newtModulePath {
				return normalizeVersion(dep.Version)
			}
		}
	}

	return Version
}
