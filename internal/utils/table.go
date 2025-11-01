package utils

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintTable prints a column-aligned plain-text table in Docker/kubectl style.
// Uses Go's text/tabwriter with parameters matching modern CLI tools:
// minwidth=0, tabwidth=8, padding=2, padchar=' ', flags=0
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

