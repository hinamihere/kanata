package core

import (
	"strings"
	"testing"
)

func TestInferIntent_SingleFunctionAdded(t *testing.T) {
	v1 := []byte(`package main
func Existing() {}
`)
	v2 := []byte(`package main
func Existing() {}
func NewFeature() {}
`)

	ast1, _ := ParseSource("main.go", v1)
	ast2, _ := ParseSource("main.go", v2)

	diff := DiffWorkspace(map[string]*FileAST{"main.go": ast1}, map[string]*FileAST{"main.go": ast2})
	intent := InferIntent(diff)

	if !strings.Contains(intent, "Add NewFeature function") {
		t.Errorf("expected intent to mention adding NewFeature, got: %s", intent)
	}
}

func TestInferIntent_FunctionModifiedAndStructAdded(t *testing.T) {
	v1 := []byte(`package main
func Process() string {
	return "v1"
}
`)
	v2 := []byte(`package main

type Config struct {
	Port int
}

func Process() string {
	return "v2"
}
`)

	ast1, _ := ParseSource("main.go", v1)
	ast2, _ := ParseSource("main.go", v2)

	diff := DiffWorkspace(map[string]*FileAST{"main.go": ast1}, map[string]*FileAST{"main.go": ast2})
	intent := InferIntent(diff)

	if !strings.Contains(intent, "Add Config type") || !strings.Contains(intent, "update Process function") {
		t.Errorf("expected combined intent for Config and Process, got: %s", intent)
	}
}
