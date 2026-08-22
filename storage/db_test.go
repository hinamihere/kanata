package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kana/core"
)

func TestStorage_FullLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. Initialize repository
	store, err := InitRepo(tempDir)
	if err != nil {
		t.Fatalf("InitRepo failed: %v", err)
	}
	defer store.Close()

	// Verify default stream
	stream, err := store.GetCurrentStream()
	if err != nil || stream != "main" {
		t.Errorf("expected default stream 'main', got %s (err: %v)", stream, err)
	}

	// 2. Create sample FileAST and record a snapshot
	sampleSrc := []byte(`package service

func ProcessUser(id string) bool {
	return len(id) > 0
}
`)
	fAST, err := core.ParseSource("service.go", sampleSrc)
	if err != nil {
		t.Fatalf("failed to parse sample source: %v", err)
	}
	fileMap := map[string]*core.FileAST{"service.go": fAST}

	treeHash := ComputeTreeHash(fileMap)
	now := time.Now().UTC()
	snapHash := ComputeSnapshotHash("", "main", "test-user", "Initial service setup", now, treeHash)

	snap := &Snapshot{
		Hash:       snapHash,
		ParentHash: "",
		WorkStream: "main",
		Author:     "test-user",
		Intent:     "Initial service setup",
		Timestamp:  now,
		TreeHash:   treeHash,
	}

	if err := store.SaveSnapshot(snap, fileMap); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// 3. Verify Head and query snapshot
	head, err := store.GetStreamHead("main")
	if err != nil || head != snapHash {
		t.Errorf("expected stream head %s, got %s (err: %v)", snapHash, head, err)
	}

	retrievedSnap, err := store.GetSnapshot(snapHash)
	if err != nil || retrievedSnap == nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if retrievedSnap.Intent != "Initial service setup" {
		t.Errorf("unexpected intent: %s", retrievedSnap.Intent)
	}

	retrievedAST, err := store.GetSnapshotAST(snapHash)
	if err != nil {
		t.Fatalf("GetSnapshotAST failed: %v", err)
	}
	if len(retrievedAST) != 1 || retrievedAST["service.go"] == nil {
		t.Fatalf("reconstructed AST missing service.go")
	}
	if _, ok := retrievedAST["service.go"].Nodes["func:ProcessUser"]; !ok {
		t.Errorf("missing func:ProcessUser in reconstructed AST")
	}

	// 4. Test Stream branching
	if err := store.CreateStream("feature-auth", snapHash); err != nil {
		t.Fatalf("CreateStream failed: %v", err)
	}
	streams, err := store.ListStreams()
	if err != nil || len(streams) < 2 {
		t.Errorf("expected at least 2 streams, got %v (err: %v)", streams, err)
	}

	// 5. Test Park & Pop
	parkID, err := store.ParkWorkspace("feature-auth", "wip auth token logic", fileMap)
	if err != nil {
		t.Fatalf("ParkWorkspace failed: %v", err)
	}

	parkedList, err := store.ListParked("feature-auth")
	if err != nil || len(parkedList) != 1 {
		t.Fatalf("expected 1 parked state, got %d", len(parkedList))
	}

	ps, restoredFiles, err := store.PopParked(parkID)
	if err != nil || ps == nil {
		t.Fatalf("PopParked failed: %v", err)
	}
	if len(restoredFiles) != 1 {
		t.Errorf("expected 1 restored file, got %d", len(restoredFiles))
	}
}

func TestStorage_OpenRepoLookup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-lookup-*")
	if err != nil {
		t.Fatalf("temp dir creation failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := InitRepo(tempDir)
	if err != nil {
		t.Fatalf("InitRepo failed: %v", err)
	}
	store.Close()

	// Nested subdirectory
	nested := filepath.Join(tempDir, "sub", "pkg", "deep")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("mkdir deep failed: %v", err)
	}

	opened, err := OpenRepo(nested)
	if err != nil {
		t.Fatalf("OpenRepo failed from nested directory: %v", err)
	}
	defer opened.Close()

	if opened.RepoPath != tempDir {
		t.Errorf("expected repo path %s, got %s", tempDir, opened.RepoPath)
	}
}
