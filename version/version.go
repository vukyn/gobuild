package version

import (
	"regexp"
	"runtime/debug"
)

// pseudoVersion matches Go module pseudo-versions, e.g.
// "v1.3.1-0.20260615072431-32e35379e0f8" (optionally suffixed with "+dirty").
// These are synthesized for untagged commits (including local builds from a
// checkout), are not real releases, and so fall through to the VCS-revision
// fallback. The timestamp may follow either "-" or "." depending on the base
// tag form (vX.Y.Z-0.<ts>-<hash> vs vX.Y.Z-<ts>-<hash>).
var pseudoVersion = regexp.MustCompile(`^v.+[-.]\d{14}-[0-9a-f]{12}(\+[0-9A-Za-z.-]+)?$`)

// Current is the CLI version, resolved once at startup from the binary's build
// information. There is no hand-maintained version string: the single source of
// truth is the git tag, which `go install github.com/vukyn/gobuild@vX.Y.Z`
// stamps into the module's build info.
var Current = resolve()

// resolve derives the version from runtime/debug build info.
//
//   - When the binary was installed from a tagged module version, the main
//     module version (e.g. "v1.4.0") is used directly.
//   - When built from a local checkout, the VCS revision is used to produce a
//     fallback such as "dev (a1b2c3d)" or "dev (a1b2c3d, dirty)".
//   - When no build info is available at all, "dev" is returned.
func resolve() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	if mainVersion := info.Main.Version; mainVersion != "" && mainVersion != "(devel)" && !pseudoVersion.MatchString(mainVersion) {
		return mainVersion
	}

	return vcsVersion(info.Settings)
}

// vcsVersion builds the "dev (...)" fallback from the VCS build settings. The
// revision is shortened to a git-style 7-character prefix and annotated with
// ", dirty" when the working tree had uncommitted changes at build time.
func vcsVersion(settings []debug.BuildSetting) string {
	var revision string
	var modified bool
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if revision == "" {
		return "dev"
	}

	if len(revision) > 7 {
		revision = revision[:7]
	}

	if modified {
		return "dev (" + revision + ", dirty)"
	}
	return "dev (" + revision + ")"
}
