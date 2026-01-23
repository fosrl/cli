package apply

import (
	"github.com/fosrl/cli/cmd/apply/blueprint"
	"github.com/spf13/cobra"
)

func ApplyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply commands",
		Long:  "Apply resources to the Pangolin server",
	}

	cmd.AddCommand(blueprint.BlueprintCmd())

	return cmd
}
