package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var diffPatch bool
var diffFileFilter string

var diffCmd = &cobra.Command{
	Use:   "diff [snapshot-a] [snapshot-b]",
	Short: "Show structural AST changes between snapshots or workspace",
	Long: `Compares semantic AST syntax trees between two historical snapshots, or between a snapshot and the current workspace.

Examples:
  kana diff                     # Diff workspace against current stream HEAD
  kana diff a1b2c3d4            # Diff workspace against snapshot a1b2c3d4
  kana diff a1b2c3d4 e5f6a7b8   # Diff snapshot a1b2c3d4 against e5f6a7b8
  kana diff -p                  # Show full AST node body patches`,
	Args: cobra.MaximumNArgs(2),
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
			return err
		}

		var oldAST, newAST map[string]*core.FileAST
		var labelA, labelB string

		switch len(args) {
		case 0:
			// Diff HEAD against Workspace
			headHash, err := store.GetStreamHead(stream)
			if err != nil {
				return err
			}
			labelA = "head"
			if headHash != "" {
				labelA = headHash[:min(12, len(headHash))]
			}
			labelB = "workspace"

			oldAST, err = store.GetSnapshotAST(headHash)
			if err != nil {
				return fmt.Errorf("failed to load head AST: %w", err)
			}

			newAST, err = core.ScanWorkspace(store.RepoPath)
			if err != nil {
				return fmt.Errorf("failed to scan workspace: %w", err)
			}

		case 1:
			// Diff specific snapshot against Workspace
			targetHash := args[0]
			snap, err := store.GetSnapshot(targetHash)
			if err != nil {
				return fmt.Errorf("snapshot %s not found: %w", targetHash, err)
			}

			labelA = snap.Hash[:min(12, len(snap.Hash))]
			labelB = "workspace"

			oldAST, err = store.GetSnapshotAST(snap.Hash)
			if err != nil {
				return err
			}

			newAST, err = core.ScanWorkspace(store.RepoPath)
			if err != nil {
				return err
			}

		case 2:
			// Diff snapshot A against snapshot B
			snapA, err := store.GetSnapshot(args[0])
			if err != nil {
				return fmt.Errorf("snapshot %s not found: %w", args[0], err)
			}
			snapB, err := store.GetSnapshot(args[1])
			if err != nil {
				return fmt.Errorf("snapshot %s not found: %w", args[1], err)
			}

			labelA = snapA.Hash[:min(12, len(snapA.Hash))]
			labelB = snapB.Hash[:min(12, len(snapB.Hash))]

			oldAST, err = store.GetSnapshotAST(snapA.Hash)
			if err != nil {
				return err
			}

			newAST, err = store.GetSnapshotAST(snapB.Hash)
			if err != nil {
				return err
			}
		}

		// Filter by file if specified
		if diffFileFilter != "" {
			filteredOld := make(map[string]*core.FileAST)
			filteredNew := make(map[string]*core.FileAST)
			if f, ok := oldAST[diffFileFilter]; ok {
				filteredOld[diffFileFilter] = f
			}
			if f, ok := newAST[diffFileFilter]; ok {
				filteredNew[diffFileFilter] = f
			}
			oldAST = filteredOld
			newAST = filteredNew
		}

		diff := core.DiffWorkspace(oldAST, newAST)

		fmt.Printf("diff %s..%s\n", labelA, labelB)

		var output string
		if diffPatch {
			output = core.FormatWorkspacePatch(diff)
		} else {
			output = core.FormatWorkspaceDiff(diff)
		}

		fmt.Print(output)
		return nil
	},
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func init() {
	diffCmd.Flags().BoolVarP(&diffPatch, "patch", "p", false, "Display full AST node body transformations")
	diffCmd.Flags().StringVar(&diffFileFilter, "file", "", "Filter diff by a specific file path")
}
