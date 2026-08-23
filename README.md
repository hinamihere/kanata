<div align="center">

# Kanata (彼方)—a semantic version control system

<img title="Kanata Logo" src="https://github.com/user-attachments/assets/3be94b1d-a6bd-422e-8cb1-5f71d829d46d" width="320" height="320">


[![Release](https://img.shields.io/github/v/release/hinamihere/kanata)](https://github.com/hinamihere/kanata/releases)
[![Build Status](https://img.shields.io/github/actions/workflow/status/hinamihere/kanata/ci.yml?branch=main)](https://github.com/hinamihere/kanata/actions)
<br/>
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Pure Go](https://img.shields.io/badge/language-pure_go-00add8.svg)](#)

**[Installation](#getting-started) &nbsp;&nbsp;&bull;&nbsp;&nbsp;**
**[Documentation](#features) &nbsp;&nbsp;&bull;&nbsp;&nbsp;**
**[Contributing](#contributing)**

</div>

## Introduction

> Kanata (彼方, meaning "beyond" or "the other side") is a next-generation, AST-based version control system built entirely in Go. You use it to track structural, intent-driven changes to your codebase, completely bypassing the friction of arbitrary line-by-line text conflicts. 

Legacy version control systems track raw lines of text. If you move a massive `struct` defining a 128-bit bus from one file to another, legacy tools register a deletion and an addition, destroying the evolutionary history of your architecture. **Kanata understands your code.** It parses source files into an Abstract Syntax Tree (AST) and tracks structural nodes—like Functions, Structs, Macros, and Traits—rather than textual lines.

---

## Getting Started

> **Important:** Kanata is currently a **v0 production foundation**. While it is fully self-hosted and stable for C, C++, Rust, Go, and Python architectures, it operates on a paradigm entirely distinct from text-based VCS. 

Kanata is a pure Go static binary with zero CGO dependencies. 

**1. Installation**
Download the latest binary for your system (`kana.exe` for Windows, `kana-linux` for Linux) from the Releases page. Place the binary in your system `PATH`.

**2. Global Configuration**
Configure your global identity (saved to `~/.kanaconfig`):
*   `kana config --global user.name "Your Name"`
*   `kana config --global user.email "you@example.com"`

**3. Initialization**
Initialize a new semantic repository inside your project:
*   `cd my-project`
*   `kana init`

---

## Multi-Language AST Engine

Kanata natively parses source files into structural nodes. When it encounters unsupported formats or assets (like `.png` or `.bin` ROM files), it automatically bypasses text parsing and stores them as byte-exact `NodeRawBlob` objects.

*   **C / C++**: `#define` (constants & macros), `#include`, `typedef struct / enum / union`, and functions.
*   **Go**: Functions, receiver methods, structs, interfaces, package clauses, and imports.
*   **Rust**: `fn`, `pub fn`, `struct`, `enum`, `trait`, `impl`, `macro_rules!`, and `use`.
*   **Python**: `def`, `async def`, `@decorators`, `class`, and imports.

---

## Workspace & Intent-Driven Snapshots

| Command | Description |
| :--- | :--- |
| `kana init` | Initialize an empty Kanata semantic repository (`.kana/`). |
| `kana status` | View uncommitted AST changes (`+added`, `~modified`, `-removed`, `renamed`). |
| `kana snapshot -i "<msg>"` | Record an atomic graph transformation with a semantic intent message. |
| `kana snapshot -a` | **Auto-Infer Intent:** Automatically generate an intent description from mutations. |
| `kana snapshot -p` | **Interactive Staging:** Interactively stage specific functions/structs node-by-node. |
| `kana rewind <hash>` | Rewind the active workspace back to any historical snapshot. |
| `kana diff -p` | View detailed semantic patches with intra-function line-by-line diffs. |

---

## Work Streams & DAG Branching

| Command | Description |
| :--- | :--- |
| `kana focus <stream>` | Switch to a work stream or create a new branch, materializing files on disk. |
| `kana stream compare <A> <B>`| **Architectural Comparison:** Compare the AST architecture of two branches before merging. |
| `kana graph` | Render a visual ASCII DAG tree of all streams and commits in your terminal. |
| `kana log` | Chronological snapshot timeline (use `-g` for the inline DAG graph). |

---

## Semantic Integration & Cherry-Picking

| Command | Description |
| :--- | :--- |
| `kana integrate <stream>` | **3-Way AST Merge:** Reconciles non-overlapping function changes without line conflicts. |
| `kana pick <hash> -f <fn>` | **Semantic Cherry-Pick:** Surgically transplant an individual function into your workspace. |

---

## Named Park Shelves

| Command | Description |
| :--- | :--- |
| `kana park -n "<name>"` | Temporarily park in-flight workspace AST changes into a named shelf. |
| `kana park show <name>` | **AST Preview:** Inspect what functions are saved inside a shelf without restoring them. |
| `kana park restore <name>` | Pop and restore parked AST transformations into the active workspace. |

---

## Semantic Code Intelligence

| Command | Description |
| :--- | :--- |
| `kana find <query>` | **Semantic Search:** Search for functions, structs, types across snapshot history. |
| `kana find -t func <query>` | Filter search specifically by node type (`func`, `type`, `macro`, `const`, `var`). |
| `kana find -H <query>` | Print the complete multi-version evolution timeline of matching symbols. |
| `kana blame <file>` | **Semantic Blame:** Attributions based on AST node modifications, not line shifts. |

---

## Distributed Networking & Web Dashboard

| Command | Description |
| :--- | :--- |
| `kana clone <url>` | Clone repositories over HTTP/HTTPS, SSH, or Local Filesystems. |
| `kana push <remote> <stream>`| Push missing snapshot graph nodes and payloads. |
| `kana serve --port 3000` | Starts a dark-mode single-page Web Dashboard + HTTP sync server. |

---

## Status & Architecture

Kanata is currently a fully self-hosted pure Go architecture with zero CGO dependencies. It features an automated `.kanaignore` engine (with `.gitignore` fallback), and seamless AST rename & move detection that preserves node evolution and blame lineage across directory changes.

All development on Kanata is tracked, versioned, and branched using Kanata itself.

## Contributing

Kanata is an open-source project, and I welcome contributions from everyone! Whether you want to fix a bug, improve the documentation, add a new language parser, or just share an idea, I'd love your help. 

Please read the [CONTRIBUTING.md](CONTRIBUTING.md) file for details on the process for submitting pull requests, testing requirements, and guidelines on AI-assisted code.

Feel free to fork the repository, make your changes, and open a Pull Request. Don't hesitate to jump in—all contributions, big and small, are appreciated!## License

Kanata is available as Open Source Software, under the MIT license. See `LICENSE` for details.
