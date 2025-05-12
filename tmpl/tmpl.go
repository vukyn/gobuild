package tmpl

const (
	PROJECT_NAME = "{{.ProjectName}}"
	GO_VERSION   = "{{.GoVersion}}"
)

const (
	MAIN_GO = `package main

import "fmt"

func main() {
	fmt.Println("Hello from ` + PROJECT_NAME + `!")
}
`

	GO_MOD = `module {{.ProjectName}}

go ` + GO_VERSION + `

require (
	// Add your dependencies here
)
`

	ENV = `# Environment variables
APP_NAME=` + PROJECT_NAME + `
APP_ENV=development
`

	MAKEFILE = `# Makefile for ` + PROJECT_NAME + `

run:
	go run main.go
	`

	README = `# ` + PROJECT_NAME + `

This is a Go project generated using gobuild CLI tool.

## Prerequisites

- Go ` + GO_VERSION + ` or higher
- Make (optional, for using Makefile commands)

## Installation

1. Clone this repository:
   ` + "```" + `bash
   git clone <your-repo-url>
   cd ` + PROJECT_NAME + `
   ` + "```" + `

2. Install dependencies:
   ` + "```" + `bash
   go mod tidy
   ` + "```" + `

## Usage

### Using Make commands

- Run the project:
  ` + "```" + `bash
  make run
  ` + "```" + `

### Direct Go commands

- Run the project:
  ` + "```" + `bash
  go run main.go
  ` + "```" + `

## Project Structure

- ` + "`main.go`" + `: Main application entry point
- ` + "`go.mod`" + `: Go module definition
- ` + "`go.sum`" + `: Go module checksums
- ` + "`.env`" + `: Environment variables
- ` + "`Makefile`" + `: Build and development commands
- ` + "`README.md`" + `: Project documentation

## License

This project is licensed under the MIT License.
`
)
