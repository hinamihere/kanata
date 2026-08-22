package cmd

import (
	"fmt"
	"os"

	"kana/storage"

	"github.com/spf13/cobra"
)

var focusList bool
var focusResume bool

var focusCmd = &cobra.Command{
	Use:   "focus [stream-name]",
	Short: "Switch or create semantic work streams",
	Long:  "Switches the active work stream context or creates a new semantic branch from current state.",
	Args:  cobra.MaximumNArgs(1),
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

		currentStream, err := store.GetCurrentStream()
		if err != nil {
			return err
		}

		if focusList || len(args) == 0 {
			streams, err := store.ListStreams()
			if err != nil {
				return fmt.Errorf("failed to list streams: %w", err)
			}

			fmt.Println("work streams:")
			for _, s := range streams {
				head, _ := store.GetStreamHead(s)
				if len(head) > 10 {
					head = head[:10]
				}
				marker := "  "
				if s == currentStream {
					marker = "* "
				}
				if head != "" {
					fmt.Printf("  %s%-18s (head: %s)\n", marker, s, head)
				} else {
					fmt.Printf("  %s%-18s (empty)\n", marker, s)
				}
			}
			return nil
		}

		targetStream := args[0]
		if targetStream == currentStream {
			fmt.Printf("already on stream '%s'\n", targetStream)
			return nil
		}

		currentHead, _ := store.GetStreamHead(currentStream)

		if err := store.CreateStream(targetStream, currentHead); err != nil {
			return fmt.Errorf("failed to create stream %s: %w", targetStream, err)
		}

		if err := store.SetCurrentStream(targetStream); err != nil {
			return fmt.Errorf("failed to switch focus to %s: %w", targetStream, err)
		}

		fmt.Printf("focused on stream: %s\n", targetStream)
		return nil
	},
}

func init() {
	focusCmd.Flags().BoolVarP(&focusList, "list", "l", false, "List all active work streams")
	focusCmd.Flags().BoolVarP(&focusResume, "resume", "r", false, "Resume previous focus context")
}
