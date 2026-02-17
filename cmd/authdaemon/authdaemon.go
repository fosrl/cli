package authdaemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/fosrl/cli/internal/logger"
	authdaemonpkg "github.com/fosrl/newt/authdaemon"
	"github.com/spf13/cobra"
)

const (
	defaultPort             = 22123
	defaultPrincipalsPath   = "/var/run/auth-daemon/principals"
	defaultCACertPath       = ""
	defaultSSHDConfigPath   = "/etc/ssh/sshd_config"
	defaultReloadSSHCommand = ""
)

var (
	errPresharedKeyRequired = errors.New("pre-shared-key is required")
	errRootRequired         = errors.New("auth-daemon must be run as root (use sudo)")
)

func AuthDaemonCmd() *cobra.Command {
	opts := struct {
		PreSharedKey     string
		Port             int
		PrincipalsFile   string
		CACertPath       string
		SSHDConfigPath   string
		ReloadSSHCommand string
	}{}

	cmd := &cobra.Command{
		Use:   "auth-daemon",
		Short: "Start the auth daemon",
		Long:  "Start the auth daemon for remote SSH authentication",
		PreRunE: func(c *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" {
				return fmt.Errorf("auth-daemon is only supported on Linux, not %s", runtime.GOOS)
			}
			if opts.PreSharedKey == "" {
				return errPresharedKeyRequired
			}
			if os.Geteuid() != 0 {
				return errRootRequired
			}
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			runAuthDaemon(opts)
		},
	}

	cmd.Flags().StringVar(&opts.PreSharedKey, "pre-shared-key", "", "Preshared key required for all API requests (required)")
	cmd.MarkFlagRequired("pre-shared-key")
	cmd.Flags().IntVar(&opts.Port, "port", defaultPort, "TCP listen port for the HTTPS server")
	cmd.Flags().StringVar(&opts.PrincipalsFile, "principals-file", defaultPrincipalsPath, "Path to the principals file (one principal per line); used by SSH or other tools")
	cmd.Flags().StringVar(&opts.CACertPath, "ca-cert-path", defaultCACertPath, "If set, write CA cert here on POST /connection when the file does not exist; PAM/OpenSSH use this")
	cmd.Flags().StringVar(&opts.SSHDConfigPath, "sshd-config-path", defaultSSHDConfigPath, "Path to sshd_config when using CA cert (used with --ca-cert-path)")
	cmd.Flags().StringVar(&opts.ReloadSSHCommand, "reload-ssh", defaultReloadSSHCommand, "Command to reload sshd after config change (e.g. \"systemctl reload sshd\"); empty = no reload")

	cmd.AddCommand(PrincipalsCmd())

	return cmd
}

// PrincipalsCmd returns the "principals" subcommand for use as AuthorizedPrincipalsCommand in sshd_config.
func PrincipalsCmd() *cobra.Command {
	opts := struct {
		PrincipalsFile string
		Username       string
	}{}

	cmd := &cobra.Command{
		Use:   "principals",
		Short: "Output principals for a username (for AuthorizedPrincipalsCommand in sshd_config)",
		Long:  "Read the principals file and print principals that match the given username, one per line. Configure in sshd_config with AuthorizedPrincipalsCommand and %u for the username.",
		PreRunE: func(c *cobra.Command, args []string) error {
			if opts.PrincipalsFile == "" {
				return errors.New("principals-file is required")
			}
			if opts.Username == "" {
				return errors.New("username is required")
			}
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			runPrincipals(opts.PrincipalsFile, opts.Username)
		},
	}

	cmd.Flags().StringVar(&opts.PrincipalsFile, "principals-file", defaultPrincipalsPath, "Path to the principals file written by the auth daemon")
	cmd.Flags().StringVar(&opts.Username, "username", "", "Username to look up (e.g. from sshd %u)")
	cmd.MarkFlagRequired("username")

	return cmd
}

func runPrincipals(principalsPath, username string) {
	list, err := authdaemonpkg.GetPrincipals(principalsPath, username)
	if err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
	if len(list) == 0 {
		fmt.Println("")
		return
	}
	for _, principal := range list {
		fmt.Println(principal)
	}
	return
}

func runAuthDaemon(opts struct {
	PreSharedKey     string
	Port             int
	PrincipalsFile   string
	CACertPath       string
	SSHDConfigPath   string
	ReloadSSHCommand string
}) {
	cfg := authdaemonpkg.Config{
		Port:               opts.Port,
		PresharedKey:       opts.PreSharedKey,
		PrincipalsFilePath: opts.PrincipalsFile,
		CACertPath:         opts.CACertPath,
		SSHDConfigPath:     opts.SSHDConfigPath,
		ReloadSSHCommand:   opts.ReloadSSHCommand,
	}

	srv, err := authdaemonpkg.NewServer(cfg)
	if err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := srv.Run(ctx); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}
