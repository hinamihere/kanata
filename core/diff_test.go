package core

import (
	"testing"
)

func TestDiffFiles(t *testing.T) {
	v1Src := []byte(`package main

func StableFunc() int {
	return 42
}

func WillModify() string {
	return "version1"
}

func WillDelete() bool {
	return true
}
`)

	v2Src := []byte(`package main

func StableFunc() int {
	return 42
}

func WillModify() string {
	return "version2-modified"
}

func BrandNewFunc() {
	println("hello")
}
`)

	ast1, err := ParseSource("main.go", v1Src)
	if err != nil {
		t.Fatalf("Parse v1 failed: %v", err)
	}

	ast2, err := ParseSource("main.go", v2Src)
	if err != nil {
		t.Fatalf("Parse v2 failed: %v", err)
	}

	diff := DiffFiles(ast1, ast2)
	if diff.ChangeType != ChangeModified {
		t.Fatalf("expected ChangeModified, got %s", diff.ChangeType)
	}

	var added, removed, modified int
	for _, nd := range diff.NodeDiffs {
		switch nd.ChangeType {
		case ChangeAdded:
			added++
			if nd.NodeName != "BrandNewFunc" {
				t.Errorf("unexpected added node: %s", nd.NodeName)
			}
		case ChangeRemoved:
			removed++
			if nd.NodeName != "WillDelete" {
				t.Errorf("unexpected removed node: %s", nd.NodeName)
			}
		case ChangeModified:
			modified++
			if nd.NodeName != "WillModify" {
				t.Errorf("unexpected modified node: %s", nd.NodeName)
			}
		}
	}

	if added != 1 || removed != 1 || modified != 1 {
		t.Errorf("expected 1 added, 1 removed, 1 modified, got %d added, %d removed, %d modified", added, removed, modified)
	}
}

func TestIntegrateWorkspaces_CleanAndConflict(t *testing.T) {
	baseSrc := []byte(`package app

func SharedA() string {
	return "base-a"
}

func SharedB() string {
	return "base-b"
}
`)

	// Stream 1 modifies SharedA
	s1Src := []byte(`package app

func SharedA() string {
	return "stream1-a"
}

func SharedB() string {
	return "base-b"
}
`)

	// Stream 2 adds FuncC, modifies SharedB, but leaves SharedA alone -> Clean merge!
	s2CleanSrc := []byte(`package app

func SharedA() string {
	return "base-a"
}

func SharedB() string {
	return "stream2-b"
}

func FuncC() bool {
	return true
}
`)

	baseAST, _ := ParseSource("app.go", baseSrc)
	s1AST, _ := ParseSource("app.go", s1Src)
	s2CleanAST, _ := ParseSource("app.go", s2CleanSrc)

	baseMap := map[string]*FileAST{"app.go": baseAST}
	s1Map := map[string]*FileAST{"app.go": s1AST}
	s2CleanMap := map[string]*FileAST{"app.go": s2CleanAST}

	cleanMerge := IntegrateWorkspaces(baseMap, s1Map, s2CleanMap)
	if cleanMerge.HasConflict {
		t.Fatalf("expected clean merge, got %d conflicts", len(cleanMerge.Conflicts))
	}

	mergedApp := cleanMerge.MergedState["app.go"]
	if mergedApp == nil {
		t.Fatalf("merged app.go is nil")
	}

	// Verify that both Stream 1's change to SharedA and Stream 2's change to SharedB and FuncC exist!
	nodeA := mergedApp.Nodes["func:SharedA"]
	if nodeA.Content != NormalizeCode(`func SharedA() string {
	return "stream1-a"
}`) {
		t.Errorf("unexpected content for SharedA: %s", nodeA.Content)
	}

	nodeB := mergedApp.Nodes["func:SharedB"]
	if nodeB.Content != NormalizeCode(`func SharedB() string {
	return "stream2-b"
}`) {
		t.Errorf("unexpected content for SharedB: %s", nodeB.Content)
	}

	if _, ok := mergedApp.Nodes["func:FuncC"]; !ok {
		t.Errorf("missing merged FuncC")
	}

	// Now test Conflict case: Stream 3 ALSO modifies SharedA differently
	s3ConflictSrc := []byte(`package app

func SharedA() string {
	return "conflict-from-s3"
}

func SharedB() string {
	return "base-b"
}
`)
	s3AST, _ := ParseSource("app.go", s3ConflictSrc)
	s3Map := map[string]*FileAST{"app.go": s3AST}

	conflictMerge := IntegrateWorkspaces(baseMap, s1Map, s3Map)
	if !conflictMerge.HasConflict {
		t.Fatalf("expected conflict on SharedA, but got none")
	}
	if len(conflictMerge.Conflicts) != 1 {
		t.Fatalf("expected exactly 1 conflict, got %d", len(conflictMerge.Conflicts))
	}
	if conflictMerge.Conflicts[0].NodeID != "func:SharedA" {
		t.Errorf("expected conflict on func:SharedA, got %s", conflictMerge.Conflicts[0].NodeID)
	}
}
