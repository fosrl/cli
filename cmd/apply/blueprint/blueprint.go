package blueprint

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/newt/logger"
	"github.com/spf13/cobra"
)

type BlueprintCmdOpts struct {
	Name     string
	Path     string
	APIKey   string
	Endpoint string
	OrgID    string
}

func BlueprintCmd() *cobra.Command {
	opts := BlueprintCmdOpts{}

	cmd := &cobra.Command{
		Use:   "blueprint",
		Short: "Apply a blueprint",
		Long:  "Apply a YAML blueprint to the Pangolin server",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Integration API: any of the three flags implies all three are required (avoids silent session fallback).
			integration := opts.APIKey != "" || opts.Endpoint != "" || opts.OrgID != ""
			if integration && (opts.APIKey == "" || opts.Endpoint == "" || opts.OrgID == "") {
				return errors.New("integration API mode requires --api-key, --endpoint, and --org together; omit all three to use your logged-in session and selected org")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := applyBlueprintMain(cmd, opts); err != nil {
				return err
			}
			logger.Info("Successfully applied blueprint!")
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "file", "f", "", "Blueprint YAML file")
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Blueprint name (default: filename without extension)")
	cmd.Flags().StringVar(&opts.APIKey, "api-key", "", "Integration API key (id.secret)")
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "Integration API host URL")
	cmd.Flags().StringVar(&opts.OrgID, "org", "", "Organization ID")
	cmd.MarkFlagRequired("file")

	return cmd
}

func applyBlueprintMain(cmd *cobra.Command, opts BlueprintCmdOpts) error {
	name := opts.Name
	if name == "" {
		filename := filepath.Base(opts.Path)
		switch ext := strings.ToLower(filepath.Ext(filename)); ext {
		case ".yaml", ".yml":
			name = strings.TrimSuffix(filename, ext)
		default:
			name = filename
		}
	}
	if len(name) < 1 || len(name) > 255 {
		return errors.New("name must be between 1-255 characters")
	}

	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	blueprintContents, err := os.ReadFile(opts.Path)
	if err != nil {
		return fmt.Errorf("failed to read blueprint file: %w", err)
	}

	blueprintContents = interpolateBlueprint(blueprintContents)

	client := apiClient
	orgID := opts.OrgID

	if opts.APIKey != "" {
		client, err = apiClient.WithIntegrationAPIKey(opts.Endpoint, opts.APIKey)
		if err != nil {
			return fmt.Errorf("failed to initialize api key client: %w", err)
		}
	} else {
		account, errAcc := accountStore.ActiveAccount()
		if errAcc != nil {
			return errAcc
		}
		if account.OrgID == "" {
			return errors.New("no organization selected")
		}
		orgID = account.OrgID
	}

	_, err = client.ApplyBlueprint(orgID, name, string(blueprintContents))
	if err != nil {
		return fmt.Errorf("failed to apply blueprint: %w", err)
	}
	return nil
}

// interpolateBlueprint finds all {{...}} tokens in the raw blueprint bytes and
// replaces recognised schemes with their resolved values. Currently supported:
//
//   - env.<VAR>  – replaced with the value of the named environment variable
//
// Any token that does not match a supported scheme is left as-is so that
// future schemes are preserved rather than silently dropped.
func interpolateBlueprint(data []byte) []byte {
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return re.ReplaceAllFunc(data, func(match []byte) []byte {
		inner := strings.TrimSpace(string(match[2 : len(match)-2]))

		if strings.HasPrefix(inner, "env.") {
			varName := strings.TrimPrefix(inner, "env.")
			return []byte(os.Getenv(varName))
		}

		// unrecognised scheme – leave the token untouched
		return match
	})
}
