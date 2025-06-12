package tmpl

const (
	GO_MOD = `module {{.ProjectName}}

go ` + GO_VERSION + `

require (
	// Add your dependencies here
)
`
)