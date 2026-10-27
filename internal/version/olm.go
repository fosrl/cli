package version

// OlmVersionOverride is the version of the github.com/fosrl/olm module this
// binary was built against (with any "v" prefix stripped), as read from
// go.mod by the build process.
//
// It's set via -ldflags, the same way NewtVersionOverride is (see Makefile /
// Dockerfile), and passed to olm as OlmConfig.OlmVersion so the server sees
// the olm version separately from the CLI's own Version. If it wasn't set -
// e.g. a plain `go build` - olm falls back to reading its module version
// from the binary's embedded build info.
var OlmVersionOverride string
