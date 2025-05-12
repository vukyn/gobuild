package main

import (
	"fmt"
	"gobuild/tmpl"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "gobuild",
		Usage: "Generate a new Golang project template",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "name",
				Aliases:  []string{"n"},
				Usage:    "Project name",
				Required: true,
			},
			&cli.StringFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Go version",
				Value:   "1.24",
			},
		},
		Action: func(c *cli.Context) error {
			var projectName string
			if c.NArg() == 0 {
				projectName = c.String("name")
			} else {
				projectName = c.Args().First()
			}
			goVersion := c.String("version")
			return generateProject(projectName, goVersion)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func generateProject(projectName, goVersion string) error {
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
		"main.go":   tmpl.MAIN_GO,
		"go.mod":    tmpl.GO_MOD,
		".env":      tmpl.ENV,
		"Makefile":  tmpl.MAKEFILE,
		"README.md": tmpl.README,
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
	fmt.Printf("To get started:\n\ncd %s\ngo mod tidy\n", projectName)
	return nil
}
