package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest an intent message based on AST changes",
	Long:  "Analyzes pending semantic AST transformations in the workspace and generates clear, intent-driven message recommendations.",
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

		headHash, err := store.GetStreamHead(stream)
		if err != nil {
			return err
		}

		headAST, err := store.GetSnapshotAST(headHash)
		if err != nil {
			return err
		}

		wsAST, err := core.ScanWorkspace(store.RepoPath)
		if err != nil {
			return err
		}

		diff := core.DiffWorkspace(headAST, wsAST)
		if len(diff.Files) == 0 {
			fmt.Println("working tree clean (no AST changes to describe)")
			return nil
		}

		intent := core.InferIntent(diff)
		fmt.Printf("suggested intent: %q\n\n", intent)
		fmt.Println("to snapshot with this intent, run:")
		fmt.Printf("  kana snapshot -a\n")
		return nil
	},
}
