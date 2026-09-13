package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vukyn/gobuild/version"

	"github.com/urfave/cli/v2"
)

func main() {
	if err := newApp().Run(reorderArgs(os.Args)); err != nil {
		log.Fatal(err)
	}
}

// newApp builds the CLI definition. It is a function rather than an inline
// literal so the flag set (notably the --go default, which must stay empty for
// toolchain detection to run) is reachable from tests.
func newApp() *cli.App {
	return &cli.App{
		Name:    "gobuild",
		Usage:   "Generate a new Golang project template",
		Version: version.Current,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Project name",
				Required: false,
			},
			&cli.StringFlag{
				Name:  "go",
				Usage: "Go version (defaults to the local toolchain)",
				// Deliberately empty: a static default makes the
				// auto-detect branch in generateProject unreachable and
				// goes stale silently at every toolchain bump.
				Value: "",
			},
			&cli.StringFlag{
				Name:    "http-template",
				Aliases: []string{"preset"},
				Usage:   "Project preset (base|fiber|platform-service|platform-service-v3|iot)",
				Value:   "base",
			},
			&cli.StringFlag{
				Name:     "module",
				Aliases:  []string{"m"},
				Usage:    "Go module path (defaults to github.com/vukyn/<name>)",
				Required: false,
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Render into an existing non-empty directory, overwriting files (an existing .env is still preserved)",
			},
		},
		Action: func(c *cli.Context) error {
			var projectName string
			if c.NArg() == 0 {
				projectName = c.String("name")
			} else {
				projectName = c.Args().First()
			}
			goVersion := c.String("go")
			preset := c.String("http-template")
			modulePath := c.String("module")
			force := c.Bool("force")
			return generateProject(projectName, goVersion, preset, modulePath, force)
		},
	}
}

// valueFlags lists the flags that consume the following token as their value.
// reorderArgs uses this to skip flag values when separating flags from the
// positional project-name argument.
var valueFlags = map[string]bool{
	"-n": true, "--name": true,
	"--go":            true,
	"--http-template": true,
	"--preset":        true,
	"-m":              true,
	"--module":        true,
}

// reorderArgs moves flag tokens ahead of positional arguments so that
// "gobuild <name> --flag value" parses the same as "gobuild --flag value <name>".
// urfave/cli v2 stops flag parsing at the first positional argument, so without
// this a trailing --http-template/--go would be silently ignored.
func reorderArgs(args []string) []string {
	if len(args) <= 1 {
		return args
	}

	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	rest := args[1:]
	for index := 0; index < len(rest); index++ {
		token := rest[index]
		if strings.HasPrefix(token, "-") {
			flags = append(flags, token)
			// Consume the value token for flags that expect one, unless the
			// value was already attached via "--flag=value".
			if !strings.Contains(token, "=") && valueFlags[token] && index+1 < len(rest) {
				index++
				flags = append(flags, rest[index])
			}
			continue
		}
		positionals = append(positionals, token)
	}

	reordered := make([]string, 0, len(args))
	reordered = append(reordered, args[0])
	reordered = append(reordered, flags...)
	reordered = append(reordered, positionals...)
	return reordered
}

// licenseAuthor resolves the copyright holder for the generated LICENSE file,
// reading the local git user.name and falling back to the platform owner.
func licenseAuthor() string {
	if out, err := exec.Command("git", "config", "user.name").Output(); err == nil {
		if name := strings.TrimSpace(string(out)); name != "" {
			return name
		}
	}
	return "vukyn"
}

// hasGoMod reports whether the rendered project contains a go.mod (i.e. it is a
// Go preset). Non-Go presets (e.g. iot) skip the go mod tidy step.
func hasGoMod(projectDir string) bool {
	_, err := os.Stat(filepath.Join(projectDir, "go.mod"))
	return err == nil
}

// detectGoVersion reads the local toolchain version for the generated go.mod
// `go` directive, falling back to a known-good value when the toolchain cannot
// be queried or its output cannot be parsed.
func detectGoVersion() string {
	if out, err := exec.Command("go", "version").Output(); err == nil {
		for _, part := range strings.Fields(string(out)) {
			if strings.HasPrefix(part, "go1.") {
				return strings.TrimPrefix(part, "go")
			}
		}
	}
	return goVersionFallback
}

// goVersionFallback is used when the local toolchain cannot be queried.
const goVersionFallback = "1.27"

// ensureTargetDir reports whether the destination is safe to render into.
// Scaffolding overwrites files by name, so an existing project directory would
// silently lose work; only an explicit --force allows it.
func ensureTargetDir(projectDir string, force bool) error {
	entries, err := os.ReadDir(projectDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect %s: %w", projectDir, err)
	}
	if len(entries) == 0 || force {
		return nil
	}
	return fmt.Errorf("refusing to render into non-empty directory %s: pass --force to overwrite its contents", projectDir)
}

func generateProject(projectName, goVersion, preset, modulePath string, force bool) error {
	// Validate every caller-supplied value before anything is rendered.
	// text/template does no escaping, so these values reach generated go.mod
	// and package.json files verbatim.
	if err := validateProjectName(projectName); err != nil {
		return err
	}

	// Default the module path to the platform convention when not overridden.
	if modulePath == "" {
		modulePath = "github.com/vukyn/" + projectName
	}
	if err := validateModulePath(modulePath); err != nil {
		return err
	}

	if goVersion == "" {
		goVersion = detectGoVersion()
	}
	if err := validateGoVersion(goVersion); err != nil {
		return err
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// projectDir is resolved ONCE and used for the render and both post-setup
	// steps. Deriving it twice previously let the render and the `go mod
	// tidy`/`git init` steps disagree about where the project lived.
	projectDir := filepath.Join(currentDir, projectName)

	if err := ensureTargetDir(projectDir, force); err != nil {
		return err
	}

	if err := os.MkdirAll(projectDir, 0755); err != nil { // #nosec G301 -- scaffolded project dir must be user-browsable
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Render the selected preset's template tree into the project directory
	data := templateData{
		ProjectName: projectName,
		GoVersion:   goVersion,
		Preset:      preset,
		ModulePath:  modulePath,
		Author:      licenseAuthor(),
		Year:        fmt.Sprintf("%d", time.Now().Year()),
	}
	if err := renderPreset(preset, data, projectDir); err != nil {
		return err
	}

	fmt.Printf("Successfully created %s project template!\n", projectName)

	var postSetupFailures []string

	// Run go mod tidy only for Go presets (those that rendered a go.mod).
	if hasGoMod(projectDir) {
		goModTidyCmd := exec.Command("go", "mod", "tidy")
		goModTidyCmd.Dir = projectDir
		goModTidyCmd.Stdout = os.Stdout
		goModTidyCmd.Stderr = os.Stderr
		fmt.Println("Running go mod tidy...")
		if err := goModTidyCmd.Run(); err != nil {
			postSetupFailures = append(postSetupFailures, fmt.Sprintf("go mod tidy: %v", err))
		}
	} else {
		fmt.Println("No go.mod (non-Go preset) — skipping go mod tidy")
	}

	// Initialize git repository
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = projectDir
	gitInitCmd.Stdout = os.Stdout
	gitInitCmd.Stderr = os.Stderr
	fmt.Println("Initializing git repository...")
	if err := gitInitCmd.Run(); err != nil {
		postSetupFailures = append(postSetupFailures, fmt.Sprintf("git init: %v", err))
	}

	// A failed post-setup step leaves a half-configured project, so it must not
	// be reported as success — previously these were warnings and the command
	// still printed "complete" and exited 0.
	if len(postSetupFailures) > 0 {
		return fmt.Errorf("project rendered at %s but post-setup failed: %s", projectDir, strings.Join(postSetupFailures, "; "))
	}

	fmt.Println("Project setup complete, you are ready to go!")
	return nil
}
