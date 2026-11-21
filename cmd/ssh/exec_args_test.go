package ssh

import (
	"reflect"
	"testing"
)

func TestBuildExecSSHArgsJITModeUsesAliasWithHostnameOverride(t *testing.T) {
	args := buildExecSSHArgs("/usr/bin/ssh", "alice", "my-server", "10.0.0.5", 0, "/tmp/key", "/tmp/cert", SSHPassthrough{})

	want := []string{
		"/usr/bin/ssh",
		"-l", "alice",
		"-o", "HostName=10.0.0.5",
		"-i", "/tmp/key",
		"-o", "CertificateFile=/tmp/cert",
		"-o", "PubkeyAuthentication=yes",
		"-o", "PreferredAuthentications=publickey,keyboard-interactive,password",
		"-o", "PasswordAuthentication=yes",
		"-o", "KbdInteractiveAuthentication=yes",
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR",
		"my-server",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("argv mismatch:\n got: %v\nwant: %v", args, want)
	}
}

func TestBuildExecSSHArgsManualModeDoesNotForceIdentity(t *testing.T) {
	args := buildExecSSHArgs("/usr/bin/ssh", "alice", "my-server", "10.0.0.5", 0, "", "", SSHPassthrough{})

	for _, a := range args {
		if a == "-i" || a == "IdentitiesOnly=yes" {
			t.Fatalf("manual mode must not force an identity, got %v in %v", a, args)
		}
	}
	if !containsPair(args, "-o", "HostName=10.0.0.5") {
		t.Fatalf("manual mode must still override the hostname, got %v", args)
	}
	if args[len(args)-1] != "my-server" {
		t.Fatalf("manual mode must target the alias so ~/.ssh/config Host blocks match, got %v", args)
	}
	if !containsPair(args, "-o", "PreferredAuthentications=publickey,keyboard-interactive,password") {
		t.Fatalf("manual mode must keep password fallback, got %v", args)
	}
}

func TestBuildExecSSHArgsAliasEqualsHostnameNoOverride(t *testing.T) {
	args := buildExecSSHArgs("/usr/bin/ssh", "", "my-server", "my-server", 0, "/tmp/key", "", SSHPassthrough{})

	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-o" && len(args[i+1]) > 8 && args[i+1][:8] == "HostName" {
			t.Fatalf("no HostName override expected when alias matches hostname, got %v", args)
		}
	}
	if args[len(args)-1] != "my-server" {
		t.Fatalf("expected hostname target, got %v", args)
	}
}

func TestBuildExecSSHArgsEmptyAliasTargetsHostname(t *testing.T) {
	args := buildExecSSHArgs("/usr/bin/ssh", "", "", "10.0.0.5", 2222, "/tmp/key", "", SSHPassthrough{})

	if args[len(args)-1] != "10.0.0.5" {
		t.Fatalf("expected resolved hostname target, got %v", args)
	}
	if !containsPair(args, "-p", "2222") {
		t.Fatalf("expected port flag, got %v", args)
	}
}

func TestBuildExecSSHArgsExplicitIdentityFilePassthrough(t *testing.T) {
	pass := ParseOpenSSHPassThrough([]string{"-i", "/home/user/.ssh/id_ed25519"})
	args := buildExecSSHArgs("/usr/bin/ssh", "alice", "my-server", "10.0.0.5", 0, "/tmp/key", "/tmp/cert", pass)

	// The user's -i must be forwarded and appear before the destination so
	// OpenSSH treats it as an additional explicit identity.
	if !containsPair(args, "-i", "/home/user/.ssh/id_ed25519") {
		t.Fatalf("explicit -i not forwarded: %v", args)
	}
	if !containsPair(args, "-i", "/tmp/key") {
		t.Fatalf("JIT key missing: %v", args)
	}
	if args[len(args)-1] != "my-server" {
		t.Fatalf("expected alias target, got %v", args)
	}
}

func TestBuildExecSSHArgsRemoteCommandAfterTarget(t *testing.T) {
	pass := ParseOpenSSHPassThrough([]string{"-L", "8080:127.0.0.1:80", "-N", "uptime"})
	args := buildExecSSHArgs("/usr/bin/ssh", "", "my-server", "10.0.0.5", 0, "/tmp/key", "", pass)

	wantTail := []string{"-L", "8080:127.0.0.1:80", "-N", "my-server", "uptime"}
	if !reflect.DeepEqual(args[len(args)-len(wantTail):], wantTail) {
		t.Fatalf("expected tail %v, got %v", wantTail, args)
	}
}

func containsPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
