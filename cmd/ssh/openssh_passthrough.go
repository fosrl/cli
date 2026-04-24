package ssh

import "strings"

type SSHPassthrough struct {
	Options       []string
	RemoteCommand []string
}

// ParseOpenSSHPassThrough walks pass-through args (e.g. args[1:] from "pangolin ssh <res> ...").
func ParseOpenSSHPassThrough(args []string) SSHPassthrough {
	if len(args) == 0 {
		return SSHPassthrough{}
	}
	var opts []string
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "--" {
			opts = append(opts, a)
			i++
			return SSHPassthrough{Options: opts, RemoteCommand: cloneStringSliceOrNil(args[i:])}
		}
		if !strings.HasPrefix(a, "-") {
			break
		}
		ex := openSSHOptionExtras(a, args, i)
		end := i + 1 + ex
		if end > len(args) {
			end = len(args)
		}
		opts = append(opts, args[i:end]...)
		i = end
	}
	return SSHPassthrough{Options: opts, RemoteCommand: cloneStringSliceOrNil(args[i:])}
}

func cloneStringSliceOrNil(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	return append([]string{}, s...)
}

// openSSHOptionExtras returns how many args after the current token should be part of the same
// option (0 = only the current token, e.g. -N; 1 = one following value, e.g. -F path).
func openSSHOptionExtras(a string, args []string, i int) int {
	if a == "" || a == "--" {
		return 0
	}
	// long options
	if len(a) > 1 && a[0] == '-' && a[1] == '-' {
		if strings.Contains(a, "=") {
			return 0
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && longOpenSSHWithArg(a) {
			return 1
		}
		return 0
	}
	if !strings.HasPrefix(a, "-") || a == "-" {
		return 0
	}
	// exactly two runes: e.g. -N, -L, -1, -2
	if len(a) == 2 {
		switch a[1] {
		case '1', '2', '3', '4', '5', '6', '7', '8', '9',
			'N', 'G', 'T', 'C', 'f', 'g', 'n', 'q', 's', 't', 'v', 'x', 'X', 'Y', 'A', 'a', 'M', 'Q', 'V', 'y':
			return 0
		case 'B', 'b', 'c', 'e', 'E', 'F', 'h', 'I', 'J', 'K', 'L', 'm', 'O', 'o', 'P', 'R', 'S', 'W', 'D', 'i', 'l', 'p', 'U':
			return 1
		}
		return 0
	}
	// combined short token
	if a[0] == '-' {
		c := a[1]
		switch c {
		case 'D':
			// -D, -D1080, -D[bind]:port
			if len(a) == 2 {
				return 1
			}
			return 0
		case 'L', 'R':
			// -L, -L8080:host:port
			if len(a) == 2 {
				return 1
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// e.g. -L with space then spec; if no ':' in a[2:] treat next as spec
				if !strings.Contains(a[2:], ":") {
					return 1
				}
			}
			return 0
		case 'O':
			// -O with command (single token) or -O and next
			if len(a) == 2 && i+1 < len(args) {
				return 1
			}
			return 0
		}
	}
	return 0
}

func longOpenSSHWithArg(s string) bool {
	known := map[string]struct{}{
		"--bind-address":  {},
		"--ciphers":       {},
		"--kex":           {},
		"--kexalgorithms": {},
		"--log-level":     {},
		"--macs":          {},
		"--keygen":        {},
		"--user":          {},
	}
	_, ok := known[strings.ToLower(s)]
	return ok
}
