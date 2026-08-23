package cmd

import (
	"fmt"
	"os"
	"strings"

	"kana/storage"

	"github.com/spf13/cobra"
)

var findType string
var findStream string
var findSnapshot string
var findContent bool
var findLimit int
var findHistory bool

var findCmd = &cobra.Command{
	Use:     "find <query>",
	Aliases: []string{"grep"},
	Short:   "Search semantic AST symbols (functions, types, macros) across history",
	Long: `Performs structured AST symbol search across current and historical snapshots.

Examples:
  kana find ParseSource                   # Search for functions/types named ParseSource
  kana find -t func Start                 # Find functions matching Start
  kana find -t type Storage               # Find structs/types matching Storage
  kana find -c "http.ListenAndServe"      # Search inside AST function bodies
  kana find ParseSource -H                # Show evolution history of matching symbols
  kana find -s all Start                  # Search across all streams`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

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

		targetStream := findStream
		if targetStream == "" && findSnapshot == "" {
			targetStream = currentStream
		} else if targetStream == "all" {
			targetStream = ""
		}

		// Normalize type shorthand (e.g. "func" -> "function", "struct" -> "type")
		nType := strings.ToLower(findType)
		if nType == "func" {
			nType = "function"
		} else if nType == "struct" || nType == "interface" {
			nType = "type"
		}

		results, err := store.SearchSymbols(query, findSnapshot, nType, targetStream, findContent, findLimit)
		if err != nil {
			return fmt.Errorf("symbol search failed: %w", err)
		}

		if len(results) == 0 {
			filterDesc := ""
			if targetStream != "" {
				filterDesc += fmt.Sprintf(" on stream '%s'", targetStream)
			}
			if nType != "" {
				filterDesc += fmt.Sprintf(" with type '%s'", nType)
			}
			fmt.Printf("no symbols found matching '%s'%s\n", query, filterDesc)
			return nil
		}

		streamDesc := targetStream
		if streamDesc == "" {
			streamDesc = "all streams"
		}
		fmt.Printf("found %d semantic symbol(s) matching '%s' (%s):\n\n", len(results), query, streamDesc)

		seen := make(map[string]bool)
		for _, r := range results {
			key := fmt.Sprintf("%s:%s", r.FilePath, r.NodeID)
			if seen[key] {
				continue
			}
			seen[key] = true

			shortHash := r.SnapshotHash
			if len(shortHash) > 8 {
				shortHash = shortHash[:8]
			}

			lines := ""
			if r.StartLine > 0 {
				lines = fmt.Sprintf(":%d-%d", r.StartLine, r.EndLine)
			}

			fmt.Printf("  • %s (%s%s)\n", r.NodeID, r.FilePath, lines)
			fmt.Printf("    signature: %s\n", r.Signature)
			fmt.Printf("    language:  %s\n", r.Language)
			fmt.Printf("    snapshot:  %s (%s: %q)\n\n", shortHash, r.Author, r.Intent)

			if findHistory {
				history, err := store.GetNodeHistory(r.FilePath, r.NodeID, 10)
				if err == nil && len(history) > 1 {
					fmt.Printf("    evolution history (%d versions):\n", len(history))
					for _, h := range history {
						hShort := h.SnapshotHash
						if len(hShort) > 8 {
							hShort = hShort[:8]
						}
						fmt.Printf("      - [%s] %s by %s: %q\n", hShort, h.Timestamp.Format("2006-01-02"), h.Author, h.Intent)
					}
					fmt.Println()
				}
			}
		}

		return nil
	},
}

func init() {
	findCmd.Flags().StringVarP(&findType, "type", "t", "", "Filter by node type ('func', 'type', 'macro', 'var', 'const')")
	findCmd.Flags().StringVarP(&findStream, "stream", "s", "", "Filter by stream ('all' for all streams)")
	findCmd.Flags().StringVar(&findSnapshot, "snapshot", "", "Search within a specific snapshot hash")
	findCmd.Flags().BoolVarP(&findContent, "content", "c", false, "Search inside AST function and struct body content")
	findCmd.Flags().IntVarP(&findLimit, "limit", "n", 30, "Maximum number of symbols to return")
	findCmd.Flags().BoolVarP(&findHistory, "history", "H", false, "Display the evolution history of each discovered symbol")
}
