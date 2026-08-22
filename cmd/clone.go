package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <remote-url> [directory]",
	Short: "Clone a remote Kanata semantic repository",
	Long: `Clones a complete semantic snapshot graph from a remote location (local path or SSH endpoint) and materializes the workspace files.

Examples:
  kana clone user@10.18.0.97:/home/user/project ./my_project
  kana clone ../original_repo ./cloned_repo`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteURL := args[0]
		targetDir := ""

		if len(args) > 1 {
			targetDir = args[1]
		} else {
			// Extract folder name from URL/path
			trimmed := strings.TrimRight(remoteURL, "/")
			parts := strings.Split(trimmed, "/")
			last := parts[len(parts)-1]
			if strings.Contains(last, ":") {
				sub := strings.Split(last, ":")
				last = sub[len(sub)-1]
			}
			targetDir = last
			if targetDir == "" || targetDir == "." {
				targetDir = "kanata_project"
			}
		}

		absTarget, err := filepath.Abs(targetDir)
		if err != nil {
			return err
		}

		if fi, err := os.Stat(absTarget); err == nil && fi.IsDir() {
			entries, _ := os.ReadDir(absTarget)
			if len(entries) > 0 {
				return fmt.Errorf("target directory %s already exists and is not empty", absTarget)
			}
		}

		fmt.Printf("cloning into '%s'...\n", targetDir)

		store, err := storage.InitRepo(absTarget)
		if err != nil {
			return fmt.Errorf("failed to initialize repository: %w", err)
		}
		defer store.Close()

		loc, err := ParseRemoteLocation(remoteURL)
		if err != nil {
			return err
		}

		// Discover remote stream head
		headHash, err := GetRemoteStreamHead(loc, "main")
		if err != nil {
			return fmt.Errorf("failed to discover remote stream: %w", err)
		}

		if headHash == "" {
			fmt.Println("warning: cloned empty repository (no snapshots recorded)")
			_ = store.AddRemote("origin", remoteURL)
			return nil
		}

		// Fetch full bundle
		bundle, err := FetchBundleFromRemote(loc, "main", "")
		if err != nil {
			return fmt.Errorf("failed to fetch bundle: %w", err)
		}

		fmt.Printf("receiving %d snapshot(s) and %d node(s)...\n", len(bundle.Snapshots), len(bundle.Nodes))

		if err := store.ImportSyncBundle(bundle); err != nil {
			return fmt.Errorf("failed to import bundle: %w", err)
		}

		// Add origin remote
		_ = store.AddRemote("origin", remoteURL)

		// Materialize workspace files
		newAST, err := store.GetSnapshotAST(headHash)
		if err != nil {
			return fmt.Errorf("failed to load snapshot AST: %w", err)
		}

		if err := core.MaterializeWorkspace(absTarget, newAST); err != nil {
			return fmt.Errorf("failed to materialize files: %w", err)
		}

		shortHash := headHash
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}

		fmt.Printf("cloned repository successfully (head: %s, stream: main)\n", shortHash)
		return nil
	},
}
