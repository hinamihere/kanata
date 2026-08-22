package core

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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
	NodeMacro    NodeType = "MACRO"
	NodeTrait    NodeType = "TRAIT"
	NodeImpl     NodeType = "IMPL"
	NodeBlock    NodeType = "BLOCK"
)

// ASTNode represents a discrete structural code element.
type ASTNode struct {
	ID          string   `json:"id"`          // Unique signature within file, e.g. "func:HandleRequest"
	Name        string   `json:"name"`        // Identifier, e.g. "HandleRequest"
	Signature   string   `json:"signature"`   // Normalized signature, e.g. "func HandleRequest(w http.ResponseWriter, r *http.Request)"
	Type        NodeType `json:"type"`        // FUNCTION, TYPE, VARIABLE, etc.
	Language    string   `json:"language"`    // "go", "c", "rust", "python", etc.
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
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		return "cpp"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts", ".tsx":
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

	// 1. Attempt Tree-sitter CGO engine first
	if err := globalTreeSitter.ParseSourceTree(fileAST, content); err == nil && len(fileAST.Nodes) > 0 {
		return fileAST, nil
	}

	// 2. High-precision semantic parser fallback
	switch lang {
	case "go":
		if err := parseGoSource(fileAST, content); err != nil {
			parseGenericSource(fileAST, content)
		}
	case "c", "cpp":
		parseCSource(fileAST, content)
	case "python":
		parsePythonSource(fileAST, content)
	case "rust":
		parseRustSource(fileAST, content)
	default:
		parseGenericSource(fileAST, content)
	}

	return fileAST, nil
}

// -----------------------------------------------------------------------------
// Go Parser
// -----------------------------------------------------------------------------

func parseGoSource(fileAST *FileAST, content []byte) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fileAST.FilePath, content, parser.ParseComments)
	if err != nil {
		return err
	}

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

// -----------------------------------------------------------------------------
// C / C++ Semantic Parser (Preprocessor & Functions & Types)
// -----------------------------------------------------------------------------

var (
	cIncludeRegex = regexp.MustCompile(`^#\s*include\s+([<"][^>"]+[>"])`)
	cDefineRegex  = regexp.MustCompile(`^#\s*define\s+([a-zA-Z_0-9]+)(\([^)]*\))?`)
	cTypeRegex    = regexp.MustCompile(`^(typedef\s+)?(struct|enum|union)\s+([a-zA-Z_0-9]+)?`)
	cFuncRegex    = regexp.MustCompile(`^[a-zA-Z_0-9*]+\s+([a-zA-Z_0-9]+)\s*\([^)]*\)\s*\{?`)
)

func parseCSource(fileAST *FileAST, content []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var currentBlock []string
	startLine := 1
	inFuncOrStruct := false
	braceDepth := 0
	currentHeader := ""
	currentType := NodeBlock

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 1. Check Include
		if incMatch := cIncludeRegex.FindStringSubmatch(trimmed); incMatch != nil && braceDepth == 0 {
			incTarget := incMatch[1]
			nodeID := fmt.Sprintf("include:%s", incTarget)
			fileAST.Nodes[nodeID] = ASTNode{
				ID:        nodeID,
				Name:      incTarget,
				Signature: fmt.Sprintf("#include %s", incTarget),
				Type:      NodeImport,
				Language:  fileAST.Language,
				StartLine: i + 1,
				EndLine:   i + 1,
				Content:   line,
				Hash:      ComputeHash([]byte(line)),
			}
			continue
		}

		// 2. Check Macro (#define)
		if defMatch := cDefineRegex.FindStringSubmatch(trimmed); defMatch != nil && braceDepth == 0 {
			macroName := defMatch[1]
			macroParams := defMatch[2]
			sig := fmt.Sprintf("#define %s%s", macroName, macroParams)
			nodeID := fmt.Sprintf("macro:%s", macroName)

			// Collect multi-line macro if trailing backslash exists
			var macroLines []string
			startM := i + 1
			for i < len(lines) {
				macroLines = append(macroLines, lines[i])
				if !strings.HasSuffix(strings.TrimRight(lines[i], " \t\r"), "\\") {
					break
				}
				i++
			}
			rawMacro := strings.Join(macroLines, "\n")
			normalized := NormalizeCode(rawMacro)

			fileAST.Nodes[nodeID] = ASTNode{
				ID:        nodeID,
				Name:      macroName,
				Signature: sig,
				Type:      NodeMacro,
				Language:  fileAST.Language,
				StartLine: startM,
				EndLine:   i + 1,
				Content:   normalized,
				Hash:      ComputeHash([]byte(normalized)),
			}
			continue
		}

		// 3. Multi-line Function or Struct / Enum block parsing
		if !inFuncOrStruct && trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			if typeMatch := cTypeRegex.FindStringSubmatch(trimmed); typeMatch != nil {
				inFuncOrStruct = true
				typeName := typeMatch[3]
				if typeName == "" {
					typeName = "anonymous"
				}
				currentHeader = fmt.Sprintf("%s %s", typeMatch[2], typeName)
				currentType = NodeTypeDecl
				startLine = i + 1
				currentBlock = nil
			} else if funcMatch := cFuncRegex.FindStringSubmatch(trimmed); funcMatch != nil && !strings.HasPrefix(trimmed, "return ") && !strings.HasPrefix(trimmed, "typedef ") {
				inFuncOrStruct = true
				funcName := funcMatch[1]
				currentHeader = fmt.Sprintf("func %s", funcName)
				currentType = NodeFunction
				startLine = i + 1
				currentBlock = nil
			}
		}

		if inFuncOrStruct {
			currentBlock = append(currentBlock, line)
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")

			if braceDepth <= 0 && (strings.Contains(line, "}") || strings.HasSuffix(trimmed, ";")) {
				raw := strings.Join(currentBlock, "\n")
				normalized := NormalizeCode(raw)
				nodeID := fmt.Sprintf("%s:%s", strings.ToLower(string(currentType)), currentHeader)

				fileAST.Nodes[nodeID] = ASTNode{
					ID:        nodeID,
					Name:      currentHeader,
					Signature: currentHeader,
					Type:      currentType,
					Language:  fileAST.Language,
					StartLine: startLine,
					EndLine:   i + 1,
					Content:   normalized,
					Hash:      ComputeHash([]byte(normalized)),
				}

				inFuncOrStruct = false
				currentBlock = nil
				braceDepth = 0
			}
			continue
		}

		// Non-structural lines collected into generic blocks if needed
		if trimmed != "" {
			currentBlock = append(currentBlock, line)
		} else if len(currentBlock) > 0 {
			emitGenericBlock(fileAST, currentBlock, startLine, i, i+1)
			currentBlock = nil
			startLine = i + 2
		}
	}

	if len(currentBlock) > 0 && !inFuncOrStruct {
		emitGenericBlock(fileAST, currentBlock, startLine, len(lines), len(lines))
	}
}

// -----------------------------------------------------------------------------
// Python Semantic Parser (Def, Async Def, Class, Decorators)
// -----------------------------------------------------------------------------

var (
	pyImportRegex = regexp.MustCompile(`^(import\s+[^#]+|from\s+[a-zA-Z_0-9.]+\s+import\s+[^#]+)`)
	pyDefRegex    = regexp.MustCompile(`^(async\s+)?def\s+([a-zA-Z_0-9]+)\s*\((.*)\)\s*(->\s*[^:]+)?\s*:`)
	pyClassRegex  = regexp.MustCompile(`^class\s+([a-zA-Z_0-9]+)(\([^)]*\))?\s*:`)
)

func parsePythonSource(fileAST *FileAST, content []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var currentBlock []string
	var pendingDecorators []string
	startLine := 1
	inBlock := false
	blockIndent := 0
	nodeName := ""
	nodeSig := ""
	nodeType := NodeFunction

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			if inBlock {
				currentBlock = append(currentBlock, line)
			}
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Decorator detection (@property, @dataclass, @app.route)
		if strings.HasPrefix(trimmed, "@") && !inBlock {
			pendingDecorators = append(pendingDecorators, line)
			if len(pendingDecorators) == 1 {
				startLine = i + 1
			}
			continue
		}

		// Import detection
		if pyImportRegex.MatchString(trimmed) && !inBlock && indent == 0 {
			nodeID := fmt.Sprintf("import:%s", ComputeHash([]byte(trimmed))[:8])
			fileAST.Nodes[nodeID] = ASTNode{
				ID:        nodeID,
				Name:      trimmed,
				Signature: trimmed,
				Type:      NodeImport,
				Language:  "python",
				StartLine: i + 1,
				EndLine:   i + 1,
				Content:   trimmed,
				Hash:      ComputeHash([]byte(trimmed)),
			}
			pendingDecorators = nil
			continue
		}

		// Function definition
		if match := pyDefRegex.FindStringSubmatch(trimmed); match != nil && (!inBlock || indent <= blockIndent) {
			if inBlock {
				emitPythonNode(fileAST, currentBlock, nodeName, nodeSig, nodeType, startLine, i)
			}

			inBlock = true
			blockIndent = indent
			funcName := match[2]
			prefix := "def"
			if match[1] != "" {
				prefix = "async def"
			}
			nodeName = funcName
			nodeSig = fmt.Sprintf("%s %s()", prefix, funcName)
			nodeType = NodeFunction

			if len(pendingDecorators) > 0 {
				currentBlock = append([]string{}, pendingDecorators...)
				pendingDecorators = nil
			} else {
				startLine = i + 1
				currentBlock = nil
			}
			currentBlock = append(currentBlock, line)
			continue
		}

		// Class definition
		if match := pyClassRegex.FindStringSubmatch(trimmed); match != nil && (!inBlock || indent <= blockIndent) {
			if inBlock {
				emitPythonNode(fileAST, currentBlock, nodeName, nodeSig, nodeType, startLine, i)
			}

			inBlock = true
			blockIndent = indent
			className := match[1]
			nodeName = className
			nodeSig = fmt.Sprintf("class %s", className)
			nodeType = NodeTypeDecl

			if len(pendingDecorators) > 0 {
				currentBlock = append([]string{}, pendingDecorators...)
				pendingDecorators = nil
			} else {
				startLine = i + 1
				currentBlock = nil
			}
			currentBlock = append(currentBlock, line)
			continue
		}

		// Inside an existing function or class block
		if inBlock {
			if indent > blockIndent || trimmed == "" {
				currentBlock = append(currentBlock, line)
			} else {
				// Block ended
				emitPythonNode(fileAST, currentBlock, nodeName, nodeSig, nodeType, startLine, i)
				inBlock = false
				currentBlock = nil
				i-- // reprocess current line
			}
		}
	}

	if inBlock && len(currentBlock) > 0 {
		emitPythonNode(fileAST, currentBlock, nodeName, nodeSig, nodeType, startLine, len(lines))
	}
}

func emitPythonNode(fileAST *FileAST, lines []string, name, sig string, nType NodeType, start, end int) {
	raw := strings.Join(lines, "\n")
	normalized := NormalizeCode(raw)
	nodeID := fmt.Sprintf("%s:%s", strings.ToLower(string(nType)), name)

	fileAST.Nodes[nodeID] = ASTNode{
		ID:        nodeID,
		Name:      name,
		Signature: sig,
		Type:      nType,
		Language:  "python",
		StartLine: start,
		EndLine:   end,
		Content:   normalized,
		Hash:      ComputeHash([]byte(normalized)),
	}
}

// -----------------------------------------------------------------------------
// Rust Semantic Parser (Fn, Struct, Enum, Trait, Impl, Macros)
// -----------------------------------------------------------------------------

var (
	rsUseRegex    = regexp.MustCompile(`^(pub\s+)?use\s+([^;]+);`)
	rsFnRegex     = regexp.MustCompile(`^(pub(\([^)]+\))?\s+)?(async\s+)?(unsafe\s+)?fn\s+([a-zA-Z_0-9]+)`)
	rsStructRegex = regexp.MustCompile(`^(pub(\([^)]+\))?\s+)?struct\s+([a-zA-Z_0-9]+)`)
	rsEnumRegex   = regexp.MustCompile(`^(pub(\([^)]+\))?\s+)?enum\s+([a-zA-Z_0-9]+)`)
	rsTraitRegex  = regexp.MustCompile(`^(pub(\([^)]+\))?\s+)?trait\s+([a-zA-Z_0-9]+)`)
	rsImplRegex   = regexp.MustCompile(`^impl(\s*<[^>]+>)?\s+([^{]+)\s*\{?`)
	rsMacroRegex  = regexp.MustCompile(`^macro_rules!\s+([a-zA-Z_0-9]+)`)
)

func parseRustSource(fileAST *FileAST, content []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var currentBlock []string
	startLine := 1
	inBlock := false
	braceDepth := 0
	nodeName := ""
	nodeSig := ""
	nodeType := NodeFunction

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			if inBlock {
				currentBlock = append(currentBlock, line)
			}
			continue
		}

		// Use statement
		if match := rsUseRegex.FindStringSubmatch(trimmed); match != nil && braceDepth == 0 {
			useTarget := match[2]
			nodeID := fmt.Sprintf("use:%s", useTarget)
			fileAST.Nodes[nodeID] = ASTNode{
				ID:        nodeID,
				Name:      useTarget,
				Signature: fmt.Sprintf("use %s", useTarget),
				Type:      NodeImport,
				Language:  "rust",
				StartLine: i + 1,
				EndLine:   i + 1,
				Content:   trimmed,
				Hash:      ComputeHash([]byte(trimmed)),
			}
			continue
		}

		// Top-level construct identification
		if !inBlock {
			if fnMatch := rsFnRegex.FindStringSubmatch(trimmed); fnMatch != nil {
				inBlock = true
				nodeName = fnMatch[5]
				nodeSig = fmt.Sprintf("fn %s", nodeName)
				nodeType = NodeFunction
				startLine = i + 1
				currentBlock = nil
			} else if stMatch := rsStructRegex.FindStringSubmatch(trimmed); stMatch != nil {
				inBlock = true
				nodeName = stMatch[3]
				nodeSig = fmt.Sprintf("struct %s", nodeName)
				nodeType = NodeTypeDecl
				startLine = i + 1
				currentBlock = nil
			} else if enMatch := rsEnumRegex.FindStringSubmatch(trimmed); enMatch != nil {
				inBlock = true
				nodeName = enMatch[3]
				nodeSig = fmt.Sprintf("enum %s", nodeName)
				nodeType = NodeTypeDecl
				startLine = i + 1
				currentBlock = nil
			} else if trMatch := rsTraitRegex.FindStringSubmatch(trimmed); trMatch != nil {
				inBlock = true
				nodeName = trMatch[3]
				nodeSig = fmt.Sprintf("trait %s", nodeName)
				nodeType = NodeTrait
				startLine = i + 1
				currentBlock = nil
			} else if implMatch := rsImplRegex.FindStringSubmatch(trimmed); implMatch != nil {
				inBlock = true
				nodeName = strings.TrimSpace(implMatch[2])
				nodeSig = fmt.Sprintf("impl %s", nodeName)
				nodeType = NodeImpl
				startLine = i + 1
				currentBlock = nil
			} else if macroMatch := rsMacroRegex.FindStringSubmatch(trimmed); macroMatch != nil {
				inBlock = true
				nodeName = macroMatch[1]
				nodeSig = fmt.Sprintf("macro_rules! %s", nodeName)
				nodeType = NodeMacro
				startLine = i + 1
				currentBlock = nil
			}
		}

		if inBlock {
			currentBlock = append(currentBlock, line)
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")

			if (braceDepth <= 0 && strings.Contains(line, "}")) || (braceDepth == 0 && strings.HasSuffix(trimmed, ";")) {
				raw := strings.Join(currentBlock, "\n")
				normalized := NormalizeCode(raw)
				nodeID := fmt.Sprintf("%s:%s", strings.ToLower(string(nodeType)), nodeName)

				fileAST.Nodes[nodeID] = ASTNode{
					ID:        nodeID,
					Name:      nodeName,
					Signature: nodeSig,
					Type:      nodeType,
					Language:  "rust",
					StartLine: startLine,
					EndLine:   i + 1,
					Content:   normalized,
					Hash:      ComputeHash([]byte(normalized)),
				}

				inBlock = false
				currentBlock = nil
				braceDepth = 0
			}
			continue
		}
	}
}

// -----------------------------------------------------------------------------
// Generic Block Parser (Fallback)
// -----------------------------------------------------------------------------

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
