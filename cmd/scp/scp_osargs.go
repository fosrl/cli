package scp

import (
	"os"
	"strings"
)

// rawSCPArgs returns the arguments passed to the scp subcommand directly from
// os.Args, bypassing Cobra/pflag flag parsing. This is necessary because pflag
// does not know whether unknown short flags (e.g. -r, -p, -v) take an argument,
// so it may incorrectly consume the following operand as the flag's value.
func rawSCPArgs() []string {
	for i, a := range os.Args {
		if a == "scp" {
			return os.Args[i+1:]
		}
	}
	return nil
}

// countSCPOperands returns the number of non-flag positional operands in args.
func countSCPOperands(args []string) int {
	count := 0
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			count += len(args[i+1:])
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			i += 1 + scpFlagExtras(a, args, i)
			continue
		}
		count++
		i++
	}
	return count
}

// scpFlagExtras returns how many additional args the given scp short flag consumes.
func scpFlagExtras(a string, args []string, i int) int {
	if len(a) == 2 {
		switch a[1] {
		// scp flags that take a value
		case 'F', 'i', 'J', 'l', 'o', 'P', 'S':
			if i+1 < len(args) {
				return 1
			}
		}
	}
	return 0
}

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
