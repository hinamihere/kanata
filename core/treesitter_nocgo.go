//go:build !cgo

package core

import (
	"fmt"
)

// TreeSitterEngine stub when CGO is disabled or C compiler is absent.
type TreeSitterEngine struct{}

// NewTreeSitterEngine returns the no-cgo AST fallback engine.
func NewTreeSitterEngine() *TreeSitterEngine {
	return &TreeSitterEngine{}
}

// ParseSourceTree returns a fallback notice to trigger pure-Go AST parsing.
func (e *TreeSitterEngine) ParseSourceTree(fileAST *FileAST, content []byte) error {
	return fmt.Errorf("cgo disabled; using native AST parser")
}
