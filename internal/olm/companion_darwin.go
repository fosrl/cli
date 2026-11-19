//go:build darwin

package olm

// CompanionAgentName is the OLM agent identifier used by Pangolin for macOS.
// Must match the agent the macOS app's packet tunnel passes to OLM.
const CompanionAgentName = "Pangolin macOS"
