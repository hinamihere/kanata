package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var rewindSteps int

var rewindCmd = &cobra.Command{
	Use:   "rewind [snapshot-hash]",
	Short: "Rewind workspace state to a prior semantic snapshot",
	Long:  "Rolls back workspace files and stream HEAD pointer to a previously recorded semantic snapshot graph.",
	Args:  cobra.MaximumNArgs(1),
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

		var targetHash string
		if len(args) > 0 {
			targetHash = args[0]
		} else if rewindSteps > 0 {
			snapshots, err := store.ListSnapshots(stream, rewindSteps+1)
			if err != nil {
				return fmt.Errorf("failed to traverse snapshot graph: %w", err)
			}
			if len(snapshots) <= rewindSteps {
				return fmt.Errorf("cannot rewind %d steps: only %d snapshots exist in stream", rewindSteps, len(snapshots))
			}
			targetHash = snapshots[rewindSteps].Hash
		} else {
			return fmt.Errorf("specify a target snapshot hash or use --steps <n>")
		}

		targetSnap, err := store.GetSnapshot(targetHash)
		if err != nil {
			return fmt.Errorf("failed to find snapshot %s: %w", targetHash, err)
		}

		snapshotAST, err := store.GetSnapshotAST(targetSnap.Hash)
		if err != nil {
			return fmt.Errorf("failed to reconstruct snapshot AST: %w", err)
		}

		if err := core.MaterializeWorkspace(store.RepoPath, snapshotAST); err != nil {
			return fmt.Errorf("failed to restore workspace files: %w", err)
		}

		if err := store.SetStreamHead(stream, targetSnap.Hash); err != nil {
			return fmt.Errorf("failed to update stream head: %w", err)
		}

		shortHash := targetSnap.Hash
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}

		fmt.Printf("rewound workspace to snapshot %s\n", shortHash)
		fmt.Printf("  stream:    %s\n", stream)
		fmt.Printf("  intent:    %s\n", targetSnap.Intent)
		fmt.Printf("  timestamp: %s\n", targetSnap.Timestamp.Format("2006-01-02 15:04:05"))

		return nil
	},
}

func init() {
	rewindCmd.Flags().IntVarP(&rewindSteps, "steps", "s", 0, "Number of snapshots to rewind backward")
}
