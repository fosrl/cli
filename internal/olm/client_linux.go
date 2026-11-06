//go:build linux

package olm

// CompanionAgentName is the OLM agent identifier used by the Pangolin Linux Client.
// Must match the Agent value in the Linux app's OLM tunnel config.
const CompanionAgentName = "Pangolin Linux"

// desktopAppSocketPath is the socket the Pangolin Linux client's OLM listens
// on. It is distinct from defaultSocketPath, which the CLI's own client uses.
const desktopAppSocketPath = "/run/pangolin-client/olm.sock"
