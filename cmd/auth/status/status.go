package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

func StatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  "Check if you are logged in and view your account information",
		Run: func(cmd *cobra.Command, args []string) {
			if err := statusMain(cmd); err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}

func statusMain(cmd *cobra.Command) error {
	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	account, err := accountStore.ActiveAccount()
	if err != nil {
		logger.Info("Status: %s", err)
		logger.Info("Run 'pangolin login' to authenticate")
		return err
	}

	// Check health before fetching user data
	healthOk, healthErr := apiClient.CheckHealth()
	isServerDown := healthErr != nil || !healthOk

	if isServerDown {
		logger.Warning("The server appears to be down.")
		fmt.Println()
		// Still show account info from stored data
		logger.Info("Status: logged in (using cached account data)")
		logger.Info("@ %s", account.Host)
		fmt.Println()

		// Display account information from stored data
		displayName := utils.AccountDisplayName(account)
		if displayName != "" {
			logger.Info("User: %s", displayName)
		}
		if account.UserID != "" {
			logger.Info("User ID: %s", account.UserID)
		}

		// Display organization information if available
		if account.OrgID != "" {
			logger.Info("Org ID: %s", account.OrgID)
		}

		return nil
	}

	// Health check passed, try to get user from API
	user, err := apiClient.GetUser()
	if err != nil {
		// Unable to get user - show error but still display account info
		logger.Warning("Failed to fetch user data: %v", err)
		fmt.Println()
		logger.Info("Status: logged in (using cached account data)")
		logger.Info("@ %s", account.Host)
		fmt.Println()

		// Display account information from stored data
		displayName := utils.AccountDisplayName(account)
		if displayName != "" {
			logger.Info("User: %s", displayName)
		}
		if account.UserID != "" {
			logger.Info("User ID: %s", account.UserID)
		}

		// Display organization information if available
		if account.OrgID != "" {
			logger.Info("Org ID: %s", account.OrgID)
		}

		return nil
	}

	// Successfully got user - logged in
	logger.Success("Status: logged in")
	// Show hostname if available
	logger.Info("@ %s", account.Host)
	fmt.Println()

	// Display user information
	displayName := utils.UserDisplayName(user)
	if displayName != "" {
		logger.Info("User: %s", displayName)
	}
	if user.UserID != "" {
		logger.Info("User ID: %s", user.UserID)
	}

	// Display organization information
	if account.OrgID != "" {
		logger.Info("Org ID: %s", account.OrgID)
	}

	// Show watermark messages if server info is available
	if account.ServerInfo != nil {
		watermark := getWatermarkMessage(account.ServerInfo)
		if watermark != "" {
			fmt.Println()
			logger.Info("%s", watermark)
		}
	}

	return nil
}

// getWatermarkMessage returns the appropriate watermark message based on server info
func getWatermarkMessage(serverInfo *config.ServerInfo) string {
	if serverInfo == nil {
		return ""
	}

	build := strings.ToLower(serverInfo.Build)
	licenseType := ""
	if serverInfo.EnterpriseLicenseType != nil {
		licenseType = strings.ToLower(*serverInfo.EnterpriseLicenseType)
	}

	// Enterprise + Personal License
	if build == "enterprise" && licenseType == "personal" {
		return "Licensed for personal use only."
	}

	// Enterprise + Unlicensed
	if build == "enterprise" && !serverInfo.EnterpriseLicenseValid {
		return "This server is unlicensed."
	}

	// OSS + No Supporter Key
	if build == "oss" && !serverInfo.SupporterStatusValid {
		return "Community Edition. Consider supporting."
	}

	return ""
}
