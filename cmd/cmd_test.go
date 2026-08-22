package cmd

import (
	"os"
	"path/filepath"
	"testing"
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
}
