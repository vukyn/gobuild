package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// templateData holds the values substituted into every template during
// rendering. Fields are exported so text/template actions can reach them.
type templateData struct {
	ProjectName string
	GoVersion   string
	Preset      string
	ModulePath  string
	Author      string
	Year        string
}

// dotfileNames maps a template base name (without the leading dot) to the
// dotted file name it should be written as. Templates live on disk without a
// leading dot so they are not hidden in the repository.
var dotfileNames = map[string]string{
	"env":       ".env",
	"gitignore": ".gitignore",
}

// outputName converts a template-relative path into the final on-disk name:
// it strips the ".tmpl" suffix (and the ".raw.tmpl" double suffix used for
// templates that must contain literal "{{" sequences) and remaps known
// dotfiles via dotfileNames.
func outputName(rel string) string {
	name := rel
	switch {
	case strings.HasSuffix(name, ".raw.tmpl"):
		name = strings.TrimSuffix(name, ".raw.tmpl")
	case strings.HasSuffix(name, ".tmpl"):
		name = strings.TrimSuffix(name, ".tmpl")
	}

	dir, base := filepath.Split(name)
	if mapped, ok := dotfileNames[base]; ok {
		base = mapped
	}
	return filepath.Join(dir, base)
}

// renderPreset walks the embedded template tree for the named preset and
// writes each rendered file into destDir, recreating any nested directory
// structure. An unknown preset yields a non-nil error.
func renderPreset(preset string, data templateData, destDir string) error {
	presetRoot := "templates/" + preset

	// Confirm the preset directory exists before walking it.
	if _, err := fs.Stat(templatesFS, presetRoot); err != nil {
		return fmt.Errorf("unknown preset %q", preset)
	}

	return fs.WalkDir(templatesFS, presetRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == presetRoot {
			return nil
		}

		// rel is the path inside the preset (e.g. "internal/handler/health.go.tmpl").
		rel := strings.TrimPrefix(path, presetRoot+"/")

		if entry.IsDir() {
			dest := filepath.Join(destDir, outputName(rel))
			if err := os.MkdirAll(dest, 0755); err != nil { // #nosec G301 -- scaffolded project dir must be user-browsable
				return fmt.Errorf("failed to create directory %s: %w", dest, err)
			}
			return nil
		}

		contents, err := fs.ReadFile(templatesFS, path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		tmpl, err := template.New(path).Option("missingkey=error").Parse(string(contents))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		var rendered bytes.Buffer
		if err := tmpl.Execute(&rendered, data); err != nil {
			return fmt.Errorf("failed to render template %s: %w", path, err)
		}

		dest := filepath.Join(destDir, outputName(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil { // #nosec G301 -- scaffolded project dir must be user-browsable
			return fmt.Errorf("failed to create directory for %s: %w", dest, err)
		}
		if err := os.WriteFile(dest, rendered.Bytes(), 0644); err != nil { // #nosec G306 -- generated source files use conventional 0644, no secrets
			return fmt.Errorf("failed to create %s: %w", dest, err)
		}
		return nil
	})
}
