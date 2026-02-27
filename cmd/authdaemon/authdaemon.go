//go:build linux

package authdaemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fosrl/cli/internal/logger"
	authdaemonpkg "github.com/fosrl/newt/authdaemon"
	"github.com/spf13/cobra"
)

const (
	defaultPort           = 22123
	defaultPrincipalsPath = "/var/run/auth-daemon/principals"
	defaultCACertPath     = "/etc/ssh/ca.pem"
)

var (
	errPresharedKeyRequired = errors.New("pre-shared-key is required")
	errRootRequired         = errors.New("auth-daemon must be run as root (use sudo)")
)

func AuthDaemonCmd() *cobra.Command {
	opts := struct {
		PreSharedKey           string
		Port                   int
		PrincipalsFile         string
		CACertPath             string
		GenerateRandomPassword bool
	}{}

	cmd := &cobra.Command{
		Use:   "auth-daemon",
		Short: "Start the auth daemon",
		Long:  "Start the auth daemon for remote SSH authentication",
		PreRunE: func(c *cobra.Command, args []string) error {
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

	cmd.Flags().StringVar(&opts.PreSharedKey, "pre-shared-key", "", "Preshared key required for all requests to the auth daemon (required)")
	cmd.MarkFlagRequired("pre-shared-key")
	cmd.Flags().IntVar(&opts.Port, "port", defaultPort, "TCP listen port for the HTTPS server")
	cmd.Flags().StringVar(&opts.PrincipalsFile, "principals-file", defaultPrincipalsPath, "Path to the principals file")
	cmd.Flags().StringVar(&opts.CACertPath, "ca-cert-path", defaultCACertPath, "Path to the CA certificate file")
	cmd.Flags().BoolVar(&opts.GenerateRandomPassword, "generate-random-password", false, "Generate a random password for authenticated users")

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
			if opts.Username == "" {
				return errors.New("username is required")
			}
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			path := opts.PrincipalsFile
			if path == "" {
				path = defaultPrincipalsPath
			}
			runPrincipals(path, opts.Username)
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
}

func runAuthDaemon(opts struct {
	PreSharedKey           string
	Port                   int
	PrincipalsFile         string
	CACertPath             string
	GenerateRandomPassword bool
}) {
	cfg := authdaemonpkg.Config{
		Port:                   opts.Port,
		PresharedKey:           opts.PreSharedKey,
		PrincipalsFilePath:     opts.PrincipalsFile,
		CACertPath:             opts.CACertPath,
		Force:                  true,
		GenerateRandomPassword: opts.GenerateRandomPassword,
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
