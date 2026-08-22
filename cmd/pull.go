package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <remote> [stream]",
	Short: "Pull semantic snapshots from a remote repository",
	Long: `Fetches missing snapshot graphs and AST nodes from a remote endpoint and updates the working tree.

Examples:
  kana pull origin main
  kana pull ubuntu main
  kana pull user@10.18.0.97:/home/user/project main`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteTarget := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		store, err := storage.OpenRepo(cwd)
		if err != nil {
			return err
		}
		defer store.Close()

		stream := ""
		if len(args) > 1 {
			stream = args[1]
		} else {
			stream, err = store.GetCurrentStream()
			if err != nil {
				return err
			}
		}

		localHead, _ := store.GetStreamHead(stream)

		loc, err := ResolveRemoteEndpoint(store, remoteTarget)
		if err != nil {
			return err
		}

		fmt.Printf("connecting to remote: %s (stream: %s)...\n", remoteTarget, stream)

		bundle, err := FetchBundleFromRemote(loc, stream, localHead)
		if err != nil {
			return err
		}

		if len(bundle.Snapshots) == 0 {
			fmt.Println("already up-to-date")
			return nil
		}

		fmt.Printf("received %d snapshot(s) and %d node(s)...\n", len(bundle.Snapshots), len(bundle.Nodes))

		if err := store.ImportSyncBundle(bundle); err != nil {
			return fmt.Errorf("failed to import bundle: %w", err)
		}

		// Reconstruct and materialize files from new head
		newHead := bundle.HeadHash
		newAST, err := store.GetSnapshotAST(newHead)
		if err != nil {
			return fmt.Errorf("failed to reconstruct snapshot AST: %w", err)
		}

		if err := core.MaterializeWorkspace(store.RepoPath, newAST); err != nil {
			return fmt.Errorf("failed to update workspace files: %w", err)
		}

		shortHash := newHead
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}

		fmt.Printf("pulled stream '%s' from %s (head: %s)\n", stream, remoteTarget, shortHash)
		return nil
	},
}
