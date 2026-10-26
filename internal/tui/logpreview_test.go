package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGetLastLogLines(t *testing.T) {
	tests := []struct {
		name    string
		content string
		n       int
		lastPos int64
		want    []string
	}{
		{name: "empty", n: 5},
		{name: "fewer lines", content: "one\ntwo\n", n: 5, want: []string{"one", "two"}},
		{name: "last lines", content: "one\ntwo\nthree\n", n: 2, want: []string{"two", "three"}},
		{name: "no trailing newline", content: "one\ntwo", n: 1, want: []string{"two"}},
		{name: "CRLF", content: "one\r\ntwo\r\n", n: 5, want: []string{"one", "two"}},
		{name: "blank lines", content: "\none\n\n", n: 5, want: []string{"", "one", ""}},
		{name: "single blank line", content: "\n", n: 5, want: []string{""}},
		{name: "unchanged", content: "one\n", n: 5, lastPos: 4},
		{name: "incremental", content: "one\ntwo\n", n: 5, lastPos: 4, want: []string{"two"}},
		{name: "negative offset", content: "one\n", n: 5, lastPos: -1, want: []string{"one"}},
		{name: "zero lines", content: "one\n", n: 0},
		{name: "negative lines", content: "one\n", n: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "client.log")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			lines, pos := getLastLogLines(path, tt.n, tt.lastPos)
			if !slices.Equal(lines, tt.want) {
				t.Errorf("lines = %q, want %q", lines, tt.want)
			}
			wantPos := int64(len(tt.content))
			if tt.n <= 0 {
				wantPos = tt.lastPos
			}
			if pos != wantPos {
				t.Errorf("position = %d, want %d", pos, wantPos)
			}
		})
	}
}

func TestGetLastLogLinesMissingFile(t *testing.T) {
	lines, pos := getLastLogLines(filepath.Join(t.TempDir(), "missing.log"), 5, 123)
	if len(lines) != 0 || pos != 123 {
		t.Fatalf("got (%q, %d), want no lines and unchanged position", lines, pos)
	}
}

func TestGetLastLogLinesReadWindow(t *testing.T) {
	longLine := strings.Repeat("x", maxLogPreviewBytes)
	alignedLine := strings.Repeat("x", maxLogPreviewBytes-12)
	crlfLine := strings.Repeat("x", maxLogPreviewBytes-9)
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{name: "partial first line", content: longLine + "\none\ntwo\n", want: []string{"one", "two"}},
		{name: "line boundary", content: "old\nfirst\n" + alignedLine + "\nlast\n", want: []string{"first", alignedLine, "last"}},
		{name: "CRLF boundary", content: "old\r\n" + crlfLine + "\r\nlast\r\n", want: []string{crlfLine, "last"}},
		{name: "oversized last line", content: longLine + "x"},
		{name: "oversized terminated line", content: longLine + "x\n"},
		{name: "after oversized line", content: longLine + "x\nlast", want: []string{"last"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "client.log")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			lines, pos := getLastLogLines(path, 5, 0)
			if !slices.Equal(lines, tt.want) {
				t.Errorf("unexpected preview: got %d lines, want %d", len(lines), len(tt.want))
			}
			if pos != int64(len(tt.content)) {
				t.Errorf("position = %d, want %d", pos, len(tt.content))
			}
		})
	}
}

func TestGetLastLogLinesWindowThreshold(t *testing.T) {
	for _, size := range []int{maxLogPreviewBytes - 1, maxLogPreviewBytes, maxLogPreviewBytes + 1} {
		for _, lastPos := range []int64{0, 4} {
			t.Run(fmt.Sprintf("size_%d_offset_%d", size, lastPos), func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "client.log")
				first := strings.Repeat("x", size-len("\nlast\n"))
				content := strings.Repeat("p", int(lastPos)) + first + "\nlast\n"
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				want := []string{first, "last"}
				if size > maxLogPreviewBytes {
					want = []string{"last"}
				}
				lines, pos := getLastLogLines(path, 5, lastPos)
				if !slices.Equal(lines, want) || pos != int64(len(content)) {
					t.Fatalf("unexpected preview or position: %d lines, position %d", len(lines), pos)
				}
			})
		}
	}
}

func TestGetLastLogLinesLargeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.log")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	// A sparse file exercises multi-GiB offsets without writing GiB of test data.
	const offset int64 = 8 << 30
	const tail = "\none\ntwo\nthree\nfour\nfive\nsix\n"
	if _, err := file.WriteAt([]byte(tail), offset); err != nil {
		t.Fatal(err)
	}
	for _, lastPos := range []int64{0, 4} {
		t.Run(fmt.Sprintf("offset_%d", lastPos), func(t *testing.T) {
			lines, pos := getLastLogLines(path, 5, lastPos)
			want := []string{"two", "three", "four", "five", "six"}
			if !slices.Equal(lines, want) || pos != offset+int64(len(tail)) {
				t.Fatalf("got (%q, %d), want (%q, %d)", lines, pos, want, offset+int64(len(tail)))
			}
		})
	}
}

func TestGetLastLogLinesAfterTruncate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.log")
	if err := os.WriteFile(path, []byte("old log line\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, lastPos := getLastLogLines(path, 5, 0)
	for _, step := range []struct {
		content string
		want    []string
	}{
		{content: "new\n", want: []string{"new"}},
		{content: ""},
		{content: "one\n", want: []string{"one"}},
		{content: "one\ntwo\n", want: []string{"two"}},
		{content: "one\ntwo\n"},
	} {
		if err := os.WriteFile(path, []byte(step.content), 0o600); err != nil {
			t.Fatal(err)
		}
		lines, pos := getLastLogLines(path, 5, lastPos)
		if !slices.Equal(lines, step.want) || pos != int64(len(step.content)) {
			t.Fatalf("got (%q, %d), want (%q, %d)", lines, pos, step.want, len(step.content))
		}
		lastPos = pos
	}
}

func BenchmarkGetLastLogLines(b *testing.B) {
	for _, size := range []int{64 << 10, 16 << 20} {
		b.Run(fmt.Sprintf("%dKiB", size>>10), func(b *testing.B) {
			path := filepath.Join(b.TempDir(), "client.log")
			line := "INFO: synthetic log entry for preview benchmark\n"
			content := strings.Repeat(line, size/len(line))
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				b.Fatal(err)
			}
			for _, lastPos := range []int64{0, int64(len(line))} {
				b.Run(fmt.Sprintf("offset_%d", lastPos), func(b *testing.B) {
					b.ReportAllocs()
					for b.Loop() {
						lines, _ := getLastLogLines(path, 5, lastPos)
						if len(lines) != 5 {
							b.Fatalf("got %d lines, want 5", len(lines))
						}
					}
				})
			}
		})
	}
}
