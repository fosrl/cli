package svcmgr

import (
	"bytes"
	"io"
	"sync"
)

// crashMarkers are line prefixes the Go runtime prints when a process dies
// from an unrecovered panic or a fatal runtime error (on Windows, "Exception"
// precedes the dump for access violations and other hardware exceptions).
var crashMarkers = [][]byte{
	[]byte("panic: "),
	[]byte("fatal error: "),
	[]byte("Exception 0x"),
}

const maxCrashMarkerLen = len("fatal error: ")

// crashWatcher is the supervised child's stdout/stderr. It passes everything
// through to out unchanged and closes Crashed the first time a line starts
// with one of crashMarkers.
//
// The supervisor needs this because a crashing Go process does not always
// exit: if a second thread faults, or the traceback itself faults, the
// runtime can park forever in its crash handler. The process then keeps its
// sockets open (so the site still looks connected) while forwarding nothing,
// and the supervisor, which only reacts to process exit, never restarts it.
type crashWatcher struct {
	out     io.Writer
	line    []byte // start of the current line, at most maxCrashMarkerLen bytes
	crashed chan struct{}
	once    sync.Once
}

func newCrashWatcher(out io.Writer) *crashWatcher {
	return &crashWatcher{
		out:     out,
		line:    make([]byte, 0, maxCrashMarkerLen),
		crashed: make(chan struct{}),
	}
}

// Crashed is closed once the child has started printing a crash.
func (w *crashWatcher) Crashed() <-chan struct{} {
	return w.crashed
}

// Write never fails or short-writes: an error here would make the child's
// own log writes fail or block, which is worse than losing log output.
func (w *crashWatcher) Write(p []byte) (int, error) {
	if w.out != nil {
		_, _ = w.out.Write(p)
	}

	rest := p
	for len(rest) > 0 {
		chunk := rest
		nl := bytes.IndexByte(rest, '\n')
		if nl >= 0 {
			chunk = rest[:nl]
		}
		if room := maxCrashMarkerLen - len(w.line); room > 0 {
			w.line = append(w.line, chunk[:min(room, len(chunk))]...)
		}
		w.check()
		if nl < 0 {
			break
		}
		w.line = w.line[:0]
		rest = rest[nl+1:]
	}
	return len(p), nil
}

func (w *crashWatcher) check() {
	for _, m := range crashMarkers {
		if bytes.HasPrefix(w.line, m) {
			w.once.Do(func() { close(w.crashed) })
			return
		}
	}
}
