package core

import (
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ScanWorkspace traverses the repository workspace, filtering via .kanaignore and .gitignore,
// producing semantic ASTs for code and raw blob nodes for binary assets.
func ScanWorkspace(repoRoot string) (map[string]*FileAST, error) {
	fileMap := make(map[string]*FileAST)
	ignorer := LoadIgnoreRules(repoRoot)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == repoRoot {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Check .kanaignore / .gitignore rules
		if ignorer.IsIgnored(relPath, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		fileAST, err := ParseSource(relPath, content)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", relPath, err)
		}

		fileMap[relPath] = fileAST
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("workspace scan failed: %w", err)
	}

	return fileMap, nil
}

// MaterializeWorkspace writes reconstructed file ASTs and binary blobs back to disk.
func MaterializeWorkspace(repoRoot string, fileMap map[string]*FileAST) error {
	for relPath, fAST := range fileMap {
		fullPath := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		// Check for binary asset blob
		if fAST.Language == "binary" {
			if rawNode, ok := fAST.Nodes["blob:raw"]; ok {
				decoded, err := base64.StdEncoding.DecodeString(rawNode.Content)
				if err == nil {
					if err := os.WriteFile(fullPath, decoded, 0644); err != nil {
						return err
					}
					continue
				}
			}
		}

		// Text/Code AST Reconstruction
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
