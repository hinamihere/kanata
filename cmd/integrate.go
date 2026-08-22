package cmd

import (
	"fmt"
	"os"
	"time"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var integrateOurs bool
var integrateTheirs bool

var integrateCmd = &cobra.Command{
	Use:   "integrate <stream-name>",
	Short: "Merge semantic node transformations from another stream",
	Long: `Integrates semantic AST graphs from a target work stream into the active stream using structural 3-way reconciliation.

Examples:
  kana integrate dev2               # Attempt automatic AST integration or write conflict markers
  kana integrate dev2 --ours        # Resolve all conflicts in favor of current stream
  kana integrate dev2 --theirs      # Resolve all conflicts in favor of incoming stream`,
	Args: cobra.ExactArgs(1),
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
			if integrateOurs {
				// Accept our version for all conflicting nodes
				for p, fAST := range mergeResult.MergedState {
					for _, conf := range mergeResult.Conflicts {
						if conf.OldNode != nil {
							fAST.Nodes[conf.NodeID] = *conf.OldNode
						}
					}
					mergeResult.MergedState[p] = fAST
				}
				fmt.Printf("resolved %d conflict(s) using --ours\n", len(mergeResult.Conflicts))
			} else if integrateTheirs {
				// Accept their version for all conflicting nodes
				for p, fAST := range mergeResult.MergedState {
					for _, conf := range mergeResult.Conflicts {
						if conf.NewNode != nil {
							fAST.Nodes[conf.NodeID] = *conf.NewNode
						}
					}
					mergeResult.MergedState[p] = fAST
				}
				fmt.Printf("resolved %d conflict(s) using --theirs\n", len(mergeResult.Conflicts))
			} else {
				// Embed conflict markers in conflicting nodes and write to workspace
				for p, fAST := range mergeResult.MergedState {
					for _, conf := range mergeResult.Conflicts {
						if _, ok := fAST.Nodes[conf.NodeID]; ok {
							oldContent := ""
							newContent := ""
							if conf.OldNode != nil {
								oldContent = conf.OldNode.Content
							}
							if conf.NewNode != nil {
								newContent = conf.NewNode.Content
							}
							markedContent := fmt.Sprintf("<<<<<<< current (%s)\n%s\n=======\n%s\n>>>>>>> incoming (%s)",
								currentStream, oldContent, newContent, targetStream)

							confNode := fAST.Nodes[conf.NodeID]
							confNode.Content = markedContent
							fAST.Nodes[conf.NodeID] = confNode
						}
					}
					mergeResult.MergedState[p] = fAST
				}

				_ = core.MaterializeWorkspace(store.RepoPath, mergeResult.MergedState)

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
				fmt.Println("\nconflict markers written to workspace files.")
				fmt.Println("edit the files to resolve, then run:")
				fmt.Println("  kana snapshot -a")
				fmt.Println("or re-run integration with:")
				fmt.Printf("  kana integrate %s --ours\n", targetStream)
				fmt.Printf("  kana integrate %s --theirs\n", targetStream)
				return nil
			}
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

func init() {
	integrateCmd.Flags().BoolVar(&integrateOurs, "ours", false, "Resolve all conflicts using current stream's version")
	integrateCmd.Flags().BoolVar(&integrateTheirs, "theirs", false, "Resolve all conflicts using incoming stream's version")
}
