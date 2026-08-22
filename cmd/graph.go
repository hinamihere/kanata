package cmd

import (
	"fmt"
	"os"
	"strings"

	"kana/storage"

	"github.com/spf13/cobra"
)

var graphAll bool

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Render an ASCII visual DAG of semantic snapshot streams",
	Long: `Displays a visual ASCII stream graph in the terminal showing branch points, merges, stream HEADs, and snapshot history.

Examples:
  kana graph          # Render visual stream graph for current stream
  kana graph --all    # Render visual stream graph across all streams`,
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

		return renderStreamGraph(store, graphAll)
	},
}

func renderStreamGraph(store *storage.Storage, allStreams bool) error {
	currentStream, err := store.GetCurrentStream()
	if err != nil {
		return err
	}

	streamHeads := make(map[string]string)
	streams, err := store.ListStreams()
	if err == nil {
		for _, s := range streams {
			h, _ := store.GetStreamHead(s)
			if h != "" {
				streamHeads[h] = s
			}
		}
	}

	activeHead, _ := store.GetStreamHead(currentStream)

	var snapshots []*storage.Snapshot
	if allStreams {
		snapshots, err = store.GetAllSnapshots()
	} else {
		snapshots, err = store.ListSnapshots(currentStream, 50)
	}
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		fmt.Printf("no snapshots recorded in stream '%s'\n", currentStream)
		return nil
	}

	fmt.Printf("\n")

	// Track active columns in the DAG
	columns := []string{}

	for _, s := range snapshots {
		shortHash := s.Hash
		if len(shortHash) > 8 {
			shortHash = shortHash[:8]
		}

		// Build decorations
		var labels []string
		if s.Hash == activeHead {
			labels = append(labels, fmt.Sprintf("head -> %s", currentStream))
		} else if streamName, ok := streamHeads[s.Hash]; ok {
			labels = append(labels, streamName)
		}

		labelStr := ""
		if len(labels) > 0 {
			labelStr = fmt.Sprintf(" (%s)", strings.Join(labels, ", "))
		}

		// Column / Track layout
		colIdx := -1
		for i, h := range columns {
			if h == s.Hash {
				colIdx = i
				break
			}
		}

		if colIdx == -1 {
			// New branch track
			columns = append(columns, s.ParentHash)
			colIdx = len(columns) - 1
		} else {
			// Continue track with parent
			if s.ParentHash != "" {
				columns[colIdx] = s.ParentHash
			} else {
				// End of lineage
				columns = append(columns[:colIdx], columns[colIdx+1:]...)
			}
		}

		// Construct graph prefix
		var prefix strings.Builder
		for i := 0; i < len(columns); i++ {
			if i == colIdx {
				prefix.WriteString("*  ")
			} else {
				prefix.WriteString("|  ")
			}
		}
		if len(columns) == 0 {
			prefix.WriteString("*  ")
		}

		// Print node line
		fmt.Printf("%s%s%s %s\n", prefix.String(), shortHash, labelStr, s.Intent)

		// Print continuity line
		if s.ParentHash != "" && len(columns) > 0 {
			var pipePrefix strings.Builder
			for i := 0; i < len(columns); i++ {
				pipePrefix.WriteString("|  ")
			}
			fmt.Printf("%s\n", pipePrefix.String())
		}
	}

	fmt.Printf("\n")
	return nil
}

func init() {
	graphCmd.Flags().BoolVarP(&graphAll, "all", "a", true, "Render graph across all semantic streams")
}
