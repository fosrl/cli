package ssh

import "strings"

// FilterForNativeMode strips SSH options that the native SSH server does not
// support and returns the safe subset along with the list of rejected tokens.
//
// The native server (newt/nativessh) handles interactive PTY sessions and
// exec requests (remote commands, scp, rsync). It does not support:
//   - Port forwarding   (-L, -R, -D)
//   - Tunnel mode       (-W)
//   - Agent forwarding  (-A, -a)
//   - X11 forwarding    (-X, -Y)
//   - ControlMaster     (-M, -S, -O)
//   - Gateway ports     (-g)
//   - Background mode   (-f)
//   - No-shell mode     (-N)
//   - Jump host         (-J)
//   - Subsystem mode    (-s)
//   - Arbitrary -o opts (may enable unsupported features or conflict with
//     the auth flags already injected by buildExecSSHArgs)
//
// Allowed: client-side presentation flags that do not require server-side
// handling: -v/-vv/-vvv (verbosity), -t (force PTY), -T (no PTY), -q (quiet),
// -C (compression), -e <char> (escape character). Remote commands are passed
// through unchanged because the native server supports exec requests.
func FilterForNativeMode(pt SSHPassthrough) (SSHPassthrough, []string) {
	var allowed []string
	var stripped []string

	opts := pt.Options
	i := 0
	for i < len(opts) {
		tok := opts[i]
		if tok == "--" {
			// Anything after "--" is a remote command delimiter; skip it and
			// treat remaining tokens as part of the remote command block.
			i++
			break
		}

		ok, consumesNext := nativeAllowedOption(tok)
		if ok {
			allowed = append(allowed, tok)
			i++
			if consumesNext && i < len(opts) {
				allowed = append(allowed, opts[i])
				i++
			}
		} else {
			stripped = append(stripped, tok)
			// Consume the value token that belongs to this flag, if any.
			extras := openSSHOptionExtras(tok, opts, i)
			for j := 0; j < extras && i+1 < len(opts); j++ {
				i++
				stripped = append(stripped, opts[i])
			}
			i++
		}
	}

	// Remote commands pass through unchanged — the native server supports exec.
	var out SSHPassthrough
	if len(allowed) > 0 {
		out.Options = allowed
	}
	out.RemoteCommand = pt.RemoteCommand
	return out, stripped
}

// FilterForNativeSCPMode strips scp(1) options that are unsafe or meaningless
// against the native SSH server and returns the safe subset and rejected tokens.
//
// Allowed scp flags: -r (recursive), -p (preserve times), -q (quiet),
// -v/-vv/… (verbosity), -C (compression), -B (batch mode), -3 (via local),
// -l <limit> (bandwidth), -c <cipher>.
// Blocked: -o (arbitrary SSH options), -J (jump host), and anything unknown.
func FilterForNativeSCPMode(pt SSHPassthrough) (SSHPassthrough, []string) {
	var allowed []string
	var stripped []string

	opts := pt.Options
	i := 0
	for i < len(opts) {
		tok := opts[i]
		if tok == "--" {
			i++
			break
		}

		ok, consumesNext := scpNativeAllowedOption(tok)
		if ok {
			allowed = append(allowed, tok)
			i++
			if consumesNext && i < len(opts) {
				allowed = append(allowed, opts[i])
				i++
			}
		} else {
			stripped = append(stripped, tok)
			extras := openSSHOptionExtras(tok, opts, i)
			for j := 0; j < extras && i+1 < len(opts); j++ {
				i++
				stripped = append(stripped, opts[i])
			}
			i++
		}
	}

	var out SSHPassthrough
	if len(allowed) > 0 {
		out.Options = allowed
	}
	// SCP operands (source/dest) are in RemoteCommand — always pass through.
	out.RemoteCommand = pt.RemoteCommand
	return out, stripped
}

// scpNativeAllowedOption reports whether a scp(1) flag is safe for the native server.
func scpNativeAllowedOption(tok string) (allowed bool, consumesNext bool) {
	if tok == "" || tok == "--" || !strings.HasPrefix(tok, "-") || tok == "-" {
		return false, false
	}

	// -v, -vv, -vvv, … — verbosity is client-side only.
	if len(tok) >= 2 {
		allV := true
		for _, c := range tok[1:] {
			if c != 'v' {
				allV = false
				break
			}
		}
		if allV {
			return true, false
		}
	}

	switch tok {
	case "-r", "-R": // recursive
		return true, false
	case "-p": // preserve modification times and modes
		return true, false
	case "-q": // quiet
		return true, false
	case "-C": // compression
		return true, false
	case "-B": // batch mode (no password prompts)
		return true, false
	case "-3": // copy via local host
		return true, false
	case "-l": // bandwidth limit — consumes next token
		return true, true
	case "-c": // cipher specification — consumes next token
		return true, true
	}

	return false, false
}

// nativeAllowedOption reports whether an ssh(1) flag is safe for the native server.
func nativeAllowedOption(tok string) (allowed bool, consumesNext bool) {
	if tok == "" || tok == "--" || !strings.HasPrefix(tok, "-") || tok == "-" {
		return false, false
	}

	// -v, -vv, -vvv, … — verbosity is client-side only.
	if len(tok) >= 2 {
		allV := true
		for _, c := range tok[1:] {
			if c != 'v' {
				allV = false
				break
			}
		}
		if allV {
			return true, false
		}
	}

	switch tok {
	case "-t": // force PTY allocation
		return true, false
	case "-T": // disable PTY allocation
		return true, false
	case "-q": // quiet mode
		return true, false
	case "-C": // compression (negotiated at transport layer)
		return true, false
	case "-e": // escape character — value is consumed
		return true, true
	}

	return false, false
}

// NativeStrippedWarning builds a concise warning string from the list of
// rejected tokens, deduplicating flag names (but not their values).
func NativeStrippedWarning(stripped []string) string {
	seen := make(map[string]struct{}, len(stripped))
	unique := make([]string, 0, len(stripped))
	for _, s := range stripped {
		if _, dup := seen[s]; !dup {
			seen[s] = struct{}{}
			unique = append(unique, s)
		}
	}
	return strings.Join(unique, " ")
}
