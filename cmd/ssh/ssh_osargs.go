package ssh

import (
	"strconv"

	"github.com/fosrl/cli/internal/logger"
)

func sshPassThroughFromOS(osArgs []string, resourceID string) []string {
	if len(osArgs) == 0 || resourceID == "" {
		return nil
	}
	sshIdx := -1
	for i := 0; i < len(osArgs); i++ {
		if osArgs[i] == "ssh" {
			sshIdx = i
		}
	}
	if sshIdx < 0 || sshIdx+1 >= len(osArgs) {
		return nil
	}
	tail := append([]string{}, osArgs[sshIdx+1:]...)
	tail = stripPangolinSSHKnownFlags(tail)
	tail = removeFirstTokenEqual(tail, resourceID)
	return tail
}

// stripPangolinSSHKnownFlags removes pangolin-only tokens so they are not forwarded to ssh(1).
// --port / -p are parsed by Cobra and applied via RunOpts.Port; stripping avoids duplicate port
// flags in the ssh(1) argv built from passthrough.
func stripPangolinSSHKnownFlags(in []string) []string {
	out := make([]string, 0, len(in))
	saidLegacyExec := false
	for i := 0; i < len(in); {
		switch {
		case in[i] == "--exec":
			if !saidLegacyExec {
				saidLegacyExec = true
				logger.Info("Note: --exec is no longer needed; the system OpenSSH client is the default. This flag is ignored and not passed to ssh(1).\n")
			}
			i++
		case in[i] == "--builtin":
			i++
		case in[i] == "--port" && i+1 < len(in):
			i += 2
		case in[i] == "-p" && i+1 < len(in) && isAllDigits(in[i+1]):
			i += 2
		default:
			out = append(out, in[i])
			i++
		}
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

func removeFirstTokenEqual(in []string, id string) []string {
	for i, s := range in {
		if s == id {
			return append(append([]string{}, in[:i]...), in[i+1:]...)
		}
	}
	return append([]string(nil), in...)
}

func mergePassThrough(osArgs []string, resourceID string, cobraTail []string) []string {
	fromOS := sshPassThroughFromOS(osArgs, resourceID)
	if len(fromOS) > 0 {
		return fromOS
	}
	if len(cobraTail) > 0 {
		return append([]string(nil), cobraTail...)
	}
	return nil
}
