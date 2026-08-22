package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"kana/core"
	"kana/storage"

	"github.com/spf13/cobra"
)

var servePort int
var serveHost string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the local Kanata Web Dashboard",
	Long: `Starts a high-density, minimalist web dashboard in your browser to inspect snapshots, AST architectures, semantic diffs, streams, and function evolution graphs.

Examples:
  kana serve
  kana serve --port 8080
  kana serve --host 0.0.0.0 --port 3000`,
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

		addr := fmt.Sprintf("%s:%d", serveHost, servePort)
		fmt.Printf("\n  🚀 Kanata Web Dashboard running at: http://localhost:%d\n", servePort)
		fmt.Println("  Press Ctrl+C to stop the server")

		mux := http.NewServeMux()

		// API Routes
		mux.HandleFunc("/api/info", handleRepoInfo(store))
		mux.HandleFunc("/api/streams", handleStreams(store))
		mux.HandleFunc("/api/snapshots", handleSnapshots(store))
		mux.HandleFunc("/api/tree", handleTree(store))
		mux.HandleFunc("/api/diff", handleDiff(store))
		mux.HandleFunc("/api/blame", handleBlame(store))
		mux.HandleFunc("/api/node-history", handleNodeHistory(store))

		// Web Dashboard Single-Page UI
		mux.HandleFunc("/", handleDashboardHTML(store))

		return http.ListenAndServe(addr, mux)
	},
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 3000, "Port to listen on")
	serveCmd.Flags().StringVar(&serveHost, "host", "0.0.0.0", "Host interface to bind")
}

// -----------------------------------------------------------------------------
// API Handlers
// -----------------------------------------------------------------------------

func handleRepoInfo(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream, _ := store.GetCurrentStream()
		headHash, _ := store.GetStreamHead(stream)
		headSnap, _ := store.GetSnapshot(headHash)

		headAST, _ := store.GetSnapshotAST(headHash)
		wsAST, _ := core.ScanWorkspace(store.RepoPath)
		wsDiff := core.DiffWorkspace(headAST, wsAST)

		resp := map[string]interface{}{
			"repo_name":     filepath.Base(store.RepoPath),
			"repo_path":     store.RepoPath,
			"active_stream": stream,
			"head_hash":     headHash,
			"head_snapshot": headSnap,
			"pending_diff": map[string]interface{}{
				"added":    wsDiff.AddedNodesCount,
				"modified": wsDiff.ModifiedNodesCount,
				"removed":  wsDiff.RemovedNodesCount,
				"files":    len(wsDiff.Files),
			},
		}
		writeJSON(w, resp)
	}
}

func handleStreams(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		streams, err := store.ListStreams()
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		type streamInfo struct {
			Name string `json:"name"`
			Head string `json:"head"`
		}
		var result []streamInfo
		for _, s := range streams {
			head, _ := store.GetStreamHead(s)
			result = append(result, streamInfo{Name: s, Head: head})
		}
		writeJSON(w, result)
	}
}

func handleSnapshots(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stream := r.URL.Query().Get("stream")
		limitStr := r.URL.Query().Get("limit")
		limit := 50
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}

		snaps, err := store.ListSnapshots(stream, limit)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, snaps)
	}
}

func handleTree(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapHash := r.URL.Query().Get("snapshot")
		if snapHash == "" {
			stream, _ := store.GetCurrentStream()
			snapHash, _ = store.GetStreamHead(stream)
		}

		astMap, err := store.GetSnapshotAST(snapHash)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, astMap)
	}
}

func handleDiff(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")

		var oldAST, newAST map[string]*core.FileAST
		var err error

		if from == "workspace" {
			oldAST, _ = core.ScanWorkspace(store.RepoPath)
		} else if from != "" {
			oldAST, err = store.GetSnapshotAST(from)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		if to == "workspace" || to == "" {
			newAST, _ = core.ScanWorkspace(store.RepoPath)
		} else {
			newAST, err = store.GetSnapshotAST(to)
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}

		diff := core.DiffWorkspace(oldAST, newAST)
		patch := core.FormatWorkspacePatch(diff)

		resp := map[string]interface{}{
			"diff":  diff,
			"patch": patch,
		}
		writeJSON(w, resp)
	}
}

func handleBlame(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		stream := r.URL.Query().Get("stream")
		if stream == "" {
			stream, _ = store.GetCurrentStream()
		}

		blames, err := store.GetFileBlame(file, stream)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, blames)
	}
}

func handleNodeHistory(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file := r.URL.Query().Get("file")
		node := r.URL.Query().Get("node")

		history, err := store.GetNodeHistory(file, node, 20)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, history)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func handleDashboardHTML(store *storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		repoName := filepath.Base(store.RepoPath)
		html := getDashboardHTML(repoName)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}
}
