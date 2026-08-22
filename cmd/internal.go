package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var internalCmd = &cobra.Command{
	Use:    "internal",
	Short:  "Internal plumbing helpers for transport synchronization",
	Hidden: true,
}

var internalStreamHeadCmd = &cobra.Command{
	Use:   "stream-head <repoDir> <stream>",
	Short: "Print head snapshot hash for a stream",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := args[0]
		stream := args[1]

		store, err := storage.OpenRepo(repoDir)
		if err != nil {
			// Remote directory not initialized yet -> no snapshots
			fmt.Println("")
			return nil
		}
		defer store.Close()

		head, _ := store.GetStreamHead(stream)
		fmt.Println(head)
		return nil
	},
}

var internalExportBundleCmd = &cobra.Command{
	Use:   "export-bundle <repoDir> <stream> [sinceHash]",
	Short: "Export a JSON sync bundle to stdout",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := args[0]
		stream := args[1]
		sinceHash := ""
		if len(args) > 2 {
			sinceHash = args[2]
		}

		store, err := storage.OpenRepo(repoDir)
		if err != nil {
			return err
		}
		defer store.Close()

		bundle, err := store.ExportSyncBundle(stream, sinceHash)
		if err != nil {
			return err
		}

		return json.NewEncoder(os.Stdout).Encode(bundle)
	},
}

var internalImportBundleCmd = &cobra.Command{
	Use:   "import-bundle <repoDir>",
	Short: "Import a JSON sync bundle from stdin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoDir := args[0]

		store, err := storage.OpenRepo(repoDir)
		if err != nil {
			// Auto-bootstrap repo if it doesn't exist yet!
			store, err = storage.InitRepo(repoDir)
			if err != nil {
				return fmt.Errorf("failed to auto-init remote repository at %s: %w", repoDir, err)
			}
		}
		defer store.Close()

		var bundle storage.SyncBundle
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read bundle from stdin: %w", err)
		}

		if err := json.Unmarshal(data, &bundle); err != nil {
			return fmt.Errorf("invalid sync bundle payload: %w", err)
		}

		if err := store.ImportSyncBundle(&bundle); err != nil {
			return err
		}

		// Materialize workspace files on remote
		if bundle.HeadHash != "" {
			newAST, err := store.GetSnapshotAST(bundle.HeadHash)
			if err == nil && len(newAST) > 0 {
				_ = core.MaterializeWorkspace(store.RepoPath, newAST)
			}
		}

		return nil
	},
}

func init() {
	internalCmd.AddCommand(internalStreamHeadCmd)
	internalCmd.AddCommand(internalExportBundleCmd)
	internalCmd.AddCommand(internalImportBundleCmd)
}
