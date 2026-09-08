package svcmgr

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// TailFile streams a growing log file to stdout, similar to `tail -f`,
// optionally showing the last `lines` prior lines first (0 shows the whole
// existing file), until interrupted. Used by the macOS and Windows service
// backends, whose logs are plain files rather than a queryable journal
// (unlike Linux, which uses `journalctl -f` directly).
func TailFile(path string, lines int) error {
	var file *os.File
	var err error
	for i := 0; i < 30; i++ { // wait up to 15 seconds for the file to appear
		file, err = os.Open(path)
		if err == nil {
			break
		}
		if i == 0 {
			fmt.Println("Waiting for log file to be created...")
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed to open log file after waiting: %w", err)
	}
	defer file.Close()

	if lines > 0 {
		tailLines, err := lastLines(file, lines)
		if err != nil {
			return fmt.Errorf("failed to read last lines: %w", err)
		}
		for _, l := range tailLines {
			fmt.Print(l)
		}
	} else if _, err := io.Copy(os.Stdout, file); err != nil {
		return fmt.Errorf("failed to read log file: %w", err)
	}

	if _, err := file.Seek(0, io.SeekEnd); err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	buf := make([]byte, 4096)
	for {
		select {
		case <-sigCh:
			return nil
		case <-ticker.C:
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return fmt.Errorf("failed to read log file: %w", err)
			}
			if n > 0 {
				os.Stdout.Write(buf[:n])
			}
		}
	}
}

// lastLines returns the last n lines already in f.
func lastLines(f *os.File, n int) ([]string, error) {
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var all []string
	for scanner.Scan() {
		all = append(all, scanner.Text()+"\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}
