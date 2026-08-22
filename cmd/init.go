package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"kana/storage"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new Kanata semantic repository",
	Long:  "Creates the .kana/ directory and local embedded metadata database for tracking AST graphs.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		absDir, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("failed to resolve directory path: %w", err)
		}

		kanaPath := filepath.Join(absDir, storage.KanaDirName)
		if _, err := os.Stat(kanaPath); err == nil {
			return fmt.Errorf("kanata repository already initialized in %s", absDir)
		}

		store, err := storage.InitRepo(absDir)
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}
		defer store.Close()

		fmt.Printf("Initialized empty Kanata repository in %s\n", kanaPath)
		fmt.Println("Active work stream: main")
		return nil
	},
}
