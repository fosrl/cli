//go:build !windows

package companioncmd

import "github.com/spf13/cobra"

func CompanionCmd() *cobra.Command {
	return nil
}
