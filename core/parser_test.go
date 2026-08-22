package core

import (
	"testing"
)

func TestParseSource_Go(t *testing.T) {
	src := []byte(`package sample

import "fmt"

const AppVersion = "1.0.0"

var GlobalCounter = 0

type Config struct {
	Port int
	Host string
}

// CalculateSum computes sum of two integers.
func CalculateSum(a int, b int) int {
	return a + b
}

func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
`)

	fAST, err := ParseSource("sample.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}

	if fAST.Language != "go" {
		t.Errorf("expected language 'go', got '%s'", fAST.Language)
	}

	if fAST.RawHash == "" {
		t.Errorf("expected non-empty RawHash")
	}

	// Verify nodes
	expectedNodes := []string{
		"pkg:sample",
		"const:AppVersion",
		"var:GlobalCounter",
		"type:Config",
		"func:CalculateSum",
		"func:(*Config) Address",
	}

	for _, expectedID := range expectedNodes {
		node, ok := fAST.Nodes[expectedID]
		if !ok {
			t.Errorf("missing expected node: %s", expectedID)
			continue
		}
		if node.Hash == "" {
			t.Errorf("node %s has empty hash", expectedID)
		}
		if node.Signature == "" {
			t.Errorf("node %s has empty signature", expectedID)
		}
	}

	fnNode := fAST.Nodes["func:CalculateSum"]
	if fnNode.Type != NodeFunction {
		t.Errorf("expected NodeFunction, got %s", fnNode.Type)
	}
	if fnNode.DocComment != "CalculateSum computes sum of two integers." {
		t.Errorf("unexpected doc comment: %q", fnNode.DocComment)
	}
}
