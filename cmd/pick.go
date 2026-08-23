package cmd

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var pickFunction string
var pickFile string
var pickAutoSnapshot bool
var pickIntent string

var pickCmd = &cobra.Command{
	Use:   "pick <snapshot-hash>",
	Short: "Transplant semantic AST node transformations into workspace",
	Long: `Selectively transplants AST transformations (entire snapshot or specific function/type) from any snapshot into your active workspace.

Examples:
  kana pick 9963ad5e                   # Transplant all transformations from snapshot
  kana pick 9963ad5e -f ParseSource    # Transplant only the 'ParseSource' function
  kana pick 9963ad5e --file parser.go  # Transplant changes affecting only parser.go
  kana pick 9963ad5e -f Start -a       # Transplant and automatically snapshot`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetHash := args[0]

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

		currentHead, err := store.GetStreamHead(currentStream)
		if err != nil {
			return err
		}

		// Check if targetHash is a work stream name
		if sHead, sErr := store.GetStreamHead(targetHash); sErr == nil && sHead != "" {
			targetHash = sHead
		}

		targetSnap, err := store.GetSnapshot(targetHash)
		if err != nil {
			// Try prefix lookup
			allSnaps, lerr := store.GetAllSnapshots()
			if lerr == nil {
				for _, s := range allSnaps {
					if strings.HasPrefix(s.Hash, targetHash) {
						targetSnap = s
						break
					}
				}
			}
			if targetSnap == nil {
				return fmt.Errorf("snapshot '%s' not found: %w", targetHash, err)
			}
		}

		// Load target snapshot AST
		targetAST, err := store.GetSnapshotAST(targetSnap.Hash)
		if err != nil {
			return fmt.Errorf("failed to load target snapshot AST: %w", err)
		}

		// Load parent AST (or empty if root)
		var parentAST map[string]*core.FileAST
		if targetSnap.ParentHash != "" {
			parentAST, _ = store.GetSnapshotAST(targetSnap.ParentHash)
		}
		if parentAST == nil {
			parentAST = make(map[string]*core.FileAST)
		}

		// Compute delta introduced by target snapshot
		delta := core.DiffWorkspace(parentAST, targetAST)

		// Scan active workspace AST
		wsAST, err := core.ScanWorkspace(store.RepoPath)
		if err != nil {
			return fmt.Errorf("failed to scan workspace: %w", err)
		}

		// Apply selected node delta
		applied, err := core.ApplyNodeDelta(wsAST, delta, pickFile, pickFunction)
		if err != nil {
			return fmt.Errorf("failed to transplant AST nodes: %w", err)
		}

		if len(applied) == 0 {
			filterDesc := ""
			if pickFunction != "" {
				filterDesc += fmt.Sprintf(" matching function '%s'", pickFunction)
			}
			if pickFile != "" {
				filterDesc += fmt.Sprintf(" in file '%s'", pickFile)
			}
			fmt.Printf("no AST transformations found in snapshot %s%s\n", targetSnap.Hash[:8], filterDesc)
			return nil
		}

		// Materialize updated workspace to disk
		if err := core.MaterializeWorkspace(store.RepoPath, wsAST); err != nil {
			return fmt.Errorf("failed to write transplanted files to disk: %w", err)
		}

		shortHash := targetSnap.Hash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}

		fmt.Printf("transplanted %d AST node(s) from snapshot %s into workspace:\n\n", len(applied), shortHash)
		for _, a := range applied {
			prefix := "~"
			if a.ChangeType == core.ChangeAdded {
				prefix = "+"
			} else if a.ChangeType == core.ChangeRemoved {
				prefix = "-"
			}
			fmt.Printf("  %s %s (%s)\n", prefix, a.FilePath, a.Signature)
		}

		if pickAutoSnapshot || pickIntent != "" {
			intent := pickIntent
			if intent == "" {
				if pickFunction != "" {
					intent = fmt.Sprintf("Cherry-pick %s from %s", pickFunction, shortHash)
				} else {
					intent = fmt.Sprintf("Cherry-pick %s: %s", shortHash, targetSnap.Intent)
				}
			}

			author := "anonymous"
			if u, err := user.Current(); err == nil {
				author = u.Username
			}

			treeHash := storage.ComputeTreeHash(wsAST)
			now := time.Now().UTC()
			snapHash := storage.ComputeSnapshotHash(currentHead, currentStream, author, intent, now, treeHash)

			snap := &storage.Snapshot{
				Hash:       snapHash,
				ParentHash: currentHead,
				WorkStream: currentStream,
				Author:     author,
				Intent:     intent,
				Timestamp:  now,
				TreeHash:   treeHash,
			}

			if err := store.SaveSnapshot(snap, wsAST); err != nil {
				return fmt.Errorf("failed to record snapshot: %w", err)
			}

			snapShort := snapHash
			if len(snapShort) > 12 {
				snapShort = snapShort[:12]
			}
			fmt.Printf("\nrecorded snapshot: %s (stream: %s)\n", snapShort, currentStream)
		} else {
			fmt.Printf("\nworkspace updated with semantic cherry-pick (run 'kana snapshot -a' to record)\n")
		}

		return nil
	},
}

func init() {
	pickCmd.Flags().StringVarP(&pickFunction, "function", "f", "", "Filter cherry-pick to a specific function, struct, or node name")
	pickCmd.Flags().StringVar(&pickFile, "file", "", "Filter cherry-pick to a specific file path")
	pickCmd.Flags().BoolVarP(&pickAutoSnapshot, "auto", "a", false, "Automatically record snapshot after cherry-picking")
	pickCmd.Flags().StringVarP(&pickIntent, "intent", "i", "", "Intent message for automatic snapshot")
}
