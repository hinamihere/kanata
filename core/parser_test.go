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

func TestParseSource_C(t *testing.T) {
	cSrc := []byte(`#include <stdio.h>
#include <stdint.h>

#define BUFFER_SIZE 4096
#define MAX(a, b) (((a) > (b)) ? (a) : (b))

typedef struct ServerConfig {
    uint16_t port;
    const char *host;
} ServerConfig;

int start_server(ServerConfig *cfg) {
    printf("starting on port %d\n", cfg->port);
    return 0;
}
`)

	fAST, err := ParseSource("server.c", cSrc)
	if err != nil {
		t.Fatalf("ParseSource C failed: %v", err)
	}

	if fAST.Language != "c" {
		t.Errorf("expected language 'c', got '%s'", fAST.Language)
	}

	// Verify includes
	if _, ok := fAST.Nodes["include:<stdio.h>"]; !ok {
		t.Errorf("missing include:<stdio.h>")
	}
	if _, ok := fAST.Nodes["include:<stdint.h>"]; !ok {
		t.Errorf("missing include:<stdint.h>")
	}

	// Verify macros
	if macro, ok := fAST.Nodes["macro:BUFFER_SIZE"]; !ok || macro.Type != NodeMacro {
		t.Errorf("missing or invalid macro:BUFFER_SIZE")
	}
	if macro, ok := fAST.Nodes["macro:MAX"]; !ok || macro.Type != NodeMacro {
		t.Errorf("missing or invalid macro:MAX")
	}

	// Verify struct / type
	if typeNode, ok := fAST.Nodes["type:struct ServerConfig"]; !ok || typeNode.Type != NodeTypeDecl {
		t.Errorf("missing or invalid type:struct ServerConfig")
	}

	// Verify function
	if fnNode, ok := fAST.Nodes["function:func start_server"]; !ok || fnNode.Type != NodeFunction {
		t.Errorf("missing or invalid function:func start_server")
	}
}

func TestParseSource_Python(t *testing.T) {
	pySrc := []byte(`import os
from dataclasses import dataclass

@dataclass
class UserProfile:
    id: str
    username: str

def compute_hash(data: bytes) -> str:
    return "sample"

@app.route("/health")
async def health_check():
    return {"status": "ok"}
`)

	fAST, err := ParseSource("service.py", pySrc)
	if err != nil {
		t.Fatalf("ParseSource Python failed: %v", err)
	}

	if fAST.Language != "python" {
		t.Errorf("expected language 'python', got '%s'", fAST.Language)
	}

	if _, ok := fAST.Nodes["type:UserProfile"]; !ok {
		t.Errorf("missing type:UserProfile")
	}

	if fn, ok := fAST.Nodes["function:compute_hash"]; !ok || fn.Type != NodeFunction {
		t.Errorf("missing function:compute_hash")
	}

	if fn, ok := fAST.Nodes["function:health_check"]; !ok || fn.Type != NodeFunction {
		t.Errorf("missing function:health_check")
	}
}

func TestParseSource_Rust(t *testing.T) {
	rsSrc := []byte(`use std::collections::HashMap;

pub struct Database {
    records: HashMap<String, String>,
}

pub trait StorageDriver {
    fn read(&self, key: &str) -> Option<String>;
}

impl StorageDriver for Database {
    fn read(&self, key: &str) -> Option<String> {
        self.records.get(key).cloned()
    }
}

macro_rules! log_info {
    ($msg:expr) => {
        println!("[INFO] {}", $msg);
    };
}

pub fn initialize_db() -> Database {
    Database { records: HashMap::new() }
}
`)

	fAST, err := ParseSource("lib.rs", rsSrc)
	if err != nil {
		t.Fatalf("ParseSource Rust failed: %v", err)
	}

	if fAST.Language != "rust" {
		t.Errorf("expected language 'rust', got '%s'", fAST.Language)
	}

	if _, ok := fAST.Nodes["use:std::collections::HashMap"]; !ok {
		t.Errorf("missing use:std::collections::HashMap")
	}

	if st, ok := fAST.Nodes["type:Database"]; !ok || st.Type != NodeTypeDecl {
		t.Errorf("missing type:Database")
	}

	if tr, ok := fAST.Nodes["trait:StorageDriver"]; !ok || tr.Type != NodeTrait {
		t.Errorf("missing trait:StorageDriver")
	}

	if im, ok := fAST.Nodes["impl:StorageDriver for Database"]; !ok || im.Type != NodeImpl {
		t.Errorf("missing impl:StorageDriver for Database")
	}

	if mc, ok := fAST.Nodes["macro:log_info"]; !ok || mc.Type != NodeMacro {
		t.Errorf("missing macro:log_info")
	}

	if fnNode, ok := fAST.Nodes["function:initialize_db"]; !ok || fnNode.Type != NodeFunction {
		t.Errorf("missing function:initialize_db")
	}
}
