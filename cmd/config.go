package cmd

import (
	"fmt"
	"os"

	"kana/storage"

	"github.com/spf13/cobra"
)

var configGlobal bool
var configList bool
var configUnset bool

var configCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Get and set repository or global options (user.name, user.email, etc.)",
	Long: `Manages local repository and global user configuration settings.

Examples:
  kana config --global user.name "Hoshino Hinami"
  kana config --global user.email "hinami@example.com"
  kana config user.name "Project Lead"               # Local repository override
  kana config user.name                              # Read current value
  kana config --list                                 # List all configured settings`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		// Mode 1: List all configurations
		if configList {
			fmt.Println("global configuration (~/.kanaconfig):")
			gCfg, _ := storage.LoadGlobalConfig()
			if len(gCfg) == 0 {
				fmt.Println("  (no global settings)")
			} else {
				for k, v := range gCfg {
					fmt.Printf("  %s = %s\n", k, v)
				}
			}

			store, err := storage.OpenRepo(cwd)
			if err == nil {
				defer store.Close()
				fmt.Println("\nlocal repository configuration (.kana/):")
				lCfg, _ := store.ListAllConfig()
				if len(lCfg) == 0 {
					fmt.Println("  (no local overrides)")
				} else {
					for k, v := range lCfg {
						fmt.Printf("  %s = %s\n", k, v)
					}
				}
			}
			return nil
		}

		if len(args) == 0 {
			return cmd.Help()
		}

		key := args[0]

		// Mode 2: Unset key
		if configUnset {
			if configGlobal {
				if err := storage.SetGlobalConfig(key, ""); err != nil {
					return fmt.Errorf("failed to unset global config: %w", err)
				}
				fmt.Printf("unset global config '%s'\n", key)
				return nil
			}

			store, err := storage.OpenRepo(cwd)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.SetConfig(key, ""); err != nil {
				return fmt.Errorf("failed to unset local config: %w", err)
			}
			fmt.Printf("unset local config '%s'\n", key)
			return nil
		}

		// Mode 3: Set key value
		if len(args) >= 2 {
			value := args[1]
			if configGlobal {
				if err := storage.SetGlobalConfig(key, value); err != nil {
					return fmt.Errorf("failed to set global config: %w", err)
				}
				fmt.Printf("set global %s = %s\n", key, value)
				return nil
			}

			store, err := storage.OpenRepo(cwd)
			if err != nil {
				return err
			}
			defer store.Close()

			if err := store.SetConfig(key, value); err != nil {
				return fmt.Errorf("failed to set local config: %w", err)
			}
			fmt.Printf("set local %s = %s\n", key, value)
			return nil
		}

		// Mode 4: Get key value
		if configGlobal {
			gCfg, _ := storage.LoadGlobalConfig()
			val, ok := gCfg[key]
			if !ok || val == "" {
				fmt.Printf("global config '%s' is not set\n", key)
				return nil
			}
			fmt.Println(val)
			return nil
		}

		// Try local then global
		store, err := storage.OpenRepo(cwd)
		if err == nil {
			defer store.Close()
			val, err := store.GetConfig(key)
			if err == nil && val != "" {
				fmt.Println(val)
				return nil
			}
		}

		gCfg, _ := storage.LoadGlobalConfig()
		if val, ok := gCfg[key]; ok && val != "" {
			fmt.Println(val)
			return nil
		}

		fmt.Printf("config '%s' is not set\n", key)
		return nil
	},
}

func init() {
	configCmd.Flags().BoolVarP(&configGlobal, "global", "g", false, "Read or write global configuration in ~/.kanaconfig")
	configCmd.Flags().BoolVarP(&configList, "list", "l", false, "List all active configuration values")
	configCmd.Flags().BoolVarP(&configUnset, "unset", "u", false, "Remove a configuration setting")
}
