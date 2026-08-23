package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var snapshotIntent string
var snapshotAuthor string
var snapshotAuto bool
var snapshotPatch bool

// Stdin source for interactive prompts (can be overridden in unit tests)
var snapshotPromptReader io.Reader = os.Stdin

var snapshotCmd = &cobra.Command{
	Use:   "snapshot [files...]",
	Short: "Record current AST graph transformation",
	Long: `Parses workspace files into AST nodes, persists structural transformations into the local database, and updates the stream head.

Examples:
  kana snapshot -i "Add User model" models/user.go
  kana snapshot -a                                   # Auto-infer intent from AST diff
  kana snapshot -p                                   # Interactively stage individual functions/nodes
  kana snapshot -i "Update auth handlers" handlers/auth.go middleware/jwt.go`,
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() {
			snapshotIntent = ""
			snapshotAuthor = ""
			snapshotAuto = false
			snapshotPatch = false
		}()

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

		if snapshotPatch {
			// Interactive function/node staging mode
			currentAST, err := core.ScanWorkspace(store.RepoPath)
			if err != nil {
				return fmt.Errorf("workspace scan failed: %w", err)
			}

			staged, stagedCount, err := runInteractiveStaging(parentAST, currentAST, snapshotPromptReader)
			if err != nil {
				return err
			}
			if stagedCount == 0 {
				fmt.Println("no AST nodes staged (snapshot aborted)")
				return nil
			}
			snapshotFiles = staged
		} else if len(args) > 0 {
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

		intent := snapshotIntent
		if intent == "" {
			intent = core.InferIntent(diff)
			fmt.Printf("inferred intent: %q\n", intent)
		}

		treeHash := storage.ComputeTreeHash(snapshotFiles)
		now := time.Now().UTC()
		snapHash := storage.ComputeSnapshotHash(parentHash, stream, author, intent, now, treeHash)

		snap := &storage.Snapshot{
			Hash:       snapHash,
			ParentHash: parentHash,
			WorkStream: stream,
			Author:     author,
			Intent:     intent,
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

func runInteractiveStaging(parentAST, currentAST map[string]*core.FileAST, in io.Reader) (map[string]*core.FileAST, int, error) {
	stagedAST := make(map[string]*core.FileAST)
	for p, f := range parentAST {
		clone := &core.FileAST{
			FilePath: f.FilePath,
			Language: f.Language,
			RawHash:  f.RawHash,
			Nodes:    make(map[string]core.ASTNode),
		}
		for id, n := range f.Nodes {
			clone.Nodes[id] = n
		}
		stagedAST[p] = clone
	}

	diff := core.DiffWorkspace(parentAST, currentAST)
	if len(diff.Files) == 0 {
		return stagedAST, 0, nil
	}

	scanner := bufio.NewScanner(in)
	stagedCount := 0
	stageAll := false

	for filePath, fDiff := range diff.Files {
		for _, nd := range fDiff.NodeDiffs {
			if stageAll {
				applyStagedNode(stagedAST, filePath, nd)
				stagedCount++
				continue
			}

			// Display node diff
			fmt.Printf("\n--- %s: %s (%s) ---\n", filePath, nd.Signature, nd.ChangeType)
			if nd.OldNode != nil && nd.NewNode != nil {
				oldLines := strings.Split(nd.OldNode.Content, "\n")
				newLines := strings.Split(nd.NewNode.Content, "\n")
				for _, lc := range core.DiffLines(oldLines, newLines) {
					switch lc.Type {
					case core.ChangeAdded:
						fmt.Printf("  + %s\n", lc.Text)
					case core.ChangeRemoved:
						fmt.Printf("  - %s\n", lc.Text)
					}
				}
			} else if nd.NewNode != nil {
				for _, line := range strings.Split(nd.NewNode.Content, "\n") {
					fmt.Printf("  + %s\n", line)
				}
			} else if nd.OldNode != nil {
				for _, line := range strings.Split(nd.OldNode.Content, "\n") {
					fmt.Printf("  - %s\n", line)
				}
			}

			// Prompt loop
			for {
				fmt.Printf("Stage %s '%s' in %s? [y,n,q,a,d,?]: ", nd.NodeType, nd.NodeName, filePath)
				if !scanner.Scan() {
					return stagedAST, stagedCount, nil
				}
				choice := strings.ToLower(strings.TrimSpace(scanner.Text()))

				if choice == "y" || choice == "yes" {
					applyStagedNode(stagedAST, filePath, nd)
					stagedCount++
					break
				} else if choice == "n" || choice == "no" {
					break
				} else if choice == "a" || choice == "all" {
					stageAll = true
					applyStagedNode(stagedAST, filePath, nd)
					stagedCount++
					break
				} else if choice == "d" || choice == "done" {
					return stagedAST, stagedCount, nil
				} else if choice == "q" || choice == "quit" {
					return stagedAST, 0, nil
				} else if choice == "?" || choice == "help" {
					fmt.Println("  y - stage this AST node")
					fmt.Println("  n - do not stage this AST node")
					fmt.Println("  a - stage this and all remaining AST nodes")
					fmt.Println("  d - finish staging with currently accepted nodes")
					fmt.Println("  q - abort snapshot entirely")
					fmt.Println("  ? - print help")
				} else {
					fmt.Println("invalid choice, type '?' for help")
				}
			}
		}
	}

	return stagedAST, stagedCount, nil
}

func applyStagedNode(stagedAST map[string]*core.FileAST, filePath string, nd core.NodeDiff) {
	fAST, ok := stagedAST[filePath]
	if !ok {
		fAST = &core.FileAST{
			FilePath: filePath,
			Nodes:    make(map[string]core.ASTNode),
		}
		stagedAST[filePath] = fAST
	}

	switch nd.ChangeType {
	case core.ChangeAdded, core.ChangeModified:
		if nd.NewNode != nil {
			fAST.Nodes[nd.NodeID] = *nd.NewNode
		}
	case core.ChangeRemoved:
		delete(fAST.Nodes, nd.NodeID)
	}

	var sb strings.Builder
	for _, n := range fAST.SortedNodes() {
		sb.WriteString(n.Hash)
	}
	fAST.RawHash = core.ComputeHash([]byte(sb.String()))
}

func init() {
	snapshotCmd.Flags().StringVarP(&snapshotIntent, "intent", "i", "", "Semantic intent description of this structural change")
	snapshotCmd.Flags().StringVar(&snapshotAuthor, "author", "", "Author or agent identifier")
	snapshotCmd.Flags().BoolVarP(&snapshotAuto, "auto", "a", false, "Automatically infer intent from AST changes")
	snapshotCmd.Flags().BoolVarP(&snapshotPatch, "patch", "p", false, "Interactively select and stage individual AST nodes")
}
