package scp

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

func buildExecSCPArgs(scpPath string, opts RunOpts, keyPath, certPath string) []string {
	args := []string{scpPath}
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	if certPath != "" {
		args = append(args, "-o", "CertificateFile="+certPath)
	}
	args = append(args,
		"-o", "PubkeyAuthentication=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "IdentitiesOnly=yes",
		"-o", "PasswordAuthentication=no",
		"-o", "KbdInteractiveAuthentication=no",
	)
	args = append(args, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR")
	if opts.Port > 0 {
		args = append(args, "-P", strconv.Itoa(opts.Port))
	}
	args = append(args, opts.Passthrough.Options...)
	args = append(args, rewriteSCPOperands(opts)...)
	return args
}

func rewriteSCPOperands(opts RunOpts) []string {
	if len(opts.Passthrough.RemoteCommand) == 0 {
		return nil
	}
	rewritten := make([]string, 0, len(opts.Passthrough.RemoteCommand))
	for _, operand := range opts.Passthrough.RemoteCommand {
		rewritten = append(rewritten, rewriteSCPOperand(operand, opts.ResourceID, opts.User, opts.Hostname))
	}
	return rewritten
}

func rewriteSCPOperand(operand, resourceID, user, hostname string) string {
	hostSpec, pathPart, ok := splitSCPOperand(operand)
	if !ok {
		return operand
	}
	if !matchesTargetHost(hostSpec, resourceID) {
		return operand
	}
	return fmt.Sprintf("%s:%s", hostWithUser(hostname, user), pathPart)
}

func splitSCPOperand(s string) (hostSpec string, pathPart string, ok bool) {
	if s == "" {
		return "", "", false
	}
	if runtime.GOOS == "windows" {
		if len(s) >= 2 && s[1] == ':' {
			if len(s) == 2 {
				return "", "", false
			}
			next := s[2]
			if next == '\\' || next == '/' {
				return "", "", false
			}
		}
	}
	idx := strings.IndexByte(s, ':')
	if idx <= 0 || idx == len(s)-1 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

func matchesTargetHost(hostSpec, resourceID string) bool {
	hostOnly := hostSpec
	if u, h, hasAt := strings.Cut(hostSpec, "@"); hasAt {
		if u == "" || h == "" {
			return false
		}
		hostOnly = h
	}
	return hostOnly == resourceID
}

func hostWithUser(hostname, user string) string {
	if user == "" {
		return hostname
	}
	return user + "@" + hostname
}
