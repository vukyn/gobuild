module testproj

go 1.24

require github.com/gofiber/fiber/v3 v3.4.0

// Transitive, pinned forward for a vulnerability fix rather than for a direct
// import. Fiber v3 pulls x/net/idna, which pulls x/text; the version it would
// otherwise resolve to (v0.38.0) carries GO-2026-5970, and govulncheck finds it
// REACHABLE from app.Listen. Minimal version selection takes the maximum, so
// this line raises it. Drop it once Fiber requires v0.39.0 or later itself.
require golang.org/x/text v0.39.0 // indirect
