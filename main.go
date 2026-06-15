package main

import (
	"fmt"
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
	app := &cli.App{
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
				Usage: "Go version",
				Value: "1.24",
			},
			&cli.StringFlag{
				Name:    "http-template",
				Aliases: []string{"preset"},
				Usage:   "Project preset (base|fiber|platform-service)",
				Value:   "base",
			},
			&cli.StringFlag{
				Name:     "module",
				Aliases:  []string{"m"},
				Usage:    "Go module path (defaults to github.com/vukyn/<name>)",
				Required: false,
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
			return generateProject(projectName, goVersion, preset, modulePath)
		},
	}

	if err := app.Run(reorderArgs(os.Args)); err != nil {
		log.Fatal(err)
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

func generateProject(projectName, goVersion, preset, modulePath string) error {
	if projectName == "" {
		return fmt.Errorf("project name is required")
	}

	// Default the module path to the platform convention when not overridden.
	if modulePath == "" {
		modulePath = "github.com/vukyn/" + projectName
	}

	if goVersion == "" {
		// Get current Go version
		if out, err := exec.Command("go", "version").Output(); err == nil {
			parts := strings.Fields(string(out))
			for _, part := range parts {
				if strings.HasPrefix(part, "go1.") {
					goVersion = strings.TrimPrefix(part, "go")
					break
				}
			}
			if goVersion == "" {
				goVersion = "1.24"
			}
		}
	}

	// Create project directory
	if err := os.MkdirAll(projectName, 0755); err != nil { // #nosec G301 -- scaffolded project dir must be user-browsable
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
	if err := renderPreset(preset, data, projectName); err != nil {
		return err
	}

	fmt.Printf("Successfully created %s project template!\n", projectName)

	// Change to project directory for subsequent commands
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Run go mod tidy in the project directory
	projectDir := filepath.Join(currentDir, projectName)
	goModTidyCmd := exec.Command("go", "mod", "tidy")
	goModTidyCmd.Dir = projectDir
	goModTidyCmd.Stdout = os.Stdout
	goModTidyCmd.Stderr = os.Stderr
	fmt.Println("Running go mod tidy...")
	if err := goModTidyCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to run go mod tidy: %v\n", err)
	}

	// Initialize git repository
	gitInitCmd := exec.Command("git", "init")
	gitInitCmd.Dir = projectDir
	gitInitCmd.Stdout = os.Stdout
	gitInitCmd.Stderr = os.Stderr
	fmt.Println("Initializing git repository...")
	if err := gitInitCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to initialize git repository: %v\n", err)
	}

	fmt.Println("Project setup complete, you are ready to go!")
	return nil
}
