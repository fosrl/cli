package blueprint

import (
	"errors"
	"fmt"
	"io"
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
	Paths    []string
	APIKey   string
	Endpoint string
	OrgID    string
}

func BlueprintCmd() *cobra.Command {
	opts := BlueprintCmdOpts{}

	cmd := &cobra.Command{
		Use:   "blueprint",
		Short: "Apply a blueprint",
		Long:  "Apply a YAML blueprint to the Pangolin server. --file may be a glob pattern (e.g. -f 'inference-*.yaml') or given multiple times; every matching file is applied one by one.",
		Args:  cobra.ArbitraryArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Integration API: any of the three flags implies all three are required (avoids silent session fallback).
			integration := opts.APIKey != "" || opts.Endpoint != "" || opts.OrgID != ""
			if integration && (opts.APIKey == "" || opts.Endpoint == "" || opts.OrgID == "") {
				return errors.New("integration API mode requires --api-key, --endpoint, and --org together; omit all three to use your logged-in session and selected org")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Extra positional args show up when the shell expands a glob (e.g. -f inference-*)
			// before we ever see it; fold them in alongside anything passed via -f itself.
			paths, err := resolveBlueprintPaths(append(append([]string{}, opts.Paths...), args...))
			if err != nil {
				return err
			}

			if opts.Name != "" && len(paths) > 1 {
				return errors.New("--name cannot be used when multiple blueprint files match; the name is derived from each filename instead")
			}

			for _, path := range paths {
				if err := applyBlueprintMain(cmd, opts, path); err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				logger.Info("Successfully applied blueprint: %s", path)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&opts.Paths, "file", "f", nil, "Blueprint YAML file path, or glob pattern (e.g. 'inference-*.yaml'); repeatable. Use '-' for stdin")
	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "Blueprint name (default: filename without extension); only valid for a single file")
	cmd.Flags().StringVar(&opts.APIKey, "api-key", "", "Integration API key (id.secret)")
	cmd.Flags().StringVar(&opts.Endpoint, "endpoint", "", "Integration API host URL")
	cmd.Flags().StringVar(&opts.OrgID, "org", "", "Organization ID")
	cmd.MarkFlagRequired("file")

	return cmd
}

// resolveBlueprintPaths expands any glob patterns among the given tokens into
// concrete file paths, passes "-" (stdin) and non-glob paths through as-is,
// and dedupes the result while preserving order.
func resolveBlueprintPaths(tokens []string) ([]string, error) {
	seen := make(map[string]bool)
	var resolved []string

	add := func(path string) {
		if !seen[path] {
			seen[path] = true
			resolved = append(resolved, path)
		}
	}

	for _, token := range tokens {
		if token == "-" || !strings.ContainsAny(token, "*?[") {
			add(token)
			continue
		}

		matches, err := filepath.Glob(token)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", token, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files matched pattern %q", token)
		}
		for _, m := range matches {
			add(m)
		}
	}

	if len(resolved) == 0 {
		return nil, errors.New("no blueprint files specified")
	}
	return resolved, nil
}

func applyBlueprintMain(cmd *cobra.Command, opts BlueprintCmdOpts, path string) error {
	if path == "-" && strings.TrimSpace(opts.Name) == "" {
		return errors.New("name is required when using --file -")
	}

	name := opts.Name
	if name == "" {
		filename := filepath.Base(path)
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

	blueprintContents, err := readBlueprint(path)
	if err != nil {
		return err
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

func readBlueprint(path string) ([]byte, error) {
	if path == "-" {
		fileInfo, err := os.Stdin.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to inspect stdin: %w", err)
		}
		if (fileInfo.Mode() & os.ModeCharDevice) == os.ModeCharDevice {
			return nil, errors.New("the option --file - is intended to work with pipes")
		}

		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read blueprint from stdin: %w", err)
		}
		if len(contents) == 0 {
			return nil, errors.New("blueprint input is empty")
		}
		return contents, nil
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read blueprint file: %w", err)
	}
	return contents, nil
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
