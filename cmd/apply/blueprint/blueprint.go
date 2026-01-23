package blueprint

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/newt/logger"
	"github.com/spf13/cobra"
)

type BlueprintCmdOpts struct {
	Name string
	Path string
}

func BlueprintCmd() *cobra.Command {
	opts := BlueprintCmdOpts{}

	cmd := &cobra.Command{
		Use:   "blueprint",
		Short: "Apply a blueprint",
		Long:  "Apply a YAML blueprint to the Pangolin server",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Path == "" {
				return errors.New("--file is required")
			}

			if _, err := os.Stat(opts.Path); err != nil {
				return err
			}

			// Strip file extension and use file basename path as name
			if opts.Name == "" {
				filename := filepath.Base(opts.Path)
				if before, ok := strings.CutSuffix(filename, ".yaml"); ok {
					opts.Name = before
				} else if before, ok := strings.CutSuffix(filename, ".yml"); ok {
					opts.Name = before
				} else {
					opts.Name = filename
				}
			}

			if len(opts.Name) < 1 || len(opts.Name) > 255 {
				return errors.New("name must be between 1-255 characters")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := applyBlueprintMain(cmd, opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&opts.Path, "file", "f", "", "Path to blueprint file (required)")
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Name of blueprint (default: filename, without extension)")
	cmd.MarkFlagRequired("file")

	return cmd
}

func applyBlueprintMain(cmd *cobra.Command, opts BlueprintCmdOpts) error {
	api := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	account, err := accountStore.ActiveAccount()
	if err != nil {
		logger.Error("Error: %v", err)
		return err
	}

	if account.OrgID == "" {
		logger.Error("Error: no organization selected. Run 'pangolin select org' first.")
		return errors.New("no organization selected")
	}

	blueprintContents, err := os.ReadFile(opts.Path)
	if err != nil {
		logger.Error("Error: failed to read blueprint file: %v", err)
		return err
	}

	_, err = api.ApplyBlueprint(account.OrgID, opts.Name, string(blueprintContents))
	if err != nil {
		logger.Error("Error: failed to apply blueprint: %v", err)
		return err
	}

	logger.Info("Successfully applied blueprint!")

	return nil
}
