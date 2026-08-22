package cmd

import (
	"fmt"
	"os"

	"kana/storage"

	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <remote> [stream]",
	Short: "Push semantic snapshots to a remote repository",
	Long: `Transfers local snapshot graphs and AST nodes to a remote endpoint (local directory or SSH target).

Examples:
  kana push origin main
  kana push ubuntu main
  kana push user@10.18.0.97:/home/user/project main`,
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

		localHead, err := store.GetStreamHead(stream)
		if err != nil {
			return err
		}
		if localHead == "" {
			return fmt.Errorf("stream '%s' has no snapshots to push", stream)
		}

		loc, err := ResolveRemoteEndpoint(store, remoteTarget)
		if err != nil {
			return err
		}

		fmt.Printf("connecting to remote: %s (stream: %s)...\n", remoteTarget, stream)

		remoteHead, _ := GetRemoteStreamHead(loc, stream)

		if remoteHead == localHead {
			fmt.Println("everything up-to-date")
			return nil
		}

		bundle, err := store.ExportSyncBundle(stream, remoteHead)
		if err != nil {
			return fmt.Errorf("failed to create sync bundle: %w", err)
		}

		if len(bundle.Snapshots) == 0 {
			fmt.Println("everything up-to-date")
			return nil
		}

		fmt.Printf("pushing %d snapshot(s) and %d node(s)...\n", len(bundle.Snapshots), len(bundle.Nodes))

		if err := PushBundleToRemote(loc, bundle); err != nil {
			return err
		}

		shortHash := localHead
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}

		fmt.Printf("pushed stream '%s' -> %s (head: %s)\n", stream, remoteTarget, shortHash)
		return nil
	},
}
