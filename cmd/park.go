package cmd

import (
	"fmt"
	"os"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var parkNote string
var parkRestore bool
var parkList bool
var parkID string
var parkShow bool
var parkDrop bool

var parkCmd = &cobra.Command{
	Use:   "park [save|list|restore|show|drop] [id-or-note]",
	Short: "Temporarily park current AST changes out of the workspace",
	Long: `Preserves in-flight semantic AST transformations without creating a formal snapshot, allowing seamless context switching.

Subcommands / Flags:
  kana park                         - Park current pending AST modifications
  kana park -n "wip-feature"         - Park with a custom shelf note/name
  kana park list                    - List all parked states
  kana park show [id]               - Preview AST modifications stored in a parked state
  kana park restore [id]            - Restore parked AST state into workspace
  kana park drop [id]               - Delete a parked state without restoring`,
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

		// Check subcommands or flags
		action := ""
		targetID := parkID
		if len(args) > 0 {
			switch args[0] {
			case "list", "ls":
				action = "list"
			case "restore", "pop":
				action = "restore"
				if len(args) > 1 {
					targetID = args[1]
				}
			case "show", "preview":
				action = "show"
				if len(args) > 1 {
					targetID = args[1]
				}
			case "drop", "delete", "rm":
				action = "drop"
				if len(args) > 1 {
					targetID = args[1]
				}
			case "save":
				action = "save"
				if len(args) > 1 && parkNote == "" {
					parkNote = args[1]
				}
			default:
				targetID = args[0]
			}
		}

		if parkList || action == "list" {
			return runParkList(store, stream)
		}

		if parkShow || action == "show" {
			return runParkShow(store, targetID)
		}

		if parkDrop || action == "drop" {
			if targetID == "" {
				return fmt.Errorf("please specify the parked state id or note to drop")
			}
			if err := store.DropParked(targetID); err != nil {
				return err
			}
			fmt.Printf("dropped parked state: %s\n", targetID)
			return nil
		}

		if parkRestore || action == "restore" {
			ps, fileMap, err := store.PopParked(targetID)
			if err != nil {
				return fmt.Errorf("failed to restore parked state: %w", err)
			}

			if err := core.MaterializeWorkspace(store.RepoPath, fileMap); err != nil {
				return fmt.Errorf("failed to materialize files: %w", err)
			}

			fmt.Printf("restored parked state %s (note: %q)\n", ps.ID, ps.Note)
			return nil
		}

		// Default: Save / Park workspace
		currentAST, err := core.ScanWorkspace(store.RepoPath)
		if err != nil {
			return fmt.Errorf("workspace scan failed: %w", err)
		}

		headHash, _ := store.GetStreamHead(stream)
		headAST, _ := store.GetSnapshotAST(headHash)

		diff := core.DiffWorkspace(headAST, currentAST)
		if len(diff.Files) == 0 {
			fmt.Println("nothing to park (no pending AST changes)")
			return nil
		}

		note := parkNote
		if note == "" {
			note = fmt.Sprintf("parked %d modified AST files", len(diff.Files))
		}

		id, err := store.ParkWorkspace(stream, note, currentAST)
		if err != nil {
			return fmt.Errorf("failed to park AST state: %w", err)
		}

		if headHash != "" {
			if err := core.MaterializeWorkspace(store.RepoPath, headAST); err != nil {
				return fmt.Errorf("failed to revert workspace to head: %w", err)
			}
		}

		fmt.Printf("parked %d file(s) with id: %s\n", len(diff.Files), id)
		fmt.Printf("  note: %q\n", note)
		fmt.Printf("  workspace reset to clean head (restore with: kana park --restore)\n")
		return nil
	},
}

func runParkList(store *storage.Storage, stream string) error {
	list, err := store.ListParked("")
	if err != nil {
		return fmt.Errorf("failed to list parked states: %w", err)
	}
	if len(list) == 0 {
		fmt.Println("no parked states")
		return nil
	}
	fmt.Println("parked shelves:")
	for _, p := range list {
		marker := ""
		if p.WorkStream == stream {
			marker = " (active stream)"
		}
		fmt.Printf("  [%s] %q (stream: %s%s, %s)\n",
			p.ID, p.Note, p.WorkStream, marker, p.Timestamp.Format("2006-01-02 15:04"))
	}
	return nil
}

func runParkShow(store *storage.Storage, id string) error {
	ps, parkedAST, err := store.GetParked(id)
	if err != nil {
		return err
	}

	headHash, _ := store.GetStreamHead(ps.WorkStream)
	headAST, _ := store.GetSnapshotAST(headHash)

	diff := core.DiffWorkspace(headAST, parkedAST)

	fmt.Printf("parked shelf [%s] note: %q (stream: %s, %s)\n\n",
		ps.ID, ps.Note, ps.WorkStream, ps.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Println(core.FormatWorkspaceDiff(diff))
	return nil
}

func init() {
	parkCmd.Flags().StringVarP(&parkNote, "note", "n", "", "Custom note or shelf name for the parked state")
	parkCmd.Flags().BoolVarP(&parkRestore, "restore", "r", false, "Restore latest or specified parked state")
	parkCmd.Flags().BoolVarP(&parkList, "list", "l", false, "List all parked states")
	parkCmd.Flags().BoolVarP(&parkShow, "show", "s", false, "Preview AST differences in parked state")
	parkCmd.Flags().BoolVarP(&parkDrop, "drop", "d", false, "Drop/delete parked state without restoring")
	parkCmd.Flags().StringVar(&parkID, "id", "", "Specific parked state ID")
}
