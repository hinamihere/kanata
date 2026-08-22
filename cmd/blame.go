package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"kana/storage"

	"github.com/spf13/cobra"
)

var blameFunction string
var blameLimit int

var blameCmd = &cobra.Command{
	Use:   "blame <file>",
	Short: "Show semantic history and author attribution per AST node",
	Long: `Traces who and which snapshot last modified each semantic function, struct, or macro in a file, ignoring whitespace and formatting churn.

Examples:
  kana blame main.c                  # Blame all AST nodes in main.c
  kana blame main.c -f add           # Trace evolution timeline of function 'add'
  kana blame service.py -f compute   # Trace evolution timeline of function 'compute'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetFile := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		store, err := storage.OpenRepo(cwd)
		if err != nil {
			return err
		}
		defer store.Close()

		stream, err := store.GetCurrentStream()
		if err != nil {
			return err
		}

		absTarget, err := filepath.Abs(targetFile)
		if err != nil {
			return err
		}
		relTarget, err := filepath.Rel(store.RepoPath, absTarget)
		if err != nil {
			return err
		}
		relTarget = filepath.ToSlash(relTarget)

		if blameFunction != "" {
			// Single node evolution history
			history, err := store.GetNodeHistory(relTarget, blameFunction, blameLimit)
			if err != nil {
				return err
			}
			if len(history) == 0 {
				fmt.Printf("no history found for '%s' in %s\n", blameFunction, relTarget)
				return nil
			}

			fmt.Printf("evolution of %s in %s (%d versions):\n\n", blameFunction, relTarget, len(history))
			for _, entry := range history {
				shortHash := entry.SnapshotHash
				if len(shortHash) > 12 {
					shortHash = shortHash[:12]
				}
				fmt.Printf("  snapshot %s  (%s)\n", shortHash, entry.Timestamp.Format("2006-01-02 15:04:05"))
				fmt.Printf("    author: %s\n", entry.Author)
				fmt.Printf("    intent: %s\n", entry.Intent)
				fmt.Printf("    hash:   %s\n\n", entry.NodeHash[:min(12, len(entry.NodeHash))])
			}
			return nil
		}

		// File-level node blame breakdown
		blameEntries, err := store.GetFileBlame(relTarget, stream)
		if err != nil {
			return err
		}

		fmt.Printf("blame %s (stream: %s)\n\n", relTarget, stream)
		for _, b := range blameEntries {
			shortHash := b.SnapshotHash
			if len(shortHash) > 12 {
				shortHash = shortHash[:12]
			}
			fmt.Printf("  %-32s  %s  %s  %q\n",
				b.Signature,
				shortHash,
				b.Timestamp.Format("2006-01-02"),
				b.Intent,
			)
		}

		return nil
	},
}

func init() {
	blameCmd.Flags().StringVarP(&blameFunction, "function", "f", "", "Filter to a specific function or struct name")
	blameCmd.Flags().IntVarP(&blameLimit, "limit", "n", 20, "Maximum evolution entries to display")
}
