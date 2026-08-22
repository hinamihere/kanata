package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var ignoredDirs = map[string]bool{
	".git":         true,
	".kana":        true,
	".idea":        true,
	".vscode":      true,
	"node_modules": true,
	"vendor":       true,
	"bin":          true,
	"dist":         true,
}

var ignoredExtensions = map[string]bool{
	".exe":   true,
	".dll":   true,
	".so":    true,
	".dylib": true,
	".db":    true,
	".db-journal": true,
	".db-wal": true,
	".tar":   true,
	".gz":    true,
	".zip":   true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".ico":   true,
}

// ScanWorkspace traverses the repository workspace and produces an AST representation of all code files.
func ScanWorkspace(repoRoot string) (map[string]*FileAST, error) {
	fileMap := make(map[string]*FileAST)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		if d.IsDir() {
			if ignoredDirs[name] || strings.HasPrefix(name, ".") && name != "." {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(name))
		if ignoredExtensions[ext] {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Parse the file
		fAST, err := ParseFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", relPath, err)
		}
		fAST.FilePath = relPath
		fileMap[relPath] = fAST

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("workspace scan failed: %w", err)
	}

	return fileMap, nil
}

// MaterializeWorkspace writes reconstructed file ASTs to disk.
func MaterializeWorkspace(repoRoot string, fileMap map[string]*FileAST) error {
	for relPath, fAST := range fileMap {
		fullPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		var sb strings.Builder
		for _, node := range fAST.SortedNodes() {
			if node.DocComment != "" {
				sb.WriteString(node.DocComment + "\n")
			}
			sb.WriteString(node.Content + "\n\n")
		}

		if err := os.WriteFile(fullPath, []byte(strings.TrimRight(sb.String(), "\n")+"\n"), 0644); err != nil {
			return err
		}
	}
	return nil
}
