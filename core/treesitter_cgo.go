//go:build cgo

package core

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
)

// TreeSitterEngine wraps tree-sitter C-bindings for multi-language AST parsing when CGO is enabled.
type TreeSitterEngine struct {
	parsers map[string]*sitter.Parser
}

// NewTreeSitterEngine initializes the tree-sitter parser suite.
func NewTreeSitterEngine() *TreeSitterEngine {
	engine := &TreeSitterEngine{
		parsers: make(map[string]*sitter.Parser),
	}

	// Initialize Go grammar parser
	goParser := sitter.NewParser()
	goParser.SetLanguage(golang.GetLanguage())
	engine.parsers["go"] = goParser

	return engine
}

// ParseSourceTree parses source content using Tree-sitter grammar and populates FileAST nodes.
func (e *TreeSitterEngine) ParseSourceTree(fileAST *FileAST, content []byte) error {
	parser, ok := e.parsers[fileAST.Language]
	if !ok || parser == nil {
		return fmt.Errorf("tree-sitter grammar for %s not configured", fileAST.Language)
	}

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return fmt.Errorf("tree-sitter parse failed: %w", err)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		return fmt.Errorf("empty syntax tree")
	}

	e.extractNodes(root, content, fileAST)
	return nil
}

func (e *TreeSitterEngine) extractNodes(root *sitter.Node, content []byte, fileAST *FileAST) {
	childCount := int(root.NamedChildCount())
	for i := 0; i < childCount; i++ {
		child := root.NamedChild(i)
		if child == nil {
			continue
		}

		nodeTypeStr := child.Type()
		switch nodeTypeStr {
		case "function_declaration", "method_declaration":
			e.extractFunctionNode(child, content, fileAST)
		case "type_declaration":
			e.extractTypeNode(child, content, fileAST)
		case "var_declaration":
			e.extractVarNode(child, content, fileAST, NodeVariable)
		case "const_declaration":
			e.extractVarNode(child, content, fileAST, NodeConstant)
		case "import_declaration":
			e.extractImportNode(child, content, fileAST)
		case "package_clause":
			e.extractPackageNode(child, content, fileAST)
		}
	}
}

func (e *TreeSitterEngine) extractFunctionNode(node *sitter.Node, content []byte, fileAST *FileAST) {
	raw := node.Content(content)
	normalized := NormalizeCode(raw)

	nameNode := node.ChildByFieldName("name")
	name := "anonymous"
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	sig := fmt.Sprintf("func %s", name)
	nodeID := fmt.Sprintf("func:%s", name)

	receiverNode := node.ChildByFieldName("receiver")
	if receiverNode != nil {
		recvContent := receiverNode.Content(content)
		sig = fmt.Sprintf("func %s %s", recvContent, name)
		nodeID = fmt.Sprintf("func:%s %s", recvContent, name)
	}

	fileAST.Nodes[nodeID] = ASTNode{
		ID:        nodeID,
		Name:      name,
		Signature: sig,
		Type:      NodeFunction,
		Language:  fileAST.Language,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   normalized,
		Hash:      ComputeHash([]byte(normalized)),
	}
}

func (e *TreeSitterEngine) extractTypeNode(node *sitter.Node, content []byte, fileAST *FileAST) {
	raw := node.Content(content)
	normalized := NormalizeCode(raw)

	count := int(node.NamedChildCount())
	for i := 0; i < count; i++ {
		spec := node.NamedChild(i)
		if spec == nil || spec.Type() != "type_spec" {
			continue
		}
		nameNode := spec.ChildByFieldName("name")
		name := "AnonymousType"
		if nameNode != nil {
			name = nameNode.Content(content)
		}

		sig := fmt.Sprintf("type %s", name)
		nodeID := fmt.Sprintf("type:%s", name)

		fileAST.Nodes[nodeID] = ASTNode{
			ID:        nodeID,
			Name:      name,
			Signature: sig,
			Type:      NodeTypeDecl,
			Language:  fileAST.Language,
			StartLine: int(node.StartPoint().Row) + 1,
			EndLine:   int(node.EndPoint().Row) + 1,
			Content:   normalized,
			Hash:      ComputeHash([]byte(normalized)),
		}
	}
}

func (e *TreeSitterEngine) extractVarNode(node *sitter.Node, content []byte, fileAST *FileAST, nType NodeType) {
	raw := node.Content(content)
	normalized := NormalizeCode(raw)

	firstLine := strings.TrimSpace(strings.Split(raw, "\n")[0])
	nodeID := fmt.Sprintf("%s:%s", strings.ToLower(string(nType)), ComputeHash([]byte(firstLine))[:8])

	fileAST.Nodes[nodeID] = ASTNode{
		ID:        nodeID,
		Name:      firstLine,
		Signature: firstLine,
		Type:      nType,
		Language:  fileAST.Language,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   normalized,
		Hash:      ComputeHash([]byte(normalized)),
	}
}

func (e *TreeSitterEngine) extractImportNode(node *sitter.Node, content []byte, fileAST *FileAST) {
	raw := node.Content(content)
	normalized := NormalizeCode(raw)
	nodeID := fmt.Sprintf("import:%s", ComputeHash([]byte(normalized))[:8])

	fileAST.Nodes[nodeID] = ASTNode{
		ID:        nodeID,
		Name:      "import",
		Signature: "import (...)",
		Type:      NodeImport,
		Language:  fileAST.Language,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   normalized,
		Hash:      ComputeHash([]byte(normalized)),
	}
}

func (e *TreeSitterEngine) extractPackageNode(node *sitter.Node, content []byte, fileAST *FileAST) {
	raw := strings.TrimSpace(node.Content(content))
	nodeID := fmt.Sprintf("pkg:%s", raw)

	fileAST.Nodes[nodeID] = ASTNode{
		ID:        nodeID,
		Name:      raw,
		Signature: raw,
		Type:      NodePackage,
		Language:  fileAST.Language,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   raw,
		Hash:      ComputeHash([]byte(raw)),
	}
}
