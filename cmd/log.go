package cmd

import (
	"fmt"
	"os"

	"kana/storage"

	"github.com/spf13/cobra"
)

var logLimit int
var logStream string
var logGraph bool

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show semantic snapshot history",
	Long:  "Displays the chronological snapshot graph timeline for the current or specified work stream.",
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

		if logGraph {
			return renderStreamGraph(store, logStream == "all" || logStream == "")
		}

		stream := logStream
		if stream == "" {
			var err error
			stream, err = store.GetCurrentStream()
			if err != nil {
				return err
			}
		}

		headHash, _ := store.GetStreamHead(stream)

		targetStreamFilter := stream
		if logStream == "all" {
			targetStreamFilter = ""
		}

		snapshots, err := store.ListSnapshots(targetStreamFilter, logLimit)
		if err != nil {
			return fmt.Errorf("failed to retrieve snapshot history: %w", err)
		}

		if len(snapshots) == 0 {
			fmt.Printf("no snapshots recorded on stream '%s'\n", stream)
			return nil
		}

		for _, s := range snapshots {
			shortHash := s.Hash
			if len(shortHash) > 12 {
				shortHash = shortHash[:12]
			}

			headMarker := ""
			if s.Hash == headHash {
				headMarker = fmt.Sprintf(" (head -> %s)", s.WorkStream)
			} else {
				headMarker = fmt.Sprintf(" (%s)", s.WorkStream)
			}

			fmt.Printf("snapshot %s%s\n", shortHash, headMarker)
			fmt.Printf("  author:    %s\n", s.Author)
			fmt.Printf("  timestamp: %s\n", s.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("  intent:    %s\n\n", s.Intent)
		}

		return nil
	},
}

func init() {
	logCmd.Flags().IntVarP(&logLimit, "limit", "n", 20, "Number of snapshots to display")
	logCmd.Flags().StringVarP(&logStream, "stream", "s", "", "Filter snapshots by work stream ('all' for all streams)")
	logCmd.Flags().BoolVarP(&logGraph, "graph", "g", false, "Render visual ASCII stream DAG")
}
