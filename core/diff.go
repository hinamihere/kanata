package core

import (
	"fmt"
	"sort"
	"strings"
)

// ChangeType defines the semantic nature of an AST modification.
type ChangeType string

const (
	ChangeAdded     ChangeType = "added"
	ChangeRemoved   ChangeType = "removed"
	ChangeModified  ChangeType = "modified"
	ChangeRenamed   ChangeType = "renamed"
	ChangeUnchanged ChangeType = "unchanged"
	ChangeConflict  ChangeType = "conflict"
)

// NodeDiff captures a granular structural change on an AST node.
type NodeDiff struct {
	NodeID     string     `json:"node_id"`
	NodeName   string     `json:"node_name"`
	NodeType   NodeType   `json:"node_type"`
	Signature  string     `json:"signature"`
	ChangeType ChangeType `json:"change_type"`
	OldHash    string     `json:"old_hash,omitempty"`
	NewHash    string     `json:"new_hash,omitempty"`
	OldNode    *ASTNode   `json:"old_node,omitempty"`
	NewNode    *ASTNode   `json:"new_node,omitempty"`
	Details    string     `json:"details,omitempty"`
}

// FileDiff aggregates semantic changes across all nodes within a single file.
type FileDiff struct {
	FilePath    string     `json:"file_path"`
	OldFilePath string     `json:"old_file_path,omitempty"`
	ChangeType  ChangeType `json:"change_type"`
	Similarity  float64    `json:"similarity,omitempty"`
	NodeDiffs   []NodeDiff `json:"node_diffs"`
	OldRawHash  string     `json:"old_raw_hash,omitempty"`
	NewRawHash  string     `json:"new_raw_hash,omitempty"`
}

// WorkspaceDiff represents the complete semantic delta across the entire repository.
type WorkspaceDiff struct {
	Files              map[string]FileDiff `json:"files"`
	AddedNodesCount    int                 `json:"added_nodes_count"`
	RemovedNodesCount  int                 `json:"removed_nodes_count"`
	ModifiedNodesCount int                 `json:"modified_nodes_count"`
}

// DiffFiles performs AST-level node comparison between an old and a new FileAST.
func DiffFiles(oldAST, newAST *FileAST) FileDiff {
	if oldAST == nil && newAST == nil {
		return FileDiff{ChangeType: ChangeUnchanged}
	}

	if oldAST == nil {
		diff := FileDiff{
			FilePath:   newAST.FilePath,
			ChangeType: ChangeAdded,
			NewRawHash: newAST.RawHash,
		}
		for _, node := range newAST.SortedNodes() {
			diff.NodeDiffs = append(diff.NodeDiffs, NodeDiff{
				NodeID:     node.ID,
				NodeName:   node.Name,
				NodeType:   node.Type,
				Signature:  node.Signature,
				ChangeType: ChangeAdded,
				NewHash:    node.Hash,
				NewNode:    &node,
				Details:    fmt.Sprintf("+ %s", node.Signature),
			})
		}
		return diff
	}

	if newAST == nil {
		diff := FileDiff{
			FilePath:   oldAST.FilePath,
			ChangeType: ChangeRemoved,
			OldRawHash: oldAST.RawHash,
		}
		for _, node := range oldAST.SortedNodes() {
			diff.NodeDiffs = append(diff.NodeDiffs, NodeDiff{
				NodeID:     node.ID,
				NodeName:   node.Name,
				NodeType:   node.Type,
				Signature:  node.Signature,
				ChangeType: ChangeRemoved,
				OldHash:    node.Hash,
				OldNode:    &node,
				Details:    fmt.Sprintf("- %s", node.Signature),
			})
		}
		return diff
	}

	diff := FileDiff{
		FilePath:   newAST.FilePath,
		ChangeType: ChangeUnchanged,
		OldRawHash: oldAST.RawHash,
		NewRawHash: newAST.RawHash,
	}

	if oldAST.RawHash != "" && oldAST.RawHash == newAST.RawHash && len(oldAST.Nodes) == len(newAST.Nodes) {
		identical := true
		for id, oldNode := range oldAST.Nodes {
			if newNode, ok := newAST.Nodes[id]; !ok || newNode.Hash != oldNode.Hash {
				identical = false
				break
			}
		}
		if identical {
			return diff
		}
	}

	visited := make(map[string]bool)

	for id, newNode := range newAST.Nodes {
		visited[id] = true
		oldNode, exists := oldAST.Nodes[id]
		if !exists {
			diff.NodeDiffs = append(diff.NodeDiffs, NodeDiff{
				NodeID:     id,
				NodeName:   newNode.Name,
				NodeType:   newNode.Type,
				Signature:  newNode.Signature,
				ChangeType: ChangeAdded,
				NewHash:    newNode.Hash,
				NewNode:    &newNode,
				Details:    fmt.Sprintf("+ %s", newNode.Signature),
			})
		} else if oldNode.Hash != newNode.Hash {
			diff.NodeDiffs = append(diff.NodeDiffs, NodeDiff{
				NodeID:     id,
				NodeName:   newNode.Name,
				NodeType:   newNode.Type,
				Signature:  newNode.Signature,
				ChangeType: ChangeModified,
				OldHash:    oldNode.Hash,
				NewHash:    newNode.Hash,
				OldNode:    &oldNode,
				NewNode:    &newNode,
				Details:    fmt.Sprintf("~ %s", newNode.Signature),
			})
		}
	}

	for id, oldNode := range oldAST.Nodes {
		if !visited[id] {
			diff.NodeDiffs = append(diff.NodeDiffs, NodeDiff{
				NodeID:     id,
				NodeName:   oldNode.Name,
				NodeType:   oldNode.Type,
				Signature:  oldNode.Signature,
				ChangeType: ChangeRemoved,
				OldHash:    oldNode.Hash,
				OldNode:    &oldNode,
				Details:    fmt.Sprintf("- %s", oldNode.Signature),
			})
		}
	}

	if len(diff.NodeDiffs) > 0 {
		diff.ChangeType = ChangeModified
		sort.Slice(diff.NodeDiffs, func(i, j int) bool {
			return diff.NodeDiffs[i].NodeID < diff.NodeDiffs[j].NodeID
		})
	}

	return diff
}

// DiffWorkspace compares an entire workspace between old and new FileAST mappings.
func DiffWorkspace(oldState, newState map[string]*FileAST) *WorkspaceDiff {
	diff := &WorkspaceDiff{
		Files: make(map[string]FileDiff),
	}

	allPaths := make(map[string]bool)
	for p := range oldState {
		allPaths[p] = true
	}
	for p := range newState {
		allPaths[p] = true
	}

	for p := range allPaths {
		oldF := oldState[p]
		newF := newState[p]
		fd := DiffFiles(oldF, newF)
		if fd.ChangeType != ChangeUnchanged {
			diff.Files[p] = fd
		}
	}

	// Semantic File Rename & Move Detection
	var addedFiles []string
	var removedFiles []string
	for p, fd := range diff.Files {
		if fd.ChangeType == ChangeAdded {
			addedFiles = append(addedFiles, p)
		} else if fd.ChangeType == ChangeRemoved {
			removedFiles = append(removedFiles, p)
		}
	}

	matchedOld := make(map[string]bool)
	for _, newP := range addedFiles {
		newF := newState[newP]
		if newF == nil || len(newF.Nodes) == 0 {
			continue
		}

		bestOld := ""
		bestScore := 0.0

		for _, oldP := range removedFiles {
			if matchedOld[oldP] {
				continue
			}
			oldF := oldState[oldP]
			if oldF == nil || len(oldF.Nodes) == 0 {
				continue
			}

			score := computeASTSimilarity(oldF, newF)
			if score > bestScore && score >= 0.60 {
				bestScore = score
				bestOld = oldP
			}
		}

		if bestOld != "" {
			matchedOld[bestOld] = true
			oldF := oldState[bestOld]

			renameDiff := DiffFiles(oldF, newF)
			renameDiff.ChangeType = ChangeRenamed
			renameDiff.OldFilePath = bestOld
			renameDiff.Similarity = bestScore

			delete(diff.Files, bestOld)
			diff.Files[newP] = renameDiff
		}
	}

	// Recompute total node change counters
	for _, fd := range diff.Files {
		for _, nd := range fd.NodeDiffs {
			switch nd.ChangeType {
			case ChangeAdded:
				diff.AddedNodesCount++
			case ChangeRemoved:
				diff.RemovedNodesCount++
			case ChangeModified:
				diff.ModifiedNodesCount++
			}
		}
	}

	return diff
}

func computeASTSimilarity(oldF, newF *FileAST) float64 {
	if oldF.RawHash != "" && oldF.RawHash == newF.RawHash {
		return 1.0
	}

	if len(oldF.Nodes) == 0 || len(newF.Nodes) == 0 {
		return 0.0
	}

	matchingNodes := 0
	for id, oldNode := range oldF.Nodes {
		if newNode, ok := newF.Nodes[id]; ok {
			if oldNode.Hash == newNode.Hash {
				matchingNodes++
			}
		}
	}

	maxNodes := len(oldF.Nodes)
	if len(newF.Nodes) > maxNodes {
		maxNodes = len(newF.Nodes)
	}

	return float64(matchingNodes) / float64(maxNodes)
}

// MergeResult represents the semantic 3-way merge outcome.
type MergeResult struct {
	MergedState map[string]*FileAST
	Conflicts   []NodeDiff
	HasConflict bool
}

// IntegrateWorkspaces performs a 3-way semantic AST integration.
func IntegrateWorkspaces(base, current, target map[string]*FileAST) *MergeResult {
	result := &MergeResult{
		MergedState: make(map[string]*FileAST),
		Conflicts:   make([]NodeDiff, 0),
	}

	allPaths := make(map[string]bool)
	for p := range base {
		allPaths[p] = true
	}
	for p := range current {
		allPaths[p] = true
	}
	for p := range target {
		allPaths[p] = true
	}

	for p := range allPaths {
		baseF := base[p]
		currF := current[p]
		targF := target[p]

		mergedF, conflicts := mergeFileAST(p, baseF, currF, targF)
		if len(conflicts) > 0 {
			result.HasConflict = true
			result.Conflicts = append(result.Conflicts, conflicts...)
		}
		if mergedF != nil && len(mergedF.Nodes) > 0 {
			result.MergedState[p] = mergedF
		}
	}

	return result
}

func mergeFileAST(path string, base, current, target *FileAST) (*FileAST, []NodeDiff) {
	var conflicts []NodeDiff

	lang := "generic"
	if current != nil {
		lang = current.Language
	} else if target != nil {
		lang = target.Language
	} else if base != nil {
		lang = base.Language
	}

	merged := &FileAST{
		FilePath: path,
		Language: lang,
		Nodes:    make(map[string]ASTNode),
	}

	allNodeIDs := make(map[string]bool)
	if base != nil {
		for id := range base.Nodes {
			allNodeIDs[id] = true
		}
	}
	if current != nil {
		for id := range current.Nodes {
			allNodeIDs[id] = true
		}
	}
	if target != nil {
		for id := range target.Nodes {
			allNodeIDs[id] = true
		}
	}

	for id := range allNodeIDs {
		var baseNode, currNode, targNode *ASTNode
		if base != nil {
			if n, ok := base.Nodes[id]; ok {
				baseNode = &n
			}
		}
		if current != nil {
			if n, ok := current.Nodes[id]; ok {
				currNode = &n
			}
		}
		if target != nil {
			if n, ok := target.Nodes[id]; ok {
				targNode = &n
			}
		}

		baseHash := ""
		if baseNode != nil {
			baseHash = baseNode.Hash
		}
		currHash := ""
		if currNode != nil {
			currHash = currNode.Hash
		}
		targHash := ""
		if targNode != nil {
			targHash = targNode.Hash
		}

		if currHash == targHash {
			if currNode != nil {
				merged.Nodes[id] = *currNode
			}
			continue
		}

		if currHash == baseHash && targHash != baseHash {
			if targNode != nil {
				merged.Nodes[id] = *targNode
			}
			continue
		}

		if targHash == baseHash && currHash != baseHash {
			if currNode != nil {
				merged.Nodes[id] = *currNode
			}
			continue
		}

		sig := id
		var nType NodeType = NodeBlock
		if currNode != nil {
			sig = currNode.Signature
			nType = currNode.Type
		} else if targNode != nil {
			sig = targNode.Signature
			nType = targNode.Type
		}

		conflict := NodeDiff{
			NodeID:     id,
			Signature:  sig,
			NodeType:   nType,
			ChangeType: ChangeConflict,
			OldNode:    currNode,
			NewNode:    targNode,
			Details:    fmt.Sprintf("conflict on %s (%s)", sig, path),
		}
		conflicts = append(conflicts, conflict)

		if currNode != nil {
			merged.Nodes[id] = *currNode
		}
	}

	return merged, conflicts
}

// FormatWorkspaceDiff generates a clean, subtle summary of semantic workspace changes.
func FormatWorkspaceDiff(diff *WorkspaceDiff) string {
	if diff == nil || len(diff.Files) == 0 {
		return "working tree clean (no AST changes detected)\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("semantic changes: +%d added, ~%d modified, -%d removed\n\n",
		diff.AddedNodesCount, diff.ModifiedNodesCount, diff.RemovedNodesCount))

	var sortedFiles []string
	for p := range diff.Files {
		sortedFiles = append(sortedFiles, p)
	}
	sort.Strings(sortedFiles)

	for _, p := range sortedFiles {
		fd := diff.Files[p]
		var statusLabel string
		switch fd.ChangeType {
		case ChangeRenamed:
			statusLabel = fmt.Sprintf("renamed (%.0f%% match)", fd.Similarity*100)
			sb.WriteString(fmt.Sprintf("  %-8s  %s -> %s\n", statusLabel, fd.OldFilePath, p))
		case ChangeAdded:
			statusLabel = "new file"
			sb.WriteString(fmt.Sprintf("  %-8s  %s\n", statusLabel, p))
		case ChangeRemoved:
			statusLabel = "deleted "
			sb.WriteString(fmt.Sprintf("  %-8s  %s\n", statusLabel, p))
		case ChangeModified:
			statusLabel = "modified"
			sb.WriteString(fmt.Sprintf("  %-8s  %s\n", statusLabel, p))
		}

		for _, nd := range fd.NodeDiffs {
			var symbol string
			switch nd.ChangeType {
			case ChangeAdded:
				symbol = "+"
			case ChangeRemoved:
				symbol = "-"
			case ChangeModified:
				symbol = "~"
			default:
				symbol = " "
			}
			sb.WriteString(fmt.Sprintf("    %s %s\n", symbol, nd.Signature))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatWorkspacePatch generates a detailed AST patch diff with node body transformations.
func FormatWorkspacePatch(diff *WorkspaceDiff) string {
	if diff == nil || len(diff.Files) == 0 {
		return "working tree clean (no AST changes detected)\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("semantic diff: +%d added, ~%d modified, -%d removed\n\n",
		diff.AddedNodesCount, diff.ModifiedNodesCount, diff.RemovedNodesCount))

	var sortedFiles []string
	for p := range diff.Files {
		sortedFiles = append(sortedFiles, p)
	}
	sort.Strings(sortedFiles)

	for _, p := range sortedFiles {
		fd := diff.Files[p]
		var statusLabel string
		switch fd.ChangeType {
		case ChangeRenamed:
			statusLabel = fmt.Sprintf("renamed from %s (%.0f%% match)", fd.OldFilePath, fd.Similarity*100)
		case ChangeAdded:
			statusLabel = "new file"
		case ChangeRemoved:
			statusLabel = "deleted"
		case ChangeModified:
			statusLabel = "modified"
		}

		sb.WriteString(fmt.Sprintf("--- %s (%s)\n", p, statusLabel))
		for _, nd := range fd.NodeDiffs {
			switch nd.ChangeType {
			case ChangeAdded:
				sb.WriteString(fmt.Sprintf("  + %s\n", nd.Signature))
				if nd.NewNode != nil && nd.NewNode.Content != "" {
					lines := strings.Split(nd.NewNode.Content, "\n")
					for _, l := range lines {
						sb.WriteString(fmt.Sprintf("    + %s\n", l))
					}
				}
			case ChangeRemoved:
				sb.WriteString(fmt.Sprintf("  - %s\n", nd.Signature))
				if nd.OldNode != nil && nd.OldNode.Content != "" {
					lines := strings.Split(nd.OldNode.Content, "\n")
					for _, l := range lines {
						sb.WriteString(fmt.Sprintf("    - %s\n", l))
					}
				}
			case ChangeModified:
				sb.WriteString(fmt.Sprintf("  ~ %s\n", nd.Signature))
				if nd.OldNode != nil && nd.NewNode != nil {
					oldLines := strings.Split(nd.OldNode.Content, "\n")
					newLines := strings.Split(nd.NewNode.Content, "\n")
					lineChanges := DiffLines(oldLines, newLines)
					for _, lc := range lineChanges {
						switch lc.Type {
						case ChangeAdded:
							sb.WriteString(fmt.Sprintf("    + %s\n", lc.Text))
						case ChangeRemoved:
							sb.WriteString(fmt.Sprintf("    - %s\n", lc.Text))
						case ChangeUnchanged:
							sb.WriteString(fmt.Sprintf("      %s\n", lc.Text))
						}
					}
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// LineChange represents a single line delta within a modified node.
type LineChange struct {
	Type ChangeType
	Text string
}

// DiffLines computes the line-by-line diff between two sets of text lines using LCS.
func DiffLines(oldLines, newLines []string) []LineChange {
	n := len(oldLines)
	m := len(newLines)

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			if oldLines[i-1] == newLines[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	var changes []LineChange
	i, j := n, m
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && oldLines[i-1] == newLines[j-1] {
			changes = append(changes, LineChange{Type: ChangeUnchanged, Text: oldLines[i-1]})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			changes = append(changes, LineChange{Type: ChangeAdded, Text: newLines[j-1]})
			j--
		} else if i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]) {
			changes = append(changes, LineChange{Type: ChangeRemoved, Text: oldLines[i-1]})
			i--
		}
	}

	for l, r := 0, len(changes)-1; l < r; l, r = l+1, r-1 {
		changes[l], changes[r] = changes[r], changes[l]
	}

	return changes
}

// AppliedTransplant details a single transplant applied to the workspace.
type AppliedTransplant struct {
	FilePath   string
	NodeID     string
	Signature  string
	ChangeType ChangeType
}

// ApplyNodeDelta applies selected node transformations from a WorkspaceDiff onto the target workspace AST.
func ApplyNodeDelta(wsAST map[string]*FileAST, delta *WorkspaceDiff, targetFile, targetFunction string) ([]AppliedTransplant, error) {
	var applied []AppliedTransplant
	if delta == nil {
		return applied, nil
	}

	targetFile = strings.TrimSpace(targetFile)
	targetFunction = strings.TrimSpace(targetFunction)

	for filePath, fDiff := range delta.Files {
		if targetFile != "" && !strings.EqualFold(filePath, targetFile) && !strings.HasSuffix(filePath, targetFile) {
			continue
		}

		fAST, exists := wsAST[filePath]
		if !exists {
			fAST = &FileAST{
				FilePath: filePath,
				Nodes:    make(map[string]ASTNode),
			}
			wsAST[filePath] = fAST
		}

		for _, nd := range fDiff.NodeDiffs {
			if targetFunction != "" {
				nameMatch := strings.EqualFold(nd.NodeName, targetFunction)
				idMatch := strings.EqualFold(nd.NodeID, targetFunction) || strings.HasSuffix(nd.NodeID, ":"+targetFunction)
				sigMatch := strings.Contains(strings.ToLower(nd.Signature), strings.ToLower(targetFunction))
				if !nameMatch && !idMatch && !sigMatch {
					continue
				}
			}

			switch nd.ChangeType {
			case ChangeAdded, ChangeModified:
				if nd.NewNode != nil {
					fAST.Nodes[nd.NodeID] = *nd.NewNode
					applied = append(applied, AppliedTransplant{
						FilePath:   filePath,
						NodeID:     nd.NodeID,
						Signature:  nd.Signature,
						ChangeType: nd.ChangeType,
					})
				}
			case ChangeRemoved:
				if _, ok := fAST.Nodes[nd.NodeID]; ok {
					delete(fAST.Nodes, nd.NodeID)
					applied = append(applied, AppliedTransplant{
						FilePath:   filePath,
						NodeID:     nd.NodeID,
						Signature:  nd.Signature,
						ChangeType: nd.ChangeType,
					})
				}
			}
		}
	}

	return applied, nil
}
