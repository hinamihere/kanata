package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kana",
	Short: "Kanata (kana) is a high-performance semantic version control system",
	Long: `Kanata (kana) is a next-generation, AST-based semantic version control system.
Unlike legacy text-diff VCS tools, Kanata parses code into Abstract Syntax Trees (ASTs),
tracks structural node transformations, and replaces legacy git terminology with intent-driven workflows:

  kana init        - Initialize a semantic repository
  kana status      - Analyze structural changes via AST diffing
  kana focus       - Switch or create semantic work streams
  kana park        - Temporarily park AST changes out of workspace
  kana integrate   - Merge semantic node transformations from another stream
  kana rewind      - Rewind workspace to a prior semantic snapshot
  kana snapshot    - Record current AST graph transformation with an intent message`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(focusCmd)
	rootCmd.AddCommand(parkCmd)
	rootCmd.AddCommand(integrateCmd)
	rootCmd.AddCommand(rewindCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(blameCmd)
	rootCmd.AddCommand(remoteCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(suggestCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(internalCmd)
}
