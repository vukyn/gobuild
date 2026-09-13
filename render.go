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
	"env":       envFileName,
	"gitignore": ".gitignore",
}

const (
	// envFileName is the generated secrets file. It is singled out in the write
	// path: it receives live credentials as soon as a scaffolded project is
	// wired up, so it is written owner-only and an existing one is never
	// replaced.
	envFileName = ".env"

	// envFileMode keeps the generated .env unreadable by other local accounts.
	envFileMode = 0600

	// generatedFileMode is the conventional mode for every other generated
	// file — ordinary source that must stay user-readable.
	generatedFileMode = 0644
)

// fileMode picks the permission bits for a rendered file by its final name.
func fileMode(dest string) os.FileMode {
	if filepath.Base(dest) == envFileName {
		return envFileMode
	}
	return generatedFileMode
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

		dest := filepath.Join(destDir, outputName(rel))

		// An existing .env is never replaced, not even under --force: the
		// template value is boilerplate while the file on disk may already hold
		// real credentials.
		if filepath.Base(dest) == envFileName {
			if _, statErr := os.Stat(dest); statErr == nil {
				fmt.Printf("Preserving existing %s (not overwritten)\n", dest)
				return nil
			}
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

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil { // #nosec G301 -- scaffolded project dir must be user-browsable
			return fmt.Errorf("failed to create directory for %s: %w", dest, err)
		}
		// #nosec G306 -- fileMode narrows the generated .env to 0600; every
		// other generated file is ordinary source that must stay user-readable
		// at the conventional 0644.
		if err := os.WriteFile(dest, rendered.Bytes(), fileMode(dest)); err != nil {
			return fmt.Errorf("failed to create %s: %w", dest, err)
		}
		return nil
	})
}
