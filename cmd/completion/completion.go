package completion

import (
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

func CompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for the specified shell.

The completion script can be sourced to enable command-line completion for pangolin.

Bash:
  $ source <(pangolin completion bash)

  To load completions for each session, execute once:
  Linux:
    $ pangolin completion bash > /etc/bash_completion.d/pangolin
  macOS:
    $ pangolin completion bash > /usr/local/etc/bash_completion.d/pangolin

Zsh:
  If shell completion is not already enabled in your environment, you will need
  to enable it. You can execute the following once:
    $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  To load completions for each session, execute once:
    $ pangolin completion zsh > "${fpath[1]}/_pangolin"

  You will need to start a new shell for this setup to take effect.

Fish:
  $ pangolin completion fish | source

  To load completions for each session, execute once:
    $ pangolin completion fish > ~/.config/fish/completions/pangolin.fish
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			if err := completionMain(cmd, args); err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}

func completionMain(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
			logger.Error("Failed to generate bash completion: %v", err)
			return err
		}
	case "zsh":
		if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
			logger.Error("Failed to generate zsh completion: %v", err)
			return err
		}
	case "fish":
		if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
			logger.Error("Failed to generate fish completion: %v", err)
			return err
		}
	}
	return nil
}
