package core

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// InferIntent synthesizes a clean, professional intent message from a WorkspaceDiff.
func InferIntent(diff *WorkspaceDiff) string {
	if diff == nil || len(diff.Files) == 0 {
		return "Workspace snapshot"
	}

	var addedFuncs, modFuncs, remFuncs []string
	var addedTypes, modTypes, remTypes []string
	var addedMacros, modMacros, remMacros []string
	var addedFiles, remFiles []string

	for filePath, fd := range diff.Files {
		baseName := filepath.Base(filePath)

		if fd.ChangeType == ChangeAdded && len(fd.NodeDiffs) == 0 {
			addedFiles = append(addedFiles, baseName)
			continue
		}
		if fd.ChangeType == ChangeRemoved && len(fd.NodeDiffs) == 0 {
			remFiles = append(remFiles, baseName)
			continue
		}

		for _, nd := range fd.NodeDiffs {
			name := nd.NodeName
			if name == "" {
				name = nd.Signature
			}

			switch nd.NodeType {
			case NodeFunction:
				switch nd.ChangeType {
				case ChangeAdded:
					addedFuncs = append(addedFuncs, name)
				case ChangeModified:
					modFuncs = append(modFuncs, name)
				case ChangeRemoved:
					remFuncs = append(remFuncs, name)
				}
			case NodeTypeDecl, NodeTrait, NodeImpl:
				switch nd.ChangeType {
				case ChangeAdded:
					addedTypes = append(addedTypes, name)
				case ChangeModified:
					modTypes = append(modTypes, name)
				case ChangeRemoved:
					remTypes = append(remTypes, name)
				}
			case NodeMacro, NodeConstant:
				switch nd.ChangeType {
				case ChangeAdded:
					addedMacros = append(addedMacros, name)
				case ChangeModified:
					modMacros = append(modMacros, name)
				case ChangeRemoved:
					remMacros = append(remMacros, name)
				}
			}
		}
	}

	sort.Strings(addedFuncs)
	sort.Strings(modFuncs)
	sort.Strings(remFuncs)
	sort.Strings(addedTypes)
	sort.Strings(modTypes)
	sort.Strings(remTypes)
	sort.Strings(addedMacros)
	sort.Strings(modMacros)
	sort.Strings(remMacros)

	var clauses []string

	// 1. Added types/structs
	if len(addedTypes) == 1 {
		clauses = append(clauses, fmt.Sprintf("Add %s type", addedTypes[0]))
	} else if len(addedTypes) > 1 {
		clauses = append(clauses, fmt.Sprintf("Add %s types", formatList(addedTypes, 2)))
	}

	// 2. Added functions
	if len(addedFuncs) == 1 {
		clauses = append(clauses, fmt.Sprintf("Add %s function", addedFuncs[0]))
	} else if len(addedFuncs) > 1 {
		clauses = append(clauses, fmt.Sprintf("Add %s functions", formatList(addedFuncs, 2)))
	}

	// 3. Modified functions
	if len(modFuncs) == 1 {
		clauses = append(clauses, fmt.Sprintf("Update %s function", modFuncs[0]))
	} else if len(modFuncs) > 1 {
		clauses = append(clauses, fmt.Sprintf("Update %s functions", formatList(modFuncs, 2)))
	}

	// 4. Modified types
	if len(modTypes) == 1 {
		clauses = append(clauses, fmt.Sprintf("Update %s struct", modTypes[0]))
	} else if len(modTypes) > 1 {
		clauses = append(clauses, fmt.Sprintf("Update %s structs", formatList(modTypes, 2)))
	}

	// 5. Added macros
	if len(addedMacros) == 1 {
		clauses = append(clauses, fmt.Sprintf("Add %s macro", addedMacros[0]))
	} else if len(addedMacros) > 1 {
		clauses = append(clauses, fmt.Sprintf("Add %s macros", formatList(addedMacros, 2)))
	}

	// 6. Modified macros
	if len(modMacros) == 1 {
		clauses = append(clauses, fmt.Sprintf("Update %s macro", modMacros[0]))
	} else if len(modMacros) > 1 {
		clauses = append(clauses, fmt.Sprintf("Update %s macros", formatList(modMacros, 2)))
	}

	// 7. Removed functions/types
	if len(remFuncs) > 0 {
		clauses = append(clauses, fmt.Sprintf("Remove %s function(s)", formatList(remFuncs, 2)))
	}
	if len(remTypes) > 0 {
		clauses = append(clauses, fmt.Sprintf("Remove %s type(s)", formatList(remTypes, 2)))
	}

	// 8. Whole files added/removed
	if len(addedFiles) > 0 {
		clauses = append(clauses, fmt.Sprintf("Add %s", formatList(addedFiles, 2)))
	}
	if len(remFiles) > 0 {
		clauses = append(clauses, fmt.Sprintf("Remove %s", formatList(remFiles, 2)))
	}

	if len(clauses) == 0 {
		// Generic fallback if only block/import edits
		if len(diff.Files) == 1 {
			for f := range diff.Files {
				return fmt.Sprintf("Update %s", filepath.Base(f))
			}
		}
		return fmt.Sprintf("Update %d file(s)", len(diff.Files))
	}

	if len(clauses) == 1 {
		return clauses[0]
	}

	if len(clauses) == 2 {
		// e.g. "Add ServerConfig type and update Process function"
		return fmt.Sprintf("%s and %s", clauses[0], lowercaseFirstWord(clauses[1]))
	}

	// 3 or more clauses: join first two and summarize
	return fmt.Sprintf("%s, %s, and other updates", clauses[0], lowercaseFirstWord(clauses[1]))
}

func lowercaseFirstWord(s string) string {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 2 {
		return strings.ToLower(parts[0]) + " " + parts[1]
	}
	return strings.ToLower(s)
}

func formatList(items []string, maxItems int) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) <= maxItems {
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
	return fmt.Sprintf("%s and %d more", strings.Join(items[:maxItems], ", "), len(items)-maxItems)
}
