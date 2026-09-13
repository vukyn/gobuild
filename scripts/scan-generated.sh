#!/usr/bin/env bash
#
# Scan the surface gobuild EMITS, not the surface it is.
#
# gosec reporting "0 issues" here covers this repository's own ~600 lines. The
# thousands of lines that reach real services live under templates/ as .tmpl
# files, which every scanner ignores: they are not Go, not JSON, not anything a
# parser will open. osv-scanner walked straight past the golden ui/package.json
# and exited 0 — not because it was clean, but because no lockfile exists
# anywhere in this repository for it to resolve against.
#
# So: render each preset into a throwaway directory and scan THAT, where the
# templates are real Go and a lockfile can be generated.
#
# Two properties keep this from rotting, and they are the point of the design:
#
#   1. The preset list is read off templates/*/ rather than written here. Adding
#      a preset enrols it automatically — nobody has to remember this file.
#   2. Which scanners run is decided by what a preset actually rendered
#      (go.mod -> Go scanners, ui/package.json -> npm audit), mirroring the
#      hasGoMod rule the tool itself uses. A preset matching neither is reported
#      as an explicit SKIP with its reason rather than quietly contributing
#      nothing, because a silent skip is exactly the failure this script exists
#      to correct.
#
# Usage:
#   scripts/scan-generated.sh            # every preset
#   scripts/scan-generated.sh iot base   # named presets only
#
# Needs network: the scaffold runs `go mod tidy`, and npm audit needs a
# generated lockfile.

set -u -o pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# NPM_AUDIT_LEVEL is the severity at which npm audit fails the run. Advisories
# below it are still printed.
npm_audit_level="${NPM_AUDIT_LEVEL:-high}"

failures=0
summary=()

note() { printf '\n\033[1m== %s\033[0m\n' "$*"; }
record() { summary+=("$1"); }

fail() {
	failures=$((failures + 1))
	record "FAIL  $1"
	printf '\033[31mFAIL\033[0m  %s\n' "$1"
}

pass() {
	record "ok    $1"
	printf '\033[32mok\033[0m    %s\n' "$1"
}

skip() {
	record "skip  $1"
	printf '\033[33mskip\033[0m  %s\n' "$1"
}

# go_tool runs an installed binary, falling back to `go run <module>@latest` so
# a missing tool is a slow run rather than a skipped check. A scanner that
# silently does not run is worse than one that is not installed.
go_tool() {
	local binary="$1" module="$2"
	shift 2
	if command -v "$binary" >/dev/null 2>&1; then
		"$binary" "$@"
	else
		printf 'note: %s not on PATH, falling back to `go run %s@latest`\n' "$binary" "$module" >&2
		go run "$module@latest" "$@"
	fi
}

presets=("$@")
if [ ${#presets[@]} -eq 0 ]; then
	# Derived, never hard-coded: one directory under templates/ is one preset.
	while IFS= read -r dir; do
		presets+=("$(basename "$dir")")
	done < <(find templates -mindepth 1 -maxdepth 1 -type d | sort)
fi

if [ ${#presets[@]} -eq 0 ]; then
	echo "no presets found under templates/ — refusing to report success" >&2
	exit 1
fi

workdir="$(mktemp -d "${TMPDIR:-/tmp}/gobuild-scan.XXXXXX")"
trap 'rm -rf "$workdir"' EXIT

note "building gobuild from source"
if ! go build -o "$workdir/gobuild" .; then
	echo "cannot build gobuild — nothing to scan" >&2
	exit 1
fi

for preset in "${presets[@]}"; do
	note "preset: $preset"

	project="scan${preset//-/}"
	presetdir="$workdir/$preset"
	mkdir -p "$presetdir"

	if ! (cd "$presetdir" && "$workdir/gobuild" --preset "$preset" "$project"); then
		fail "$preset: scaffold failed (a preset that cannot be generated cannot be scanned)"
		continue
	fi
	generated="$presetdir/$project"

	scanned=0

	# --- Go surface -------------------------------------------------------
	if [ -f "$generated/go.mod" ]; then
		scanned=1

		if (cd "$generated" && go_tool govulncheck golang.org/x/vuln/cmd/govulncheck ./...); then
			pass "$preset: govulncheck"
		else
			fail "$preset: govulncheck"
		fi

		if (cd "$generated" && go_tool gosec github.com/securego/gosec/v2/cmd/gosec -quiet ./...); then
			pass "$preset: gosec"
		else
			fail "$preset: gosec"
		fi
	fi

	# --- npm surface ------------------------------------------------------
	if [ -f "$generated/ui/package.json" ]; then
		scanned=1

		if ! command -v npm >/dev/null 2>&1; then
			fail "$preset: npm audit (npm not installed — dependency advisories went unchecked)"
		elif ! (cd "$generated/ui" && npm install --package-lock-only --ignore-scripts --no-audit --no-fund >/dev/null); then
			# The generated project ships no lockfile by design, so one is
			# resolved here. Without it npm audit and osv-scanner both exit 0
			# having inspected nothing.
			fail "$preset: npm audit (could not resolve a lockfile)"
		elif (cd "$generated/ui" && npm audit --audit-level="$npm_audit_level"); then
			pass "$preset: npm audit (fails at $npm_audit_level and above)"
		else
			fail "$preset: npm audit (advisories at $npm_audit_level or above)"
		fi
	fi

	# --- everything else --------------------------------------------------
	if [ "$scanned" -eq 0 ]; then
		# Today this is `iot`: C++/PlatformIO firmware. No Go module and no
		# package.json, so govulncheck, gosec, osv-scanner and npm audit all
		# have nothing to open — none of them applies, and pretending otherwise
		# by running them for a green tick would be worse than saying so.
		#
		# Its real dependency risk is the pinned kuino tag in platformio.ini,
		# and the gate for that is `pio run` on the scaffold (~35s with the
		# ESP32 toolchain cached), which is a toolchain this script does not
		# require. It is NOT wired in here on purpose: it would make every run
		# depend on PlatformIO. Run it by hand when touching the preset — see
		# CLAUDE.md, "Pinned dependency versions in templates rot silently".
		skip "$preset: no Go module and no package.json — no applicable scanner (see CLAUDE.md for the pio gate)"
	fi
done

note "summary"
printf '%s\n' "${summary[@]}"

if [ "$failures" -gt 0 ]; then
	printf '\n%d check(s) failed\n' "$failures"
	exit 1
fi

printf '\nall checks passed\n'
