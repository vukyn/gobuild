package middlewares

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ownershipScanRoot is the tree this invariant scans: everything under internal/,
// reached relative to this package's directory.
const ownershipScanRoot = ".."

// ownershipMinScannedFiles guards against a path typo making the walk vacuous —
// the one way this test could pass while proving nothing. A freshly scaffolded
// service has ~23 scannable files under internal/ and only grows, so this floor is
// deliberately low; raise it if you like, but never let it reach zero.
const ownershipMinScannedFiles = 15

// ownershipAllowedFiles are the files permitted to release a di container: whoever
// CREATES a sub-container owns it.
//
//   - middlewares/middleware.go — DiContainerMiddleware creates the request
//     sub-container and releases it with its own defer.
//
// Add an entry ONLY for another genuine creator, with a comment saying why. A
// startup one-off that opens its own sub-container outside any request qualifies
// (an idempotent owner/seed bootstrap in internal/app is the usual example). A
// handler never does: it only borrows the container from the Fiber locals.
//
// cmd/main.go's shutdown teardown of the app container is outside internal/, so it
// is not scanned and needs no entry.
var ownershipAllowedFiles = map[string]bool{
	filepath.Join("middlewares", "middleware.go"): true,
}

// forbiddenReleaseCalls are the ways a borrower could release a container it does
// not own. `ctn` is the name handlers bind the request container to.
var forbiddenReleaseCalls = []string{
	"ctn.Delete()",
	"ctn.DeleteWithSubContainers()",
}

// TestNoHandlerReleasesRequestContainer pins the container-ownership rule
// STATICALLY: whoever creates a sub-container releases it, and nothing else may.
//
// It exists because a handler that releases a container it does not own is
// invisible every other way. The compiler is happy — `ctn` is in scope and used.
// sarulabs/di's second Delete returns nil, so there is no error. The only runtime
// symptom is every registered Close running twice, and Request-scoped Close hooks
// are typically debug log lines. A runtime test cannot see it either: such a test
// mounts stub routes, so a `defer ctn.Delete()` living in a REAL handler never
// executes inside it and a "Close ran exactly once" assertion still passes.
//
// This is not hypothetical. Four services in this platform shipped the
// handler-owned pattern and leaked a sub-container on every request that
// short-circuited before reaching a handler — auth 401s, RBAC 403s, rate-limit
// 429s, /assets, /favicon.svg, 404s, every SPA render. Measured on one of them:
// live heap climbed monotonically to 58 MB over 60k unauthenticated 401s, against
// 3-4 MB once the middleware owned the release. Sweeping the handlers fixed it (50,
// 55 and 86 call sites respectively); this test is what keeps it fixed, and in one
// of those sweeps it caught a removal that a stray `git checkout --` had silently
// restored.
func TestNoHandlerReleasesRequestContainer(t *testing.T) {
	root, err := filepath.Abs(ownershipScanRoot)
	if err != nil {
		t.Fatalf("resolve scan root: %v", err)
	}

	scanned := 0
	violations := make([]string, 0)

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if ownershipAllowedFiles[relative] {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++

		for number, line := range strings.Split(string(content), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				// A comment may legitimately quote the forbidden call while
				// explaining the rule.
				continue
			}
			for _, call := range forbiddenReleaseCalls {
				if strings.Contains(trimmed, call) {
					violations = append(violations, relative+":"+strconv.Itoa(number+1)+": "+trimmed)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	if scanned < ownershipMinScannedFiles {
		t.Fatalf("scanned only %d .go files under %s, want at least %d — the invariant is not actually looking at the codebase", scanned, root, ownershipMinScannedFiles)
	}

	if len(violations) > 0 {
		t.Fatalf("%d file(s) release a di container they do not own:\n\t%s\n\nDiContainerMiddleware creates the request sub-container and releases it on every path; a handler only borrows it. A second Delete returns nil but re-runs every registered Close, so this is invisible at runtime — remove the call.",
			len(violations), strings.Join(violations, "\n\t"))
	}
}
