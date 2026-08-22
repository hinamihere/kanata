package cmd

import (
	"fmt"
	"os"

	"kana/storage"

	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote repository tracking endpoints",
	Long:  "View, add, or remove remote repository endpoints for SSH and local directory synchronization.",
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

		remotes, err := store.ListRemotes()
		if err != nil {
			return err
		}

		if len(remotes) == 0 {
			fmt.Println("no remotes configured (use 'kana remote add <name> <url>')")
			return nil
		}

		fmt.Println("configured remotes:")
		for name, url := range remotes {
			fmt.Printf("  %-12s  %s\n", name, url)
		}
		return nil
	},
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a remote repository endpoint",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		url := args[1]

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		store, err := storage.OpenRepo(cwd)
		if err != nil {
			return err
		}
		defer store.Close()

		if err := store.AddRemote(name, url); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}

		fmt.Printf("added remote '%s' -> %s\n", name, url)
		return nil
	},
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote repository endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		store, err := storage.OpenRepo(cwd)
		if err != nil {
			return err
		}
		defer store.Close()

		if err := store.RemoveRemote(name); err != nil {
			return fmt.Errorf("failed to remove remote: %w", err)
		}

		fmt.Printf("removed remote '%s'\n", name)
		return nil
	},
}

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
}
