package apply

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

type ApplyBlueprintCmdOpts struct {
	Name string
	Path string
}

func ApplyBlueprintCommand() *cobra.Command {
	opts := ApplyBlueprintCmdOpts{}

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply a blueprint",
		Long:  "Apply a YAML blueprint to the Pangolin server",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}

			opts.Path = args[0]

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

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Name of blueprint (default: filename, without extension)")

	return cmd
}

func applyBlueprintMain(cmd *cobra.Command, opts ApplyBlueprintCmdOpts) error {
	api := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	if _, err := accountStore.ActiveAccount(); err != nil {
		logger.Error("Error: %v", err)
		return err
	}

	blueprintContents, err := os.ReadFile(opts.Name)
	if err != nil {
		logger.Error("Error: failed to read blueprint file: %v", err)
		return err
	}

	_, err = api.ApplyBlueprint(opts.Name, string(blueprintContents))
	if err != nil {
		logger.Error("Error: failed to apply blueprint: %v", err)
		return err
	}

	logger.Info("Successfully applied blueprint!")

	return nil
}
