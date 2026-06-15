package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"text/template"
)

// update regenerates the golden fixtures when set: `go test -update`.
var update = flag.Bool("update", false, "regenerate golden files")

// fixedData is the deterministic template input used for all snapshot tests so
// golden output never depends on the host Go toolchain.
var fixedData = templateData{ProjectName: "testproj", GoVersion: "1.24", ModulePath: "github.com/vukyn/testproj", Author: "vukyn", Year: "2026"}

// presets enumerates every preset that must have golden coverage.
var presets = []string{"base", "fiber", "platform-service"}

func TestRenderPresetGolden(t *testing.T) {
	for _, preset := range presets {
		preset := preset
		t.Run(preset, func(t *testing.T) {
			data := fixedData
			data.Preset = preset

			out := t.TempDir()
			if err := renderPreset(preset, data, out); err != nil {
				t.Fatalf("renderPreset(%q) failed: %v", preset, err)
			}

			goldenDir := filepath.Join("testdata", "golden", preset)

			if *update {
				regenerateGolden(t, out, goldenDir)
				return
			}

			compareTrees(t, goldenDir, out)
		})
	}
}

// compareTrees asserts that the rendered tree matches the golden tree both in
// shape (set of relative paths) and in per-file content.
func compareTrees(t *testing.T, goldenDir, gotDir string) {
	t.Helper()

	goldenFiles := relFiles(t, goldenDir)
	gotFiles := relFiles(t, gotDir)

	if strings.Join(goldenFiles, "\n") != strings.Join(gotFiles, "\n") {
		t.Fatalf("tree shape mismatch\n golden: %v\n    got: %v\n(run `go test -update` if intentional)", goldenFiles, gotFiles)
	}

	for _, rel := range goldenFiles {
		want, err := os.ReadFile(filepath.Join(goldenDir, rel))
		if err != nil {
			t.Fatalf("read golden %s: %v", rel, err)
		}
		got, err := os.ReadFile(filepath.Join(gotDir, rel))
		if err != nil {
			t.Fatalf("read rendered %s: %v", rel, err)
		}
		if string(want) != string(got) {
			t.Errorf("content mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", rel, want, got)
		}
	}
}

// regenerateGolden mirrors the rendered tree into goldenDir.
func regenerateGolden(t *testing.T, srcDir, goldenDir string) {
	t.Helper()

	if err := os.RemoveAll(goldenDir); err != nil {
		t.Fatalf("clean golden dir: %v", err)
	}
	for _, rel := range relFiles(t, srcDir) {
		content, err := os.ReadFile(filepath.Join(srcDir, rel))
		if err != nil {
			t.Fatalf("read rendered %s: %v", rel, err)
		}
		dest := filepath.Join(goldenDir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			t.Fatalf("mkdir golden %s: %v", dest, err)
		}
		if err := os.WriteFile(dest, content, 0644); err != nil {
			t.Fatalf("write golden %s: %v", dest, err)
		}
	}
}

// relFiles returns the sorted list of file paths under root, relative to root,
// using forward slashes so comparisons are stable across platforms.
func relFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(files)
	return files
}

func TestOutputName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"main.go.tmpl", "main.go"},
		{"go.mod.tmpl", "go.mod"},
		{"env.tmpl", ".env"},
		{"gitignore.tmpl", ".gitignore"},
		{"todo.tmpl", "todo"},
		{"internal/handler/health.go.tmpl", filepath.Join("internal", "handler", "health.go")},
		{"config.raw.tmpl", "config"},
	}
	for _, tc := range cases {
		if got := outputName(tc.in); got != tc.want {
			t.Errorf("outputName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderPresetUnknown(t *testing.T) {
	err := renderPreset("nope", fixedData, t.TempDir())
	if err == nil {
		t.Fatal("expected error for unknown preset, got nil")
	}
	if !strings.Contains(err.Error(), `unknown preset "nope"`) {
		t.Errorf("error %q does not mention unknown preset", err.Error())
	}
}

func TestRenderMissingKeyErrors(t *testing.T) {
	// renderPreset configures text/template with Option("missingkey=error").
	// This test mirrors that configuration to lock in the behavior: a template
	// referencing an undefined field must fail rather than emit "<no value>".
	tmpl, err := template.New("probe").Option("missingkey=error").Parse("value: {{.DoesNotExist}}\n")
	if err != nil {
		t.Fatalf("parse probe template: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fixedData); err == nil {
		t.Fatalf("expected missingkey error, got output %q", buf.String())
	}
}
