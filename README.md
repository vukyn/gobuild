# gobuild - Basic Golang Project Template Generator

A simple yet powerful CLI tool to generate complete Golang project templates with a single command.

[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/vukyn/gobuild)

## Features

-   Generates a complete project structure in seconds
-   Creates essential files (main.go, go.mod, .env, Makefile, README.md)
-   Automatically initializes git repository
-   Runs `go mod tidy` to set up dependencies
-   Uses your local Go version for generated files
-   Customizable project name

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/vukyn/gobuild.git
cd gobuild

# Build the binary
go build -o gobuild

# Optional: Move to your PATH for global access
mv gobuild /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/vukyn/gobuild@latest
```

## Usage

```bash
# Simple usage
gobuild hello-world

# With custom go version
gobuild hello-world --go 1.24.2
```

## Generated Project Structure

The tool generates the following structure:

```
hello-world/
├── .env                # Environment variables
├── Makefile            # Common development commands
├── README.md           # Project documentation
├── go.mod              # Go module configuration
├── todo                # Todo list
└── main.go             # Main application entry point
```

## Example Output

```
$ gobuild awesomeproject
$ Successfully created awesomeproject project template!
$ Running go mod tidy...
$ Initializing git repository...
$ Initialized empty Git repository in /Users/username/awesomeproject/.git/
$ Project setup complete, you are ready to go!
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
