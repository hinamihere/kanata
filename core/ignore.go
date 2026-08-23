package core

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreEngine handles file pattern exclusions from .kanaignore and .gitignore.
type IgnoreEngine struct {
	patterns []ignoreRule
}

type ignoreRule struct {
	pattern    string
	isDirOnly  bool
	isNegation bool
	isRootOnly bool
}

// LoadIgnoreRules loads ignore rules from repoRoot looking for .kanaignore, then .gitignore.
func LoadIgnoreRules(repoRoot string) *IgnoreEngine {
	engine := &IgnoreEngine{
		patterns: make([]ignoreRule, 0),
	}

	// Always ignore VCS internal directories
	engine.AddRule(".git/")
	engine.AddRule(".kana/")
	engine.AddRule(".DS_Store")
	engine.AddRule("Thumbs.db")

	// Try reading .kanaignore first
	kanaIgnorePath := filepath.Join(repoRoot, ".kanaignore")
	if data, err := os.ReadFile(kanaIgnorePath); err == nil {
		engine.parseRuleBytes(data)
		return engine
	}

	// Fallback to .gitignore if present
	gitIgnorePath := filepath.Join(repoRoot, ".gitignore")
	if data, err := os.ReadFile(gitIgnorePath); err == nil {
		engine.parseRuleBytes(data)
	}

	return engine
}

// AddRule adds an explicit rule string to the engine.
func (e *IgnoreEngine) AddRule(line string) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	rule := ignoreRule{}
	if strings.HasPrefix(line, "!") {
		rule.isNegation = true
		line = strings.TrimPrefix(line, "!")
	}

	if strings.HasPrefix(line, "/") {
		rule.isRootOnly = true
		line = strings.TrimPrefix(line, "/")
	}

	if strings.HasSuffix(line, "/") {
		rule.isDirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	rule.pattern = filepath.ToSlash(line)
	e.patterns = append(e.patterns, rule)
}

func (e *IgnoreEngine) parseRuleBytes(data []byte) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		e.AddRule(scanner.Text())
	}
}

// IsIgnored tests if a relative workspace path should be excluded.
func (e *IgnoreEngine) IsIgnored(relPath string, isDir bool) bool {
	relPath = filepath.ToSlash(strings.TrimPrefix(relPath, "./"))
	if relPath == "" || relPath == "." {
		return false
	}

	ignored := false
	pathParts := strings.Split(relPath, "/")
	baseName := pathParts[len(pathParts)-1]

	for _, rule := range e.patterns {
		matched := false

		// Check ancestor directory matches first
		for i := 0; i < len(pathParts)-1; i++ {
			ancestor := pathParts[i]
			ancestorRel := strings.Join(pathParts[:i+1], "/")
			if rule.isRootOnly {
				if matchGlob(rule.pattern, ancestorRel) {
					matched = true
					break
				}
			} else {
				if matchGlob(rule.pattern, ancestor) || matchGlob(rule.pattern, ancestorRel) {
					matched = true
					break
				}
			}
		}

		if !matched {
			if rule.isDirOnly && !isDir {
				continue
			}

			if rule.isRootOnly {
				if matchGlob(rule.pattern, relPath) {
					matched = true
				}
			} else {
				if matchGlob(rule.pattern, baseName) || matchGlob(rule.pattern, relPath) {
					matched = true
				}
			}
		}

		if matched {
			if rule.isNegation {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

func matchGlob(pattern, target string) bool {
	if pattern == target {
		return true
	}
	matched, err := filepath.Match(pattern, target)
	if err == nil && matched {
		return true
	}
	// Case-insensitive fallback
	matched, err = filepath.Match(strings.ToLower(pattern), strings.ToLower(target))
	return err == nil && matched
}
