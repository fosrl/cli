package configure

import (
	"fmt"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

// virtualApiKeyPrefix matches src/lib/virtualApiKeyFormat.ts's VIRTUAL_API_KEY_PREFIX.
const virtualApiKeyPrefix = "pangolin-key-"

// AuthMode is how the configured client authenticates to the AI gateway.
type AuthMode string

const (
	AuthModeKeyed   AuthMode = "keyed"
	AuthModeKeyless AuthMode = "keyless"
)

// Auth is the resolved credential (if any) to write into a client's config.
type Auth struct {
	Mode AuthMode
	Key  string
}

// clientWriter writes the local config file(s) for one AI client and returns
// the paths it touched, for the success message.
type clientWriter func(endpoint string, auth Auth) ([]string, error)

// clientReset removes the settings the matching clientWriter adds, leaving
// every other key in those files alone. It returns only the paths it actually
// changed, so a no-op reset reports nothing, and it never creates a file that
// wasn't already there.
type clientReset func() ([]string, error)

type client struct {
	write clientWriter
	reset clientReset
}

var clients = map[string]client{
	"claude":   {write: writeClaudeConfig, reset: resetClaudeConfig},
	"codex":    {write: writeCodexConfig, reset: resetCodexConfig},
	"opencode": {write: writeOpencodeConfig, reset: resetOpencodeConfig},
	"gemini":   {write: writeGeminiConfig, reset: resetGeminiConfig},
}

type ConfigureCmdOpts struct {
	OrgID    string
	Resource string
	Reset    bool
}

func ConfigureCmd() *cobra.Command {
	opts := ConfigureCmdOpts{}

	cmd := &cobra.Command{
		Use:   "configure <client> [key]",
		Short: "Configure a local AI client to use a Pangolin AI gateway resource",
		Long: "Writes local config files (e.g. ~/.claude/settings.json) so an AI client talks to a Pangolin\n" +
			"resource acting as an AI gateway. Supported clients: claude, codex, opencode, gemini.\n\n" +
			"If [key] is omitted, an API key is fetched automatically for public resources (private/site\n" +
			"resources need no key). Pass [key] to configure with a credential obtained elsewhere without\n" +
			"making any API calls for it.\n\n" +
			"Pass --reset to undo this: it deletes exactly the settings this command writes (it can't\n" +
			"restore values they replaced) and leaves the rest of those files untouched.",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return configureMain(cmd, opts, args)
		},
	}

	cmd.Flags().StringVar(&opts.OrgID, "org", "", "Organization ID (defaults to your active org)")
	cmd.Flags().StringVar(&opts.Resource, "resource", "", "Resource niceId or domain to configure against (defaults to auto-pick/prompt)")
	cmd.Flags().BoolVar(&opts.Reset, "reset", false, "Remove the settings this command writes for <client> instead of writing them")

	return cmd
}

func configureMain(cmd *cobra.Command, opts ConfigureCmdOpts, args []string) error {
	clientArg := args[0]

	c, ok := clients[clientArg]
	if !ok {
		if clientArg == "cursor" {
			return fmt.Errorf("cursor must be configured manually; see the AI client config instructions in the dashboard")
		}
		return fmt.Errorf("unsupported client %q; expected one of: claude, codex, opencode, gemini", clientArg)
	}

	if opts.Reset {
		if len(args) == 2 {
			return fmt.Errorf("--reset takes no [key] argument")
		}
		return resetMain(clientArg, c.reset)
	}

	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	orgID, err := utils.ResolveOrgID(accountStore, opts.OrgID)
	if err != nil {
		return err
	}

	var resource *api.LauncherResource
	if opts.Resource != "" {
		resource, err = utils.ResolveResourceByIdentifier(apiClient, orgID, opts.Resource)
	} else {
		resource, err = utils.SelectResourceForm(apiClient, orgID)
	}
	if err != nil {
		return err
	}

	if resource.AccessURL == nil {
		return fmt.Errorf(
			"resource %q has no stable access URL and can't be used for AI client configuration "+
				"(wildcard domain or unsupported mode)",
			resource.NiceID,
		)
	}
	endpoint := *resource.AccessURL

	auth, err := resolveAuth(apiClient, orgID, resource, args)
	if err != nil {
		return err
	}

	paths, err := c.write(endpoint, auth)
	if err != nil {
		return err
	}

	logger.Success("Configured %s for resource %s (%s)", clientArg, resource.Name, resource.NiceID)
	for _, path := range paths {
		logger.Info("Wrote %s", path)
	}

	return nil
}

// resetMain removes one client's Pangolin settings. It needs no org, resource
// or credential, so it makes no API calls and works while signed out.
func resetMain(clientArg string, reset clientReset) error {
	paths, err := reset()
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		logger.Info("No Pangolin configuration found for %s; nothing to remove", clientArg)
		return nil
	}

	logger.Success("Removed Pangolin configuration for %s", clientArg)
	for _, path := range paths {
		logger.Info("Updated %s", path)
	}

	return nil
}

// resolveAuth determines the credential (if any) to configure the client with:
// a literal key passed on the command line wins outright and skips all API
// calls; otherwise public resources get a fetched virtual API key, and private
// ("site") resources are keyless.
func resolveAuth(apiClient *api.Client, orgID string, resource *api.LauncherResource, args []string) (Auth, error) {
	if len(args) == 2 {
		return Auth{Mode: AuthModeKeyed, Key: args[1]}, nil
	}

	if resource.ResourceType != "public" {
		return Auth{Mode: AuthModeKeyless}, nil
	}

	full, err := apiClient.GetResourceByNiceID(orgID, resource.NiceID)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to look up resource details: %w", err)
	}

	keys, err := apiClient.ListMyVirtualApiKeys(orgID, full.ResourceGUID)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to list your API keys for this resource: %w", err)
	}

	secretData, err := apiClient.GetMyVirtualApiKey(orgID, keys.UserKey.VirtualApiKeyID)
	if err != nil {
		return Auth{}, fmt.Errorf("failed to fetch API key secret: %w", err)
	}

	credential := virtualApiKeyPrefix + secretData.VirtualApiKey.VirtualApiKeyID + "." + secretData.VirtualApiKey.Secret
	return Auth{Mode: AuthModeKeyed, Key: credential}, nil
}
