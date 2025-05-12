package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vukyn/gobuild/tmpl"

	"github.com/urfave/cli/v2"
)

var (
	Version = "1.2.2"
)

func main() {
	app := &cli.App{
		Name:    "gobuild",
		Usage:   "Generate a new Golang project template",
		Version: Version,
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
		},
		Action: func(c *cli.Context) error {
			var projectName string
			if c.NArg() == 0 {
				projectName = c.String("name")
			} else {
				projectName = c.Args().First()
			}
			goVersion := c.String("go")
			return generateProject(projectName, goVersion)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func generateProject(projectName, goVersion string) error {
	if projectName == "" {
		return fmt.Errorf("project name is required")
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
	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Template files to be created
	files := map[string]string{
		"main.go":    tmpl.MAIN_GO,
		"go.mod":     tmpl.GO_MOD,
		".env":       tmpl.ENV,
		"Makefile":   tmpl.MAKEFILE,
		"README.md":  tmpl.README,
		".gitignore": tmpl.GIT_IGNORE,
	}

	// Create each file in the project directory
	for filename, content := range files {
		content = strings.ReplaceAll(content, tmpl.PROJECT_NAME, projectName)
		content = strings.ReplaceAll(content, tmpl.GO_VERSION, goVersion)
		filePath := filepath.Join(projectName, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", filename, err)
		}
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
