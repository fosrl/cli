package client

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

type ClientLogsCmdOpts struct {
	Follow bool
	Lines  int
}

func ClientLogsCmd() *cobra.Command {
	opts := ClientLogsCmdOpts{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "View client logs",
		Long:  "View client logs. Use -f to follow log output.",
		Run: func(cmd *cobra.Command, args []string) {
			clientLogsMain(cmd, &opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Follow, "follow", "f", false, "Follow log output (like tail -f)")
	cmd.Flags().IntVarP(&opts.Lines, "lines", "n", 0, "Number of lines to show (0 = all lines, only used with -f to show lines before following)")

	return cmd
}

func clientLogsMain(cmd *cobra.Command, opts *ClientLogsCmdOpts) {
	cfg := config.ConfigFromContext(cmd.Context())

	if opts.Follow {
		// Follow the log file
		if err := watchLogFile(cfg.LogFile, opts.Lines); err != nil {
			logger.Error("Error: %v", err)
			os.Exit(1)
		}

		return
	}

	// Just print the current log file contents
	if opts.Lines > 0 {
		// Show last N lines
		if err := printLastLines(cfg.LogFile, opts.Lines); err != nil {
			logger.Error("Error: %v", err)
			os.Exit(1)
		}
	} else {
		// Show all lines
		if err := printLogFile(cfg.LogFile); err != nil {
			logger.Error("Error: %v", err)
			os.Exit(1)
		}
	}
}

// printLogFile prints the contents of the log file
func printLogFile(logPath string) error {
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log file does not exist: %s", logPath)
		}
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	// Read and print entire file
	_, err = io.Copy(os.Stdout, file)
	if err != nil {
		return fmt.Errorf("failed to read log file: %v", err)
	}

	return nil
}

// printLastLines prints the last N lines of a file
func printLastLines(logPath string, n int) error {
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("log file does not exist: %s", logPath)
		}
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	lines, err := getLastLines(file, n)
	if err != nil {
		return fmt.Errorf("failed to read last lines: %v", err)
	}

	// Print the lines
	for _, line := range lines {
		fmt.Print(line)
	}

	return nil
}

// getLastLines reads the last N lines from a file
func getLastLines(file *os.File, n int) ([]string, error) {
	// If n is 0 or negative, read all lines
	if n <= 0 {
		var lines []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text()+"\n")
		}
		return lines, scanner.Err()
	}

	// Read all lines and return the last N
	scanner := bufio.NewScanner(file)
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text()+"\n")
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last N lines
	if len(allLines) <= n {
		return allLines, nil
	}
	return allLines[len(allLines)-n:], nil
}

// watchLogFile watches and follows the log file (similar to tail -f)
func watchLogFile(logPath string, numLines int) error {
	// Wait for the log file to be created if it doesn't exist
	var file *os.File
	var err error
	for i := 0; i < 30; i++ { // Wait up to 15 seconds
		file, err = os.Open(logPath)
		if err == nil {
			break
		}
		if i == 0 {
			fmt.Printf("Waiting for log file to be created...\n")
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err != nil {
		return fmt.Errorf("failed to open log file after waiting: %v", err)
	}
	defer file.Close()

	// Show last N lines before following (if numLines > 0, otherwise show all)
	if numLines > 0 {
		lines, err := getLastLines(file, numLines)
		if err != nil {
			return fmt.Errorf("failed to read last lines: %v", err)
		}
		// Print the lines
		for _, line := range lines {
			fmt.Print(line)
		}
	} else {
		// Show all previous lines
		file.Seek(0, io.SeekStart)
		_, err := io.Copy(os.Stdout, file)
		if err != nil {
			return fmt.Errorf("failed to read log file: %v", err)
		}
	}

	// Seek to the end of the file to only show new logs
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %v", err)
	}

	// Set up signal handling for graceful exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Create a ticker to check for new content
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	buffer := make([]byte, 4096)

	for {
		select {
		case <-sigCh:
			fmt.Printf("\n\nStopping log watch...\n")
			return nil

		case <-ticker.C:
			// Read new content
			n, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				// Try to reopen the file in case it was recreated
				file.Close()
				file, err = os.Open(logPath)
				if err != nil {
					return fmt.Errorf("error reopening log file: %v", err)
				}
				// Seek to end again
				file.Seek(0, io.SeekEnd)
				continue
			}

			if n > 0 {
				// Print the new content
				fmt.Print(string(buffer[:n]))
			}
		}
	}
}
