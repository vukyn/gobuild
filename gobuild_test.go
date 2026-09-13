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

	"github.com/urfave/cli/v2"
)

// update regenerates the golden fixtures when set: `go test -update`.
var update = flag.Bool("update", false, "regenerate golden files")

// fixedData is the deterministic template input used for all snapshot tests so
// golden output never depends on the host Go toolchain.
var fixedData = templateData{ProjectName: "testproj", GoVersion: "1.24", ModulePath: "github.com/vukyn/testproj", Author: "vukyn", Year: "2026"}

// presets enumerates every preset that must have golden coverage.
var presets = []string{"base", "fiber", "platform-service", "iot"}

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

func TestHasGoMod(t *testing.T) {
	dir := t.TempDir()
	if hasGoMod(dir) {
		t.Fatal("hasGoMod(empty dir) = true, want false")
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if !hasGoMod(dir) {
		t.Fatal("hasGoMod(dir with go.mod) = false, want true")
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

// ---------------------------------------------------------------------------
// Input validation
//
// Every value below reaches a generated file through text/template, which does
// no escaping. The rejection tests are the security boundary; they must stay
// rejections rather than becoming sanitizations.
// ---------------------------------------------------------------------------

// moduleInjectionPayload is the module path that made the generated go.mod carry
// an attacker-controlled `require` directive, which the automatic `go mod tidy`
// then resolved.
const moduleInjectionPayload = "evil.com/x\"\nrequire github.com/attacker/backdoor v1.0.0\n// "

// nameInjectionPayload is the project name that produced *valid* JSON in
// ui/package.json with an extra "overrides" key, substituting react with an
// attacker package at npm install time.
const nameInjectionPayload = `svc", "overrides": {"react": "npm:evil-pkg@1.0.0"}, "y": "`

// safeModulePath is a known-good --module value, used so a rejection test
// exercises the check it names rather than tripping an earlier one.
const safeModulePath = "github.com/vukyn/safe"

func TestValidateProjectName(t *testing.T) {
	valid := []string{"svc", "my-service", "my_service", "my.service", "a", "A1", "0abc",
		strings.Repeat("a", 64)}
	for _, name := range valid {
		if err := validateProjectName(name); err != nil {
			t.Errorf("validateProjectName(%q) = %v, want nil", name, err)
		}
	}

	invalid := map[string]string{
		"empty":            "",
		"json injection":   nameInjectionPayload,
		"traversal":        "../../escaped",
		"parent":           "..",
		"absolute":         "/tmp/x",
		"separator":        "a/b",
		"windows sep":      `a\b`,
		"leading dot":      ".hidden",
		"leading dash":     "-svc",
		"space":            "my svc",
		"newline":          "svc\nrm -rf /",
		"quote":            `svc"`,
		"dollar":           "svc$(id)",
		"too long":         strings.Repeat("a", 65),
		"only dot":         ".",
		"trailing sep":     "svc/",
		"nested traversal": "a/../../b",
	}
	for label, name := range invalid {
		if err := validateProjectName(name); err == nil {
			t.Errorf("validateProjectName(%q) [%s] = nil, want error", name, label)
		}
	}
}

func TestValidateModulePath(t *testing.T) {
	for _, path := range []string{"github.com/vukyn/svc", "example.com/a/b/c", "github.com/vukyn/My_Svc"} {
		if err := validateModulePath(path); err != nil {
			t.Errorf("validateModulePath(%q) = %v, want nil", path, err)
		}
	}

	invalid := map[string]string{
		"empty":             "",
		"require injection": moduleInjectionPayload,
		"newline only":      "evil.com/x\nrequire foo v1",
		"space":             "evil.com/a b",
		"no dot in host":    "svc",
		"quote":             `evil.com/x"`,
	}
	for label, path := range invalid {
		if err := validateModulePath(path); err == nil {
			t.Errorf("validateModulePath(%q) [%s] = nil, want error", path, label)
		}
	}
}

func TestValidateGoVersion(t *testing.T) {
	for _, version := range []string{"1.27", "1.27.1", "2.0", "1.24.0"} {
		if err := validateGoVersion(version); err != nil {
			t.Errorf("validateGoVersion(%q) = %v, want nil", version, err)
		}
	}

	invalid := map[string]string{
		"empty":         "",
		"major only":    "1",
		"prefixed":      "go1.27",
		"four parts":    "1.27.1.2",
		"non numeric":   "1.x",
		"rc":            "1.27rc1",
		"directive inj": "1.24\nrequire github.com/attacker/backdoor v1.0.0",
	}
	for label, version := range invalid {
		if err := validateGoVersion(version); err == nil {
			t.Errorf("validateGoVersion(%q) [%s] = nil, want error", version, label)
		}
	}
}

// TestGenerateProjectRejectsUnsafeInput asserts the validation runs before any
// filesystem work: a rejected invocation must leave nothing behind, inside the
// working directory or above it.
func TestGenerateProjectRejectsUnsafeInput(t *testing.T) {
	cases := []struct {
		label       string
		projectName string
		goVersion   string
		modulePath  string
	}{
		// The name cases pass an explicitly valid --module. Without it the
		// default module path is derived from the name, so module.CheckPath
		// rejects the invocation first and the name check goes unexercised —
		// the test would then pass even with name validation removed.
		{"json injection in name", nameInjectionPayload, "1.27", safeModulePath},
		{"path traversal in name", "../../escaped", "1.27", safeModulePath},
		{"absolute path in name", "/tmp/gobuild-escape-probe", "1.27", safeModulePath},
		{"empty name", "", "1.27", safeModulePath},
		{"require injection in module", "svc", "1.27", moduleInjectionPayload},
		{"directive injection in go version", "svc", "1.24\nrequire github.com/attacker/backdoor v1.0.0", safeModulePath},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			// The working directory sits two levels below the observed root so
			// that "../.." — the traversal the scanner demonstrated — lands
			// inside the window this test inspects. A shallower layout lets an
			// escape land above the assertion and go unnoticed.
			root := t.TempDir()
			work := filepath.Join(root, "outer", "work")
			if err := os.MkdirAll(work, 0o755); err != nil {
				t.Fatalf("mkdir work: %v", err)
			}
			t.Chdir(work)

			err := generateProject(tc.projectName, tc.goVersion, "base", tc.modulePath, false)
			if err == nil {
				t.Fatalf("generateProject(%q, %q, base, %q) = nil, want error", tc.projectName, tc.goVersion, tc.modulePath)
			}
			if got := relFiles(t, root); len(got) != 0 {
				t.Errorf("rejected invocation wrote files: %v", got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Filesystem safety
// ---------------------------------------------------------------------------

func TestGenerateProjectRefusesNonEmptyDirectory(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)

	existing := filepath.Join(work, "victim")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("mkdir victim: %v", err)
	}
	secret := "DB_PASSWORD=hunter2\n"
	if err := os.WriteFile(filepath.Join(existing, ".env"), []byte(secret), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	source := "package main // REAL CODE\n"
	if err := os.WriteFile(filepath.Join(existing, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	err := generateProject("victim", "1.27", "base", "", false)
	if err == nil {
		t.Fatal("generateProject into non-empty dir = nil, want error")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error %q does not point at --force", err.Error())
	}

	got, readErr := os.ReadFile(filepath.Join(existing, ".env"))
	if readErr != nil {
		t.Fatalf("read .env: %v", readErr)
	}
	if string(got) != secret {
		t.Errorf(".env was modified: %q", got)
	}
	got, readErr = os.ReadFile(filepath.Join(existing, "main.go"))
	if readErr != nil {
		t.Fatalf("read main.go: %v", readErr)
	}
	if string(got) != source {
		t.Errorf("main.go was modified: %q", got)
	}
}

// TestRenderPresetPreservesExistingEnv pins the rule that survives --force: a
// .env already on disk may hold live credentials and is never replaced by the
// template's placeholder version.
func TestRenderPresetPreservesExistingEnv(t *testing.T) {
	dest := t.TempDir()
	secret := "DB_PASSWORD=hunter2\n"
	if err := os.WriteFile(filepath.Join(dest, ".env"), []byte(secret), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	data := fixedData
	data.Preset = "base"
	if err := renderPreset("base", data, dest); err != nil {
		t.Fatalf("renderPreset: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, ".env"))
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if string(got) != secret {
		t.Errorf("existing .env was overwritten: %q", got)
	}
	// The rest of the tree still rendered.
	if _, err := os.Stat(filepath.Join(dest, "main.go")); err != nil {
		t.Errorf("main.go not rendered alongside preserved .env: %v", err)
	}
}

// TestRenderPresetFileModes asserts the generated .env is owner-only while
// ordinary generated source keeps the conventional 0644.
func TestRenderPresetFileModes(t *testing.T) {
	for _, preset := range []string{"base", "fiber", "platform-service"} {
		t.Run(preset, func(t *testing.T) {
			dest := t.TempDir()
			data := fixedData
			data.Preset = preset
			if err := renderPreset(preset, data, dest); err != nil {
				t.Fatalf("renderPreset(%q): %v", preset, err)
			}

			sawEnv := false
			for _, rel := range relFiles(t, dest) {
				info, err := os.Stat(filepath.Join(dest, rel))
				if err != nil {
					t.Fatalf("stat %s: %v", rel, err)
				}
				want := os.FileMode(generatedFileMode)
				if filepath.Base(rel) == envFileName {
					sawEnv = true
					want = os.FileMode(envFileMode)
				}
				if got := info.Mode().Perm(); got != want {
					t.Errorf("%s mode = %04o, want %04o", rel, got, want)
				}
			}
			if !sawEnv {
				t.Fatalf("preset %q rendered no %s — test would pass vacuously", preset, envFileName)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// projectDir consistency (render target == post-setup target)
// ---------------------------------------------------------------------------

// TestGenerateProjectPostSetupUsesRenderedDir uses the iot preset because it
// renders no go.mod, so the run stays offline: `go mod tidy` is skipped and only
// the local `git init` runs.
func TestGenerateProjectPostSetupUsesRenderedDir(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)

	if err := generateProject("postproj", "1.27", "iot", "", false); err != nil {
		t.Fatalf("generateProject: %v", err)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	projectDir := filepath.Join(currentDir, "postproj")

	if _, err := os.Stat(filepath.Join(projectDir, "platformio.ini")); err != nil {
		t.Fatalf("preset did not render into %s: %v", projectDir, err)
	}
	// git init must have targeted the same directory the render wrote to.
	info, err := os.Stat(filepath.Join(projectDir, ".git"))
	if err != nil {
		t.Fatalf("git init did not run in the rendered directory %s: %v", projectDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s/.git is not a directory", projectDir)
	}
}

// TestGenerateProjectFailsWhenPostSetupFails pins that a failed post-setup step
// is an error rather than a warning printed above a success line. `git init`
// is made to fail by leaving a plain file where it expects to create .git.
func TestGenerateProjectFailsWhenPostSetupFails(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)

	projectDir := filepath.Join(work, "postfail")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".git"), []byte("not a gitfile\n"), 0o644); err != nil {
		t.Fatalf("write .git: %v", err)
	}

	var err error
	output := captureStdout(t, func() {
		// --force is required because the directory already holds the .git file.
		err = generateProject("postfail", "1.27", "iot", "", true)
	})

	if err == nil {
		t.Fatal("generateProject with failing git init = nil, want error")
	}
	if !strings.Contains(err.Error(), "git init") {
		t.Errorf("error %q does not name the failing step", err.Error())
	}
	if strings.Contains(output, "Project setup complete") {
		t.Errorf("success line printed despite post-setup failure:\n%s", output)
	}
}

// captureStdout swaps os.Stdout for a pipe while fn runs and returns what was
// written. Generated output is a handful of lines, well under the pipe buffer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}

// ---------------------------------------------------------------------------
// Go version resolution
// ---------------------------------------------------------------------------

// TestDetectGoVersionMatchesToolchain keeps the --go auto-detect branch alive:
// the flag's default is empty precisely so this path runs.
func TestDetectGoVersionMatchesToolchain(t *testing.T) {
	got := detectGoVersion()
	if err := validateGoVersion(got); err != nil {
		t.Fatalf("detectGoVersion() = %q, which fails validation: %v", got, err)
	}
	if got == goVersionFallback {
		// Acceptable only when the toolchain could not be queried; flag it so a
		// silently broken detector is visible rather than passing quietly.
		t.Logf("detectGoVersion() returned the fallback %q", goVersionFallback)
	}
}

// TestGoFlagDefaultIsEmpty guards the choice behind A10: a static default makes
// the toolchain-detection branch unreachable, which is how every generated
// go.mod came out pinned to an old version on a newer toolchain.
func TestGoFlagDefaultIsEmpty(t *testing.T) {
	var found bool
	for _, flagDef := range newApp().Flags {
		stringFlag, ok := flagDef.(*cli.StringFlag)
		if !ok || stringFlag.Name != "go" {
			continue
		}
		found = true
		if stringFlag.Value != "" {
			t.Errorf("--go default = %q, want \"\" so detectGoVersion runs", stringFlag.Value)
		}
	}
	if !found {
		t.Fatal("no --go string flag registered")
	}
}

// TestDetectGoVersionFallback covers the branch taken when the toolchain cannot
// be queried at all; it must still yield a version the generated go.mod accepts.
func TestDetectGoVersionFallback(t *testing.T) {
	t.Setenv("PATH", "")

	got := detectGoVersion()
	if got != goVersionFallback {
		t.Fatalf("detectGoVersion() with no toolchain on PATH = %q, want %q", got, goVersionFallback)
	}
	if err := validateGoVersion(got); err != nil {
		t.Fatalf("fallback %q fails validation: %v", got, err)
	}
}

// TestEnsureTargetDir covers each branch of the destination check directly,
// including the --force path that lets a non-empty directory through.
func TestEnsureTargetDir(t *testing.T) {
	root := t.TempDir()

	missing := filepath.Join(root, "missing")
	if err := ensureTargetDir(missing, false); err != nil {
		t.Errorf("ensureTargetDir(missing, force=false) = %v, want nil", err)
	}

	empty := filepath.Join(root, "empty")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatalf("mkdir empty: %v", err)
	}
	if err := ensureTargetDir(empty, false); err != nil {
		t.Errorf("ensureTargetDir(empty, force=false) = %v, want nil", err)
	}

	occupied := filepath.Join(root, "occupied")
	if err := os.MkdirAll(occupied, 0o755); err != nil {
		t.Fatalf("mkdir occupied: %v", err)
	}
	if err := os.WriteFile(filepath.Join(occupied, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := ensureTargetDir(occupied, false); err == nil {
		t.Error("ensureTargetDir(non-empty, force=false) = nil, want error")
	}
	if err := ensureTargetDir(occupied, true); err != nil {
		t.Errorf("ensureTargetDir(non-empty, force=true) = %v, want nil", err)
	}
}
