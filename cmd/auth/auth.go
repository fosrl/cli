package auth

import (
	"github.com/fosrl/cli/cmd/auth/login"
	"github.com/fosrl/cli/cmd/auth/logout"
	"github.com/fosrl/cli/cmd/auth/status"
	"github.com/spf13/cobra"
)

var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  "Manage authentication and sessions",
}

func init() {
	AuthCmd.AddCommand(login.LoginCmd)
	AuthCmd.AddCommand(logout.LogoutCmd)
	AuthCmd.AddCommand(status.StatusCmd)
}
