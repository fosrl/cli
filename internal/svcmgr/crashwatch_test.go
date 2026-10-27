package svcmgr

import (
	"bytes"
	"testing"
)

func crashedNow(w *crashWatcher) bool {
	select {
	case <-w.Crashed():
		return true
	default:
		return false
	}
}

func TestCrashWatcherPassesOutputThrough(t *testing.T) {
	var buf bytes.Buffer
	w := newCrashWatcher(&buf)
	in := "INFO: 2026/10/24 07:13:46 UDP Forwarder: started\nfatal error: unexpected signal\n"
	n, err := w.Write([]byte(in))
	if err != nil || n != len(in) {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(in))
	}
	if buf.String() != in {
		t.Fatalf("output = %q, want %q", buf.String(), in)
	}
}

func TestCrashWatcherDetectsMarkers(t *testing.T) {
	for _, line := range []string{
		"panic: runtime error: invalid memory address or nil pointer dereference\n",
		"fatal error: unexpected signal during runtime execution\n",
		"Exception 0xc0000005 0x0 0xe8 0x7ff75a258cc5\n",
	} {
		w := newCrashWatcher(nil)
		w.Write([]byte("INFO: normal line\n"))
		if crashedNow(w) {
			t.Fatalf("flagged a crash before %q", line)
		}
		w.Write([]byte(line))
		if !crashedNow(w) {
			t.Errorf("did not detect %q", line)
		}
	}
}

func TestCrashWatcherMarkerSplitAcrossWrites(t *testing.T) {
	w := newCrashWatcher(nil)
	for _, part := range []string{"INFO: x\nfat", "al er", "ror: fault\n"} {
		w.Write([]byte(part))
	}
	if !crashedNow(w) {
		t.Fatal("did not detect a marker split across writes")
	}
}

func TestCrashWatcherIgnoresMarkerMidLine(t *testing.T) {
	w := newCrashWatcher(nil)
	w.Write([]byte("INFO: handler recovered from panic: boom\nERROR: fatal error: nope\n"))
	w.Write([]byte("INFO: a very long line that keeps going past the marker length "))
	w.Write([]byte("panic: still the same line\n"))
	if crashedNow(w) {
		t.Fatal("flagged a marker that was not at the start of a line")
	}
}
