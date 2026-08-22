package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Scan workspace and analyze structural changes via AST diffing",
	Long:  "Scans all source files, constructs current ASTs, and compares them against the latest snapshot in the active work stream.",
	RunE: func(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("failed to retrieve current stream: %w", err)
		}

		headHash, err := store.GetStreamHead(stream)
		if err != nil {
			return fmt.Errorf("failed to get stream head: %w", err)
		}

		fmt.Printf("stream: %s\n", stream)
		if headHash != "" {
			snap, _ := store.GetSnapshot(headHash)
			shortHash := headHash
			if len(shortHash) > 12 {
				shortHash = shortHash[:12]
			}
			intent := ""
			if snap != nil {
				intent = fmt.Sprintf(" - %s", snap.Intent)
			}
			fmt.Printf("head:   %s%s\n\n", shortHash, intent)
		} else {
			fmt.Println("head:   (no snapshots yet)")
			fmt.Println()
		}

		headAST, err := store.GetSnapshotAST(headHash)
		if err != nil {
			return fmt.Errorf("failed to load snapshot AST: %w", err)
		}

		currentAST, err := core.ScanWorkspace(store.RepoPath)
		if err != nil {
			return fmt.Errorf("failed to scan workspace: %w", err)
		}

		diff := core.DiffWorkspace(headAST, currentAST)
		output := core.FormatWorkspaceDiff(diff)
		fmt.Print(output)

		return nil
	},
}
