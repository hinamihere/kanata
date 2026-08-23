package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kana/storage"
)

func TestCLI_EndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-cli-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current wd: %v", err)
	}
	defer os.Chdir(origWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	// 1. Test init
	rootCmd.SetArgs([]string{"init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana init failed: %v", err)
	}

	// 2. Create sample source files
	codeV1 := `package core

type Engine struct {
	ID string
}

func Start() bool {
	return true
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "engine.go"), []byte(codeV1), 0644); err != nil {
		t.Fatalf("failed to write engine.go: %v", err)
	}

	codeUtils := `package core

func UtilityHelper() string {
	return "ok"
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "utils.go"), []byte(codeUtils), 0644); err != nil {
		t.Fatalf("failed to write utils.go: %v", err)
	}

	// 3. Test status
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana status failed: %v", err)
	}

	// 4. Test selective snapshot (only snapshot engine.go first)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Initial engine bootstrap", "engine.go"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana snapshot selective failed: %v", err)
	}

	// Snapshot utils.go
	rootCmd.SetArgs([]string{"snapshot", "-i", "Add utility helpers", "utils.go"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana snapshot selective utils failed: %v", err)
	}

	// 5. Test focus (create new stream)
	rootCmd.SetArgs([]string{"focus", "feature-async"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana focus failed: %v", err)
	}

	// 6. Test park
	codeV2 := `package core

type Engine struct {
	ID string
}

func Start() bool {
	return true
}

func Stop() bool {
	return false
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "engine.go"), []byte(codeV2), 0644); err != nil {
		t.Fatalf("failed to update engine.go: %v", err)
	}

	rootCmd.SetArgs([]string{"park", "-n", "Parked in-progress Stop func"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park failed: %v", err)
	}

	// Test park --list
	rootCmd.SetArgs([]string{"park", "--list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park --list failed: %v", err)
	}

	// Test park --restore
	rootCmd.SetArgs([]string{"park", "--restore"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park --restore failed: %v", err)
	}

	// 7. Snapshot on feature-async
	rootCmd.SetArgs([]string{"snapshot", "-i", "Add Stop function to engine"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana snapshot on stream failed: %v", err)
	}

	// 8. Focus back to main and integrate feature-async
	rootCmd.SetArgs([]string{"focus", "main"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana focus main failed: %v", err)
	}

	rootCmd.SetArgs([]string{"integrate", "feature-async"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana integrate failed: %v", err)
	}

	// 9. Test log
	rootCmd.SetArgs([]string{"log"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana log failed: %v", err)
	}

	rootCmd.SetArgs([]string{"log", "--stream", "all"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana log --stream all failed: %v", err)
	}

	// 10. Test diff
	rootCmd.SetArgs([]string{"diff"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana diff failed: %v", err)
	}

	rootCmd.SetArgs([]string{"diff", "-p"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana diff -p failed: %v", err)
	}

	// 11. Test blame
	rootCmd.SetArgs([]string{"blame", "engine.go"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana blame failed: %v", err)
	}

	rootCmd.SetArgs([]string{"blame", "engine.go", "-f", "Start"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana blame -f failed: %v", err)
	}

	// 12. Test remote management
	rootCmd.SetArgs([]string{"remote", "add", "backup", "/tmp/backup_repo"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana remote add failed: %v", err)
	}

	rootCmd.SetArgs([]string{"remote"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana remote list failed: %v", err)
	}

	rootCmd.SetArgs([]string{"remote", "remove", "backup"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana remote remove failed: %v", err)
	}

	// 13. Test graph
	rootCmd.SetArgs([]string{"graph"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana graph failed: %v", err)
	}

	rootCmd.SetArgs([]string{"log", "-g"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana log -g failed: %v", err)
	}
}

func TestCLI_ClonePushPull_Local(t *testing.T) {
	primaryDir, err := os.MkdirTemp("", "kana-primary-*")
	if err != nil {
		t.Fatalf("failed to create primary temp dir: %v", err)
	}
	defer os.RemoveAll(primaryDir)

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	// 1. Initialize primary repo
	_ = os.Chdir(primaryDir)
	rootCmd.SetArgs([]string{"init"})
	_ = rootCmd.Execute()

	_ = os.WriteFile(filepath.Join(primaryDir, "main.go"), []byte("package main\nfunc App() {}\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Initial app commit"})
	_ = rootCmd.Execute()

	// 2. Clone primary into secondary repo
	cloneDir, err := os.MkdirTemp("", "kana-cloned-*")
	if err != nil {
		t.Fatalf("failed to create clone temp dir: %v", err)
	}
	defer os.RemoveAll(cloneDir)

	clonedDest := filepath.Join(cloneDir, "my_clone")
	rootCmd.SetArgs([]string{"clone", primaryDir, clonedDest})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana clone failed: %v", err)
	}

	// Verify cloned file exists
	if _, err := os.Stat(filepath.Join(clonedDest, "main.go")); err != nil {
		t.Fatalf("cloned file main.go does not exist: %v", err)
	}

	// 3. Make new snapshot in primary
	_ = os.Chdir(primaryDir)
	_ = os.WriteFile(filepath.Join(primaryDir, "main.go"), []byte("package main\nfunc App() {}\nfunc Extra() {}\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Add Extra function"})
	_ = rootCmd.Execute()

	// 4. Pull in cloned repo from primary
	_ = os.Chdir(clonedDest)
	rootCmd.SetArgs([]string{"pull", "origin", "main"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana pull failed: %v", err)
	}

	// Verify Extra() exists in cloned repo
	content, _ := os.ReadFile(filepath.Join(clonedDest, "main.go"))
	if !strings.Contains(string(content), "Extra") {
		t.Errorf("expected pulled changes with 'Extra', got: %s", string(content))
	}

	// 5. Test suggest and auto snapshot
	_ = os.WriteFile(filepath.Join(clonedDest, "main.go"), []byte("package main\nfunc App() {}\nfunc Extra() {}\nfunc AutoInferred() {}\n"), 0644)
	rootCmd.SetArgs([]string{"suggest"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana suggest failed: %v", err)
	}

	rootCmd.SetArgs([]string{"snapshot", "-a"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana snapshot -a failed: %v", err)
	}

	// 6. Test cherry-pick
	rootCmd.SetArgs([]string{"focus", "feature-pick"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana focus failed: %v", err)
	}

	_ = os.WriteFile(filepath.Join(clonedDest, "main.go"), []byte("package main\nfunc App() {}\nfunc Extra() {}\nfunc AutoInferred() {}\nfunc TransplantedHelper() string { return \"picked\" }\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Add TransplantedHelper on feature branch"})
	_ = rootCmd.Execute()

	rootCmd.SetArgs([]string{"focus", "main"})
	_ = rootCmd.Execute()

	// Verify main currently doesn't have TransplantedHelper
	mContent, _ := os.ReadFile(filepath.Join(clonedDest, "main.go"))
	if strings.Contains(string(mContent), "TransplantedHelper") {
		t.Fatalf("expected main to not have TransplantedHelper yet")
	}

	// Pick only TransplantedHelper from feature-pick head
	rootCmd.SetArgs([]string{"pick", "feature-pick", "-f", "TransplantedHelper"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana pick failed: %v", err)
	}

	// Verify main now has TransplantedHelper
	mContentAfter, _ := os.ReadFile(filepath.Join(clonedDest, "main.go"))
	if !strings.Contains(string(mContentAfter), "TransplantedHelper") {
		t.Fatalf("expected main to contain TransplantedHelper after pick, got: %s", string(mContentAfter))
	}

	// 7. Test stream compare
	rootCmd.SetArgs([]string{"stream", "compare", "feature-pick", "main"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana stream compare failed: %v", err)
	}

	rootCmd.SetArgs([]string{"stream", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana stream list failed: %v", err)
	}

	// 8. Test named park shelf
	_ = os.WriteFile(filepath.Join(clonedDest, "main.go"), []byte("package main\nfunc App() {}\nfunc Extra() {}\nfunc AutoInferred() {}\nfunc TransplantedHelper() string { return \"picked\" }\nfunc ParkedFunc() {}\n"), 0644)
	rootCmd.SetArgs([]string{"park", "-n", "wip-shelf"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park -n failed: %v", err)
	}

	rootCmd.SetArgs([]string{"park", "list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park list failed: %v", err)
	}

	rootCmd.SetArgs([]string{"park", "show", "wip-shelf"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park show failed: %v", err)
	}

	rootCmd.SetArgs([]string{"park", "restore", "wip-shelf"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana park restore failed: %v", err)
	}

	// 9. Test find
	rootCmd.SetArgs([]string{"find", "TransplantedHelper"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana find failed: %v", err)
	}

	rootCmd.SetArgs([]string{"find", "-t", "func", "Extra"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana find -t func failed: %v", err)
	}
}

func TestCLI_HTTP_ClonePushPull(t *testing.T) {
	primaryDir, err := os.MkdirTemp("", "kana-http-primary-*")
	if err != nil {
		t.Fatalf("failed to create primary temp dir: %v", err)
	}
	defer os.RemoveAll(primaryDir)

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	// 1. Initialize primary repo
	_ = os.Chdir(primaryDir)
	rootCmd.SetArgs([]string{"init"})
	_ = rootCmd.Execute()

	_ = os.WriteFile(filepath.Join(primaryDir, "server.go"), []byte("package main\nfunc StartServer() {}\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Initial server commit"})
	_ = rootCmd.Execute()

	store, err := storage.OpenRepo(primaryDir)
	if err != nil {
		t.Fatalf("failed to open primary storage: %v", err)
	}
	defer store.Close()

	// 2. Start mock HTTP server exposing transport endpoints
	mux := http.NewServeMux()
	mux.HandleFunc("/api/transport/head", handleTransportHead(store))
	mux.HandleFunc("/api/transport/export-bundle", handleTransportExportBundle(store))
	mux.HandleFunc("/api/transport/import-bundle", handleTransportImportBundle(store))

	server := httptest.NewServer(mux)
	defer server.Close()

	// 3. Clone repository over HTTP
	cloneDir, err := os.MkdirTemp("", "kana-http-cloned-*")
	if err != nil {
		t.Fatalf("failed to create clone temp dir: %v", err)
	}
	defer os.RemoveAll(cloneDir)

	clonedDest := filepath.Join(cloneDir, "http_clone")
	rootCmd.SetArgs([]string{"clone", server.URL, clonedDest})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana http clone failed: %v", err)
	}

	// Verify cloned file exists
	if _, err := os.Stat(filepath.Join(clonedDest, "server.go")); err != nil {
		t.Fatalf("cloned file server.go does not exist: %v", err)
	}

	// 4. Make new snapshot in clone and push over HTTP
	_ = os.Chdir(clonedDest)
	_ = os.WriteFile(filepath.Join(clonedDest, "server.go"), []byte("package main\nfunc StartServer() {}\nfunc StopServer() {}\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Add StopServer over HTTP"})
	_ = rootCmd.Execute()

	rootCmd.SetArgs([]string{"push", "origin", "main"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana http push failed: %v", err)
	}

	// 5. Verify pushed state on server
	newHead, _ := store.GetStreamHead("main")
	snap, _ := store.GetSnapshot(newHead)
	if snap == nil || snap.Intent != "Add StopServer over HTTP" {
		t.Fatalf("expected server to have pushed snapshot, got: %+v", snap)
	}
}

func TestCLI_InteractiveSnapshotStaging(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-stage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	_ = os.Chdir(tempDir)
	rootCmd.SetArgs([]string{"init"})
	_ = rootCmd.Execute()

	_ = os.WriteFile(filepath.Join(tempDir, "app.go"), []byte("package main\nfunc Base() {}\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Initial app base"})
	_ = rootCmd.Execute()

	// Add 2 new functions in app.go
	_ = os.WriteFile(filepath.Join(tempDir, "app.go"), []byte("package main\nfunc Base() {}\nfunc StagedHelper() {}\nfunc UnstagedHelper() {}\n"), 0644)

	// Pipe "y" (stage first new node) and "n" (skip second new node)
	snapshotPromptReader = strings.NewReader("y\nn\n")
	defer func() { snapshotPromptReader = os.Stdin }()

	rootCmd.SetArgs([]string{"snapshot", "-p", "-i", "Selectively stage StagedHelper only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("interactive snapshot failed: %v", err)
	}

	store, err := storage.OpenRepo(tempDir)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()

	headHash, _ := store.GetStreamHead("main")
	headAST, _ := store.GetSnapshotAST(headHash)

	// Verify head snapshot contains StagedHelper
	appAST, ok := headAST["app.go"]
	if !ok {
		t.Fatalf("expected app.go in head snapshot AST")
	}

	if _, ok := appAST.Nodes["func:StagedHelper"]; !ok {
		t.Errorf("expected func:StagedHelper to be staged in snapshot")
	}
	if _, ok := appAST.Nodes["func:UnstagedHelper"]; ok {
		t.Errorf("expected func:UnstagedHelper NOT to be staged in snapshot")
	}

	// Verify workspace on disk still has both
	content, _ := os.ReadFile(filepath.Join(tempDir, "app.go"))
	if !strings.Contains(string(content), "UnstagedHelper") {
		t.Errorf("expected workspace file to retain unstaged changes")
	}
}

func TestCLI_ConfigAndAuthorIdentity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)

	_ = os.Chdir(tempDir)
	rootCmd.SetArgs([]string{"init"})
	_ = rootCmd.Execute()

	// 1. Set local config
	rootCmd.SetArgs([]string{"config", "user.name", "Hoshino Hinami"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana config set user.name failed: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "user.email", "hinami@example.com"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana config set user.email failed: %v", err)
	}

	rootCmd.SetArgs([]string{"config", "--list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana config --list failed: %v", err)
	}

	// 2. Make snapshot and verify author string
	_ = os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc Hello() {}\n"), 0644)
	rootCmd.SetArgs([]string{"snapshot", "-i", "Initial configured commit"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("kana snapshot failed: %v", err)
	}

	store, err := storage.OpenRepo(tempDir)
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer store.Close()

	headHash, _ := store.GetStreamHead("main")
	snap, _ := store.GetSnapshot(headHash)
	if snap == nil {
		t.Fatalf("snapshot is nil")
	}

	expectedAuthor := "Hoshino Hinami <hinami@example.com>"
	if snap.Author != expectedAuthor {
		t.Errorf("expected snapshot author %q, got %q", expectedAuthor, snap.Author)
	}
}
