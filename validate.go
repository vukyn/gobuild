package main

import (
	"fmt"
	"regexp"

	"golang.org/x/mod/module"
)

// Every value below is substituted into generated files by text/template,
// which performs NO escaping of any kind: whatever arrives on the command line
// is written verbatim into go.mod, package.json, firmware sources and so on.
// A newline in a module path therefore becomes a new `require` directive that
// the automatic `go mod tidy` then resolves, and a quote in a project name
// becomes new keys in ui/package.json. Validation happens once, up front, and
// rejects rather than sanitizes — a rewritten value would silently produce a
// project the caller did not ask for.

// projectNamePattern constrains the project name to a single safe path
// segment. Besides gating template substitution this is what keeps the name
// usable as a directory name: no separator, no "..", no leading dot and no
// leading slash can match, so the rendered tree always lands in a fresh
// subdirectory of the working directory rather than somewhere above or
// outside it.
var projectNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

// goVersionPattern accepts MAJOR.MINOR with an optional patch component, the
// only shapes a go.mod `go` directive takes.
var goVersionPattern = regexp.MustCompile(`^\d+\.\d+(\.\d+)?$`)

// validateProjectName reports whether the name is safe to use both as a
// template value and as the destination directory name.
func validateProjectName(projectName string) error {
	if projectName == "" {
		return fmt.Errorf("project name is required")
	}
	if !projectNamePattern.MatchString(projectName) {
		return fmt.Errorf(
			"invalid project name %q: must start with a letter or digit and contain only letters, digits, '.', '_' or '-' (max 64 characters); path separators, '..' and absolute paths are not allowed",
			projectName,
		)
	}
	return nil
}

// validateModulePath checks the Go module path with the same rules the go
// command applies, which rejects the whitespace, quotes and newlines an
// injected go.mod directive needs.
func validateModulePath(modulePath string) error {
	if err := module.CheckPath(modulePath); err != nil {
		return fmt.Errorf("invalid module path: %w", err)
	}
	return nil
}

// validateGoVersion checks the value written into the generated go.mod `go`
// directive.
func validateGoVersion(goVersion string) error {
	if !goVersionPattern.MatchString(goVersion) {
		return fmt.Errorf("invalid Go version %q: want MAJOR.MINOR or MAJOR.MINOR.PATCH (e.g. 1.27 or 1.27.1)", goVersion)
	}
	return nil
}
