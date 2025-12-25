package completion

import (
	"os"

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
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			}
		},
	}

	return cmd
}
