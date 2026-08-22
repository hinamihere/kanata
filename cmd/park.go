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

var parkCmd = &cobra.Command{
	Use:   "park",
	Short: "Temporarily park current AST changes out of the workspace",
	Long:  "Preserves in-flight semantic AST transformations without creating a formal snapshot, allowing seamless context switching.",
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

		if parkList {
			list, err := store.ListParked(stream)
			if err != nil {
				return fmt.Errorf("failed to list parked states: %w", err)
			}
			if len(list) == 0 {
				fmt.Printf("no parked states on stream '%s'\n", stream)
				return nil
			}
			fmt.Printf("parked states on stream '%s':\n", stream)
			for _, p := range list {
				fmt.Printf("  id: %s  note: %q  time: %s\n", p.ID, p.Note, p.Timestamp.Format("2006-01-02 15:04:05"))
			}
			return nil
		}

		if parkRestore {
			ps, fileMap, err := store.PopParked(parkID)
			if err != nil {
				return fmt.Errorf("failed to restore parked state: %w", err)
			}

			if err := core.MaterializeWorkspace(store.RepoPath, fileMap); err != nil {
				return fmt.Errorf("failed to materialize files: %w", err)
			}

			fmt.Printf("restored parked state %s (note: %q)\n", ps.ID, ps.Note)
			return nil
		}

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
		fmt.Println("  workspace reset to clean head (restore with: kana park --restore)")
		return nil
	},
}

func init() {
	parkCmd.Flags().StringVarP(&parkNote, "note", "n", "", "Descriptive note for parked AST state")
	parkCmd.Flags().BoolVarP(&parkRestore, "restore", "r", false, "Restore latest or specified parked state")
	parkCmd.Flags().BoolVarP(&parkList, "list", "l", false, "List all parked states")
	parkCmd.Flags().StringVar(&parkID, "id", "", "Specific parked state ID to restore")
}
