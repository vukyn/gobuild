# gobuild templates

This directory holds the project scaffolding templates, embedded into the
`gobuild` binary at build time via `//go:embed all:templates` (see `embed.go`).
Each subdirectory is a **preset** — a self-contained, full file set rendered
when the user selects it (`--http-template <preset>`, default `base`).

## Layout

```
templates/
  base/     # plain "hello world" Go project (default preset)
  fiber/    # Fiber v3 HTTP server with /health + graceful shutdown
```

Presets are self-contained: each folder carries its own copy of every file it
generates (including static ones like gitignore/todo). There is no layering or
overlay between presets — minor duplication of static files is accepted in
exchange for simplicity.

## Conventions

### `.tmpl` suffix

Every template file ends in `.tmpl`. At write time the suffix is stripped and
the file is rendered with Go's `text/template` using `missingkey=error`, so a
typo'd placeholder fails loudly instead of silently emitting `<no value>`.

Available template fields (see `templateData` in `render.go`):

- `{{.ProjectName}}` — the project / module name
- `{{.GoVersion}}` — the Go version (`--go`, defaults to the local toolchain)
- `{{.Preset}}` — the selected preset name

### Dotfile mapping

Dotfiles are stored **without** a leading dot so they are not hidden in this
repository. The dot is re-added at write time via an explicit lookup table in
`render.go` (`dotfileNames`):

| template name   | written as   |
| --------------- | ------------ |
| `env.tmpl`      | `.env`       |
| `gitignore.tmpl`| `.gitignore` |

To add another dotfile, drop the template (without dot) and add a row to
`dotfileNames`.

### `.raw.tmpl` double suffix

A file that must contain a literal `{{` (which `text/template` would otherwise
try to interpret) can use the `.raw.tmpl` double suffix. `outputName` strips
both suffixes. Note: such files are still executed as templates, so escape
literal braces with ``{{"{{"}}``. Neither `base` nor `fiber` needs this today;
the convention exists for future presets.

### Nested directories

Directory structure inside a preset is reproduced verbatim under the generated
project (e.g. `fiber/internal/handler/health.go.tmpl` →
`<project>/internal/handler/health.go`). Intermediate directories are created
automatically.

## Adding things

- **Add a file to a preset**: drop a new `<name>.tmpl` (or `<name>.raw.tmpl`)
  into the preset folder. No code changes needed — the walker picks it up.
- **Add a preset**: create a new subdirectory under `templates/` with the full
  file set. Wire its name into the `--http-template` usage string in `main.go`
  and add golden coverage in `gobuild_test.go` (`-update` regenerates goldens).
