package cmd

import (
	"fmt"
	"os"
	"time"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var integrateCmd = &cobra.Command{
	Use:   "integrate <stream-name>",
	Short: "Merge semantic node transformations from another stream",
	Long:  "Integrates semantic AST graphs from a target work stream into the active stream using structural 3-way reconciliation.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetStream := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		store, err := storage.OpenRepo(cwd)
		if err != nil {
			return err
		}
		defer store.Close()

		currentStream, err := store.GetCurrentStream()
		if err != nil {
			return err
		}

		if currentStream == targetStream {
			return fmt.Errorf("cannot integrate stream '%s' into itself", targetStream)
		}

		currentHead, err := store.GetStreamHead(currentStream)
		if err != nil {
			return err
		}

		targetHead, err := store.GetStreamHead(targetStream)
		if err != nil {
			return err
		}

		if targetHead == "" {
			fmt.Printf("target stream '%s' has no recorded snapshots\n", targetStream)
			return nil
		}

		if currentHead == targetHead {
			fmt.Println("already up-to-date (no integration needed)")
			return nil
		}

		currentAST, err := store.GetSnapshotAST(currentHead)
		if err != nil {
			return fmt.Errorf("failed to load current AST: %w", err)
		}

		targetAST, err := store.GetSnapshotAST(targetHead)
		if err != nil {
			return fmt.Errorf("failed to load target AST: %w", err)
		}

		baseAST := make(map[string]*core.FileAST)
		currSnap, _ := store.GetSnapshot(currentHead)
		if currSnap != nil && currSnap.ParentHash != "" {
			baseAST, _ = store.GetSnapshotAST(currSnap.ParentHash)
		}

		mergeResult := core.IntegrateWorkspaces(baseAST, currentAST, targetAST)

		if mergeResult.HasConflict {
			fmt.Printf("conflict: semantic AST conflicts detected during integration with '%s':\n\n", targetStream)
			for _, conf := range mergeResult.Conflicts {
				fmt.Printf("  conflict in %s: %s\n", conf.NodeType, conf.Signature)
				if conf.OldNode != nil {
					fmt.Printf("    our version:   hash %s (lines %d-%d)\n", conf.OldNode.Hash[:8], conf.OldNode.StartLine, conf.OldNode.EndLine)
				}
				if conf.NewNode != nil {
					fmt.Printf("    their version: hash %s (lines %d-%d)\n", conf.NewNode.Hash[:8], conf.NewNode.StartLine, conf.NewNode.EndLine)
				}
			}
			fmt.Println("\nresolve semantic differences before completing integration.")
			return nil
		}

		if err := core.MaterializeWorkspace(store.RepoPath, mergeResult.MergedState); err != nil {
			return fmt.Errorf("failed to write merged files: %w", err)
		}

		treeHash := storage.ComputeTreeHash(mergeResult.MergedState)
		now := time.Now().UTC()
		intent := fmt.Sprintf("Integrate stream '%s' into '%s'", targetStream, currentStream)
		snapHash := storage.ComputeSnapshotHash(currentHead, currentStream, "kanata-integrator", intent, now, treeHash)

		snap := &storage.Snapshot{
			Hash:       snapHash,
			ParentHash: currentHead,
			WorkStream: currentStream,
			Author:     "kanata-integrator",
			Intent:     intent,
			Timestamp:  now,
			TreeHash:   treeHash,
		}

		if err := store.SaveSnapshot(snap, mergeResult.MergedState); err != nil {
			return fmt.Errorf("failed to save integrated snapshot: %w", err)
		}

		shortHash := snapHash
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}

		fmt.Printf("integrated stream '%s' into '%s' (snapshot: %s)\n", targetStream, currentStream, shortHash)
		fmt.Printf("  reconciled %d files cleanly at AST level\n", len(mergeResult.MergedState))

		return nil
	},
}
