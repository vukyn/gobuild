# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Purpose

`gobuild` is a small CLI (module `github.com/vukyn/gobuild`) that scaffolds new Go projects from templates. Given a project name (positional arg or `--name`/`-n`) and an optional `--go` version, it creates a directory containing `main.go`, `go.mod`, `.env`, `Makefile`, `README.md`, `.gitignore`, and `todo`, then runs `go mod tidy` and `git init` inside it.

This is a standalone tool — it does **not** follow the platform service template (no domains, no DI, no clean-architecture layers, no UI).

## Structure

```
main.go     # urfave/cli/v2 entrypoint + generateProject(): mkdir, render templates, write files, tidy + git init
tmpl/       # template string constants — one file per generated artifact
  tmpl.go        # shared placeholders: PROJECT_NAME ({{.ProjectName}}), GO_VERSION ({{.GoVersion}})
  t_go.mod.go    # GO_MOD
  t_main.go      # MAIN_GO
  t_env.go       # ENV
  t_gitignore.go # GIT_IGNORE
  t_makefile.go  # MAKEFILE
  t_readme.go    # README
  t_todo.go      # TODO
version/    # version.Current — the CLI's own version string
```

Templates are plain string constants; placeholders are substituted with `strings.ReplaceAll` (no `text/template` execution).

## Commands

```bash
make build      # go build -o bin/ ./$(PRJ)
make install    # go install ./$(PRJ)
make tag        # git tag -a v$(VERSION) + push (VERSION comes from .env)

go build ./...  # verify
go vet ./...
```

The Makefile sources `.env` (currently empty placeholders for `PRJ`/`VERSION`).

## Conventions

- **Adding a generated file**: create a new `tmpl/t_<name>.go` with a single string constant, use the shared placeholders from `tmpl/tmpl.go`, then wire it into the `files` map in `main.go`'s `generateProject`.
- New placeholders go in `tmpl/tmpl.go` and get their own `strings.ReplaceAll` line in the render loop.
- File permissions are deliberate for scaffolder output: `0755` dirs / `0644` files, annotated with `// #nosec` (G301/G306) — generated projects must be user-readable and contain no secrets. Keep those annotations when touching the write paths.
- Bump `version/version.go` when cutting a release; tag via `make tag`.
