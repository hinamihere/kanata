package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoreEngine_Rules(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-ignore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ignoreContent := `
# Build artifacts
build/
*.obj
*.exe
/dist/
target/

# Logs
*.log
!important.log

# Temp folders
node_modules/
`
	_ = os.WriteFile(filepath.Join(tempDir, ".kanaignore"), []byte(ignoreContent), 0644)

	ignorer := LoadIgnoreRules(tempDir)

	tests := []struct {
		path     string
		isDir    bool
		expected bool
	}{
		{".git", true, true},
		{".kana", true, true},
		{"build", true, true},
		{"build/output.o", false, true},
		{"src/main.obj", false, true},
		{"src/game.exe", false, true},
		{"dist", true, true},
		{"src/dist", true, false}, // /dist/ is root only
		{"target", true, true},
		{"node_modules", true, true},
		{"node_modules/react/index.js", false, true},
		{"server.log", false, true},
		{"important.log", false, false}, // Negation rule
		{"src/main.go", false, false},
		{"README.md", false, false},
	}

	for _, tc := range tests {
		got := ignorer.IsIgnored(tc.path, tc.isDir)
		if got != tc.expected {
			t.Errorf("IsIgnored(%q, isDir=%v) = %v; expected %v", tc.path, tc.isDir, got, tc.expected)
		}
	}
}

func TestBinaryAsset_ParseAndMaterialize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kana-binary-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create binary payload with null bytes (e.g. PNG / ROM header)
	binaryData := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0xFF, 0x00, 0x55}
	binaryPath := filepath.Join(tempDir, "assets", "logo.png")
	_ = os.MkdirAll(filepath.Dir(binaryPath), 0755)
	_ = os.WriteFile(binaryPath, binaryData, 0644)

	// Scan workspace
	wsAST, err := ScanWorkspace(tempDir)
	if err != nil {
		t.Fatalf("ScanWorkspace failed: %v", err)
	}

	fAST, ok := wsAST["assets/logo.png"]
	if !ok {
		t.Fatalf("expected assets/logo.png to be scanned")
	}

	if fAST.Language != "binary" {
		t.Errorf("expected language 'binary', got: %s", fAST.Language)
	}

	if _, ok := fAST.Nodes["blob:raw"]; !ok {
		t.Fatalf("expected blob:raw node in binary FileAST")
	}

	// Materialize to another directory and verify exact byte match
	destDir, _ := os.MkdirTemp("", "kana-binary-dest-*")
	defer os.RemoveAll(destDir)

	if err := MaterializeWorkspace(destDir, wsAST); err != nil {
		t.Fatalf("MaterializeWorkspace failed: %v", err)
	}

	readBack, err := os.ReadFile(filepath.Join(destDir, "assets", "logo.png"))
	if err != nil {
		t.Fatalf("failed to read materialized binary file: %v", err)
	}

	if string(readBack) != string(binaryData) {
		t.Errorf("materialized binary data mismatch: got %v, expected %v", readBack, binaryData)
	}
}
