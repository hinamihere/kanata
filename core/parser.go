package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NodeType represents the semantic category of an AST node.
type NodeType string

const (
	NodeFunction NodeType = "FUNCTION"
	NodeTypeDecl NodeType = "TYPE"
	NodeVariable NodeType = "VARIABLE"
	NodeConstant NodeType = "CONSTANT"
	NodeImport   NodeType = "IMPORT"
	NodePackage  NodeType = "PACKAGE"
	NodeBlock    NodeType = "BLOCK"
)

// ASTNode represents a discrete structural code element.
type ASTNode struct {
	ID          string   `json:"id"`          // Unique signature within file, e.g. "func:HandleRequest"
	Name        string   `json:"name"`        // Identifier, e.g. "HandleRequest"
	Signature   string   `json:"signature"`   // Normalized signature, e.g. "func HandleRequest(w http.ResponseWriter, r *http.Request)"
	Type        NodeType `json:"type"`        // FUNCTION, TYPE, VARIABLE, etc.
	Language    string   `json:"language"`    // "go", "c", etc.
	StartLine   int      `json:"start_line"`
	EndLine     int      `json:"end_line"`
	Content     string   `json:"content"`     // Raw source content of the node
	Hash        string   `json:"hash"`        // SHA-256 hash of normalized content
	DocComment  string   `json:"doc_comment"` // Associated doc comments
}

// FileAST encapsulates all semantic structural nodes within a file.
type FileAST struct {
	FilePath string             `json:"file_path"`
	Language string             `json:"language"`
	Nodes    map[string]ASTNode `json:"nodes"`    // Keyed by ASTNode.ID
	RawHash  string             `json:"raw_hash"` // Whole-file content hash
}

// ComputeHash computes a SHA-256 hexadecimal digest of input data.
func ComputeHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// NormalizeCode strips extraneous outer whitespace while preserving internal structure.
func NormalizeCode(code string) string {
	lines := strings.Split(code, "\n")
	var cleaned []string
	for _, l := range lines {
		trimmed := strings.TrimRight(l, " \t\r")
		cleaned = append(cleaned, trimmed)
	}
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[0]) == "" {
		cleaned = cleaned[1:]
	}
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}
	return strings.Join(cleaned, "\n")
}

// DetectLanguage identifies the source language by file extension.
func DetectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	default:
		return "generic"
	}
}

// ParseFile parses a source code file into a structured FileAST with semantic nodes.
func ParseFile(filePath string) (*FileAST, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return ParseSource(filePath, content)
}

var globalTreeSitter = NewTreeSitterEngine()

// ParseSource parses in-memory source content into a FileAST.
func ParseSource(filePath string, content []byte) (*FileAST, error) {
	lang := DetectLanguage(filePath)
	rawHash := ComputeHash(content)

	fileAST := &FileAST{
		FilePath: filePath,
		Language: lang,
		Nodes:    make(map[string]ASTNode),
		RawHash:  rawHash,
	}

	// Attempt Tree-sitter parsing first
	if err := globalTreeSitter.ParseSourceTree(fileAST, content); err == nil && len(fileAST.Nodes) > 0 {
		return fileAST, nil
	}

	// Fallback to high-precision Go AST or generic block parser
	switch lang {
	case "go":
		if err := parseGoSource(fileAST, content); err != nil {
			parseGenericSource(fileAST, content)
		}
	default:
		parseGenericSource(fileAST, content)
	}

	return fileAST, nil
}

// parseGoSource parses Go source files using Go's AST engine.
func parseGoSource(fileAST *FileAST, content []byte) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileAST.FilePath, content, parser.ParseComments)
	if err != nil {
		return err
	}

	// Package declaration node
	if node.Name != nil {
		pkgName := node.Name.Name
		pkgNode := ASTNode{
			ID:        fmt.Sprintf("pkg:%s", pkgName),
			Name:      pkgName,
			Signature: fmt.Sprintf("package %s", pkgName),
			Type:      NodePackage,
			Language:  "go",
			StartLine: fset.Position(node.Package).Line,
			EndLine:   fset.Position(node.Name.End()).Line,
			Content:   fmt.Sprintf("package %s", pkgName),
			Hash:      ComputeHash([]byte(fmt.Sprintf("package %s", pkgName))),
		}
		fileAST.Nodes[pkgNode.ID] = pkgNode
	}

	// Iterate declarations
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			extractGoFunc(fset, content, d, fileAST)
		case *ast.GenDecl:
			extractGoGenDecl(fset, content, d, fileAST)
		}
	}

	return nil
}

func extractGoFunc(fset *token.FileSet, content []byte, d *ast.FuncDecl, fileAST *FileAST) {
	startOffset := fset.Position(d.Pos()).Offset
	endOffset := fset.Position(d.End()).Offset
	if startOffset < 0 || endOffset > len(content) || startOffset >= endOffset {
		return
	}

	rawContent := string(content[startOffset:endOffset])
	normalized := NormalizeCode(rawContent)

	var recvStr string
	if d.Recv != nil && len(d.Recv.List) > 0 {
		recvType := formatNode(fset, d.Recv.List[0].Type)
		recvStr = fmt.Sprintf("(%s) ", recvType)
	}

	funcName := d.Name.Name
	sig := fmt.Sprintf("func %s%s", recvStr, funcName)
	nodeID := fmt.Sprintf("func:%s%s", recvStr, funcName)

	var doc string
	if d.Doc != nil {
		doc = d.Doc.Text()
	}

	astNode := ASTNode{
		ID:         nodeID,
		Name:       funcName,
		Signature:  sig,
		Type:       NodeFunction,
		Language:   "go",
		StartLine:  fset.Position(d.Pos()).Line,
		EndLine:    fset.Position(d.End()).Line,
		Content:    normalized,
		Hash:       ComputeHash([]byte(normalized)),
		DocComment: strings.TrimSpace(doc),
	}
	fileAST.Nodes[astNode.ID] = astNode
}

func extractGoGenDecl(fset *token.FileSet, content []byte, d *ast.GenDecl, fileAST *FileAST) {
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			startOffset := fset.Position(s.Pos()).Offset
			endOffset := fset.Position(s.End()).Offset
			if startOffset < 0 || endOffset > len(content) || startOffset >= endOffset {
				continue
			}

			rawContent := string(content[startOffset:endOffset])
			normalized := NormalizeCode(rawContent)
			typeName := s.Name.Name
			sig := fmt.Sprintf("type %s", typeName)
			nodeID := fmt.Sprintf("type:%s", typeName)

			var doc string
			if d.Doc != nil {
				doc = d.Doc.Text()
			}

			astNode := ASTNode{
				ID:         nodeID,
				Name:       typeName,
				Signature:  sig,
				Type:       NodeTypeDecl,
				Language:   "go",
				StartLine:  fset.Position(s.Pos()).Line,
				EndLine:    fset.Position(s.End()).Line,
				Content:    normalized,
				Hash:       ComputeHash([]byte(normalized)),
				DocComment: strings.TrimSpace(doc),
			}
			fileAST.Nodes[astNode.ID] = astNode

		case *ast.ValueSpec:
			var nodeType NodeType
			var prefix string
			switch d.Tok {
			case token.CONST:
				nodeType = NodeConstant
				prefix = "const"
			default:
				nodeType = NodeVariable
				prefix = "var"
			}

			for _, name := range s.Names {
				startOffset := fset.Position(s.Pos()).Offset
				endOffset := fset.Position(s.End()).Offset
				if startOffset < 0 || endOffset > len(content) || startOffset >= endOffset {
					continue
				}

				rawContent := string(content[startOffset:endOffset])
				normalized := NormalizeCode(rawContent)
				varName := name.Name
				sig := fmt.Sprintf("%s %s", prefix, varName)
				nodeID := fmt.Sprintf("%s:%s", prefix, varName)

				astNode := ASTNode{
					ID:        nodeID,
					Name:      varName,
					Signature: sig,
					Type:      nodeType,
					Language:  "go",
					StartLine: fset.Position(s.Pos()).Line,
					EndLine:   fset.Position(s.End()).Line,
					Content:   normalized,
					Hash:      ComputeHash([]byte(normalized)),
				}
				fileAST.Nodes[astNode.ID] = astNode
			}

		case *ast.ImportSpec:
				pathVal := ""
				if s.Path != nil {
					pathVal = s.Path.Value
				}
				nodeID := fmt.Sprintf("import:%s", pathVal)
				astNode := ASTNode{
					ID:        nodeID,
					Name:      pathVal,
					Signature: fmt.Sprintf("import %s", pathVal),
					Type:      NodeImport,
					Language:  "go",
					StartLine: fset.Position(s.Pos()).Line,
					EndLine:   fset.Position(s.End()).Line,
					Content:   pathVal,
					Hash:      ComputeHash([]byte(pathVal)),
				}
				fileAST.Nodes[astNode.ID] = astNode
		}
	}
}

func formatNode(fset *token.FileSet, node ast.Node) string {
	if node == nil {
		return ""
	}
	switch n := node.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.StarExpr:
		return "*" + formatNode(fset, n.X)
	case *ast.SelectorExpr:
		return formatNode(fset, n.X) + "." + n.Sel.Name
	default:
		return "unknown"
	}
}

// parseGenericSource segments source files by top-level functional and declaration blocks.
func parseGenericSource(fileAST *FileAST, content []byte) {
	lines := strings.Split(string(content), "\n")
	var currentBlock []string
	startLine := 1
	blockIdx := 1

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(currentBlock) > 0 {
				emitGenericBlock(fileAST, currentBlock, startLine, idx, blockIdx)
				blockIdx++
				currentBlock = nil
			}
			startLine = idx + 2
			continue
		}
		currentBlock = append(currentBlock, line)
	}

	if len(currentBlock) > 0 {
		emitGenericBlock(fileAST, currentBlock, startLine, len(lines), blockIdx)
	}
}

func emitGenericBlock(fileAST *FileAST, lines []string, start, end, idx int) {
	raw := strings.Join(lines, "\n")
	normalized := NormalizeCode(raw)
	if normalized == "" {
		return
	}
	id := fmt.Sprintf("block:%d", idx)
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 50 {
		firstLine = firstLine[:47] + "..."
	}

	astNode := ASTNode{
		ID:        id,
		Name:      fmt.Sprintf("Block #%d", idx),
		Signature: firstLine,
		Type:      NodeBlock,
		Language:  fileAST.Language,
		StartLine: start,
		EndLine:   end,
		Content:   normalized,
		Hash:      ComputeHash([]byte(normalized)),
	}
	fileAST.Nodes[astNode.ID] = astNode
}

// SortedNodes returns all nodes sorted alphabetically by their ID.
func (f *FileAST) SortedNodes() []ASTNode {
	nodes := make([]ASTNode, 0, len(f.Nodes))
	for _, n := range f.Nodes {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes
}
