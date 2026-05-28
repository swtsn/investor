# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Go TUI application to track investment diversification and allocation.

## Working with Claude

- **CONTEXT.md** (repo root) — living domain model, glossary, and core rules; read it at the start of any significant task
- **docs/adr/** — architecture decision records (ADR-001 through ADR-004 established); create a new one when a non-obvious architectural choice is made
- **Plans** — stored at `~/.claude/projects/-repos-investor/plans/`, never repo-local

## Commands

```bash
go build ./...              # build all
go test ./...               # run all tests
go test ./internal/foo/...  # run tests in a specific package
go vet ./...                # lint
```

Add a Makefile as the project grows.

## Architecture

TUI stack: **bubbletea** (framework) + **lipgloss** (styling) + **bubbles** (components), following the pattern in `/repos/trade-tracker-go`.

Structure is being established — update this section as decisions are made and recorded in `docs/adr/`.
