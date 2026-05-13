package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "pdf",
	Short: "A friendly CLI for converting files to PDF",
	Long: `pdf is a command-line tool for converting documents to PDF.

Get started:
  pdf convert document.md`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		color.Red("Error: %v", err)
		os.Exit(1)
	}
}
