package main

import "embed"

// templatesFS holds every preset's template tree, embedded at build time so
// the binary is self-contained and runs from any working directory.
//
//go:embed all:templates
var templatesFS embed.FS
