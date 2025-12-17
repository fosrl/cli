package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fosrl/cli/cmd"
	"github.com/spf13/cobra/doc"
)

const fmTemplate = `---
date: %s
title: "%s"
slug: %s
url: %s
---

`

func main() {
	var (
		outputDir       = flag.String("dir", "./docs", "Output directory for generated documentation")
		withFrontMatter = flag.Bool("frontmatter", false, "Add Hugo front matter to generated files")
		baseURL         = flag.String("baseurl", "/commands", "Base URL for command links (used with front matter)")
	)
	flag.Parse()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Get the root command
	rootCmd, _ := cmd.RootCommand(false)

	var err error
	if *withFrontMatter {
		// Generate with custom file prepender and link handler
		filePrepender := func(filename string) string {
			now := time.Now().Format(time.RFC3339)
			name := filepath.Base(filename)
			base := strings.TrimSuffix(name, path.Ext(name))
			url := *baseURL + "/" + strings.ToLower(base) + "/"
			title := strings.ReplaceAll(base, "_", " ")
			return fmt.Sprintf(fmTemplate, now, title, base, url)
		}

		linkHandler := func(name string) string {
			base := strings.TrimSuffix(name, path.Ext(name))
			return *baseURL + "/" + strings.ToLower(base) + "/"
		}

		err = doc.GenMarkdownTreeCustom(rootCmd, *outputDir, filePrepender, linkHandler)
	} else {
		// Generate standard markdown without front matter
		err = doc.GenMarkdownTree(rootCmd, *outputDir)
	}

	if err != nil {
		log.Fatalf("Failed to generate markdown docs: %v", err)
	}

	log.Printf("Successfully generated markdown documentation in %s", *outputDir)

	// List generated files
	files, err := filepath.Glob(filepath.Join(*outputDir, "*.md"))
	if err == nil {
		log.Printf("Generated %d documentation files:", len(files))
		for _, file := range files {
			log.Printf("  - %s", filepath.Base(file))
		}
	}
}
