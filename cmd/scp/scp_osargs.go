package scp

import "strings"

// parseSCPRemoteHost scans scp operands and returns the username and resource ID
// from the first remote operand (host:path or user@host:path). Local paths are skipped.
func parseSCPRemoteHost(args []string) (username, resourceID string, found bool) {
	for _, arg := range args {
		hostSpec, _, ok := splitSCPOperand(arg)
		if !ok {
			continue
		}
		if u, h, hasAt := strings.Cut(hostSpec, "@"); hasAt {
			if u != "" && h != "" {
				return u, h, true
			}
		} else {
			return "", hostSpec, true
		}
	}
	return "", "", false
}
