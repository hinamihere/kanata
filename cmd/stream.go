package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var streamComparePatch bool

var streamCmd = &cobra.Command{
	Use:   "stream",
	Short: "Inspect and compare semantic work streams",
	Long: `Manages and compares semantic work streams at the architectural AST level.

Subcommands:
  kana stream list                       - List all active work streams
  kana stream compare <streamA> [streamB] - Compare structural AST differences between two streams`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStreamList()
	},
}

var streamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all semantic work streams",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStreamList()
	},
}

var streamCompareCmd = &cobra.Command{
	Use:   "compare <streamA> [streamB]",
	Short: "Compare architectural AST differences between two streams",
	Long: `Performs an architectural AST comparison between two semantic streams, highlighting added, modified, and removed functions, types, and macros.

Examples:
  kana stream compare feature-async main
  kana stream compare feature-async       # Compares against active stream
  kana stream compare feature-async -p    # Includes line-level LCS diffs`,
	Args: cobra.RangeArgs(1, 2),
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

		streamA := args[0]
		streamB := currentStream
		if len(args) == 2 {
			streamB = args[1]
		}

		headA, err := store.GetStreamHead(streamA)
		if err != nil || headA == "" {
			return fmt.Errorf("stream '%s' has no recorded snapshots", streamA)
		}

		headB, err := store.GetStreamHead(streamB)
		if err != nil || headB == "" {
			return fmt.Errorf("stream '%s' has no recorded snapshots", streamB)
		}

		if headA == headB {
			fmt.Printf("streams '%s' and '%s' are identical (pointing to snapshot %s)\n", streamA, streamB, headA[:8])
			return nil
		}

		astA, err := store.GetSnapshotAST(headA)
		if err != nil {
			return fmt.Errorf("failed to load AST for stream '%s': %w", streamA, err)
		}

		astB, err := store.GetSnapshotAST(headB)
		if err != nil {
			return fmt.Errorf("failed to load AST for stream '%s': %w", streamB, err)
		}

		diff := core.DiffWorkspace(astB, astA)

		shortA := headA
		if len(shortA) > 8 {
			shortA = shortA[:8]
		}
		shortB := headB
		if len(shortB) > 8 {
			shortB = shortB[:8]
		}

		fmt.Printf("comparing stream '%s' (%s) vs '%s' (%s):\n\n", streamA, shortA, streamB, shortB)

		if len(diff.Files) == 0 {
			fmt.Println("  no semantic differences detected.")
			return nil
		}

		var addedNodes []string
		var modifiedNodes []string
		var removedNodes []string

		for file, fDiff := range diff.Files {
			for _, nd := range fDiff.NodeDiffs {
				desc := fmt.Sprintf("%s: %s", file, nd.Signature)
				switch nd.ChangeType {
				case core.ChangeAdded:
					addedNodes = append(addedNodes, desc)
				case core.ChangeModified:
					modifiedNodes = append(modifiedNodes, desc)
				case core.ChangeRemoved:
					removedNodes = append(removedNodes, desc)
				}
			}
		}

		if len(addedNodes) > 0 {
			fmt.Printf("+ added in '%s' (%d):\n", streamA, len(addedNodes))
			for _, item := range addedNodes {
				fmt.Printf("  + %s\n", item)
			}
			fmt.Println()
		}

		if len(modifiedNodes) > 0 {
			fmt.Printf("~ modified between streams (%d):\n", len(modifiedNodes))
			for _, item := range modifiedNodes {
				fmt.Printf("  ~ %s\n", item)
			}
			fmt.Println()
		}

		if len(removedNodes) > 0 {
			fmt.Printf("- only in '%s' (%d):\n", streamB, len(removedNodes))
			for _, item := range removedNodes {
				fmt.Printf("  - %s\n", item)
			}
			fmt.Println()
		}

		fmt.Printf("summary: +%d added, ~%d modified, -%d removed\n",
			diff.AddedNodesCount, diff.ModifiedNodesCount, diff.RemovedNodesCount)

		if streamComparePatch {
			fmt.Printf("\n--- semantic line patch ---\n\n")
			fmt.Println(core.FormatWorkspacePatch(diff))
		}

		return nil
	},
}

func runStreamList() error {
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

	streams, err := store.ListStreams()
	if err != nil {
		return fmt.Errorf("failed to list streams: %w", err)
	}

	fmt.Println("work streams:")
	for _, s := range streams {
		head, _ := store.GetStreamHead(s)
		shortHead := head
		if len(shortHead) > 10 {
			shortHead = shortHead[:10]
		}
		marker := "  "
		if s == currentStream {
			marker = "* "
		}
		if shortHead != "" {
			fmt.Printf("  %s%-18s (head: %s)\n", marker, s, shortHead)
		} else {
			fmt.Printf("  %s%-18s (empty)\n", marker, s)
		}
	}

	return nil
}

func init() {
	streamCompareCmd.Flags().BoolVarP(&streamComparePatch, "patch", "p", false, "Display line-level LCS AST diffs")
	streamCmd.AddCommand(streamListCmd)
	streamCmd.AddCommand(streamCompareCmd)
}
