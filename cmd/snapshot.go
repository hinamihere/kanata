package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var snapshotIntent string
var snapshotAuthor string

var snapshotCmd = &cobra.Command{
	Use:   "snapshot [files...]",
	Short: "Record current AST graph transformation",
	Long: `Parses workspace files into AST nodes, persists structural transformations into the local database, and updates the stream head.

Optionally specify one or more file paths to selectively snapshot only those files:
  kana snapshot -i "Add User model" models/user.go
  kana snapshot -i "Update auth handlers" handlers/auth.go middleware/jwt.go`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if snapshotIntent == "" {
			return fmt.Errorf("an intent message is required (use --intent or -i)")
		}

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
			return fmt.Errorf("failed to get current stream: %w", err)
		}

		parentHash, _ := store.GetStreamHead(stream)

		author := snapshotAuthor
		if author == "" {
			if u, err := user.Current(); err == nil {
				author = u.Username
			} else {
				author = "kanata-user"
			}
		}

		parentAST, err := store.GetSnapshotAST(parentHash)
		if err != nil {
			return fmt.Errorf("failed to read parent AST: %w", err)
		}

		var snapshotFiles map[string]*core.FileAST

		if len(args) > 0 {
			snapshotFiles = make(map[string]*core.FileAST)
			for p, fast := range parentAST {
				snapshotFiles[p] = fast
			}

			for _, fileArg := range args {
				absPath, err := filepath.Abs(fileArg)
				if err != nil {
					return fmt.Errorf("invalid file path %s: %w", fileArg, err)
				}

				relPath, err := filepath.Rel(store.RepoPath, absPath)
				if err != nil {
					return fmt.Errorf("file %s is outside repository: %w", fileArg, err)
				}
				relPath = filepath.ToSlash(relPath)

				if _, err := os.Stat(absPath); os.IsNotExist(err) {
					delete(snapshotFiles, relPath)
				} else {
					fAST, err := core.ParseFile(absPath)
					if err != nil {
						return fmt.Errorf("failed to parse %s: %w", relPath, err)
					}
					fAST.FilePath = relPath
					snapshotFiles[relPath] = fAST
				}
			}
		} else {
			currentAST, err := core.ScanWorkspace(store.RepoPath)
			if err != nil {
				return fmt.Errorf("workspace scan failed: %w", err)
			}
			snapshotFiles = currentAST
		}

		diff := core.DiffWorkspace(parentAST, snapshotFiles)
		if len(diff.Files) == 0 && parentHash != "" {
			fmt.Println("nothing to snapshot (workspace AST matches stream head)")
			return nil
		}

		treeHash := storage.ComputeTreeHash(snapshotFiles)
		now := time.Now().UTC()
		snapHash := storage.ComputeSnapshotHash(parentHash, stream, author, snapshotIntent, now, treeHash)

		snap := &storage.Snapshot{
			Hash:       snapHash,
			ParentHash: parentHash,
			WorkStream: stream,
			Author:     author,
			Intent:     snapshotIntent,
			Timestamp:  now,
			TreeHash:   treeHash,
		}

		if err := store.SaveSnapshot(snap, snapshotFiles); err != nil {
			return fmt.Errorf("failed to persist snapshot: %w", err)
		}

		shortHash := snapHash
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}

		fmt.Printf("snapshot: %s (stream: %s)\n", shortHash, stream)
		fmt.Printf("  intent: %s\n", snap.Intent)
		fmt.Printf("  author: %s\n", snap.Author)
		fmt.Printf("  delta:  +%d added, ~%d modified, -%d removed (%d files)\n",
			diff.AddedNodesCount, diff.ModifiedNodesCount, diff.RemovedNodesCount, len(diff.Files))

		return nil
	},
}

func init() {
	snapshotCmd.Flags().StringVarP(&snapshotIntent, "intent", "i", "", "Semantic intent description of this structural change")
	snapshotCmd.Flags().StringVarP(&snapshotAuthor, "author", "a", "", "Author or agent identifier")
	_ = snapshotCmd.MarkFlagRequired("intent")
}
