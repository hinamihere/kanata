# Contributing to Kanata

Thank you for your interest in contributing to Kanata. This is an open-source project, and contributions are welcome from anyone interested in advancing semantic version control.

---

## AI and LLM Assisted Contributions

Using AI assistance to write parsers, optimize algorithms, draft tests, or improve documentation is welcome. Good code is good code regardless of the tools used to produce it.

However, if you submit AI-assisted code, please adhere to the following standards:

1. **Full Ownership:** You are expected to understand the code you submit. You should be able to explain the architectural choices, AST logic, and edge-case handling during code review.
2. **Verification:** Do not blindly copy-paste generated code. Ensure that the logic is sound, uses appropriate standard library primitives, and does not introduce memory or performance regressions.
3. **Passing Tests:** All existing and new tests must pass locally (`go test ./...`) before opening a pull request.

---

## How to Contribute

1. Fork the repository and clone your fork locally.
2. Create a dedicated branch for your feature or bug fix:
   `git checkout -b feature/my-new-feature`
3. Implement your changes following idiomatic Go patterns and the existing repository structure.
4. Ensure all tests pass:
   `go test -v ./...`
5. Commit your changes with clear intent messages.
6. Open a Pull Request describing the architectural changes and the problem being solved.

---

## Areas for Contribution

If you are looking for places to contribute, the following areas are open for expansion:

* **Language Parsers:** Implementing native AST parsers for additional languages (e.g., TypeScript, Java, C#).
* **Editor Integrations:** Building plugins and extensions for Neovim, Doom Emacs, or VS Code to surface inline semantic diffs and blame.
* **Storage & Performance:** Profiling and optimizing SQLite queries, node hashing, and snapshot materialization.
* **Documentation & Examples:** Expanding user guides, workflows, and edge-case documentation.

Draft Pull Requests are welcome if you want early feedback on an architectural proposal before completing the implementation.
