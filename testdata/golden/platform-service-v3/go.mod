module github.com/vukyn/testproj

go 1.24

// No `golang.org/x/text` or `gofiber/utils` forward-pin here, deliberately —
// unlike the `fiber` and `platform-service` presets. kuery already requires
// `x/text v0.41.0` (above GO-2026-5970's v0.39.0 fix line), and
// `template/html/v3` uses `gofiber/utils/v2`, so the v1 module carrying
// GO-2025-4208 is not in this graph at all. Re-check with `govulncheck` after
// any kuery bump rather than assuming.
//
// `github.com/gofiber/fiber/v2` does NOT appear here even though kuery is one
// module carrying both the v2 and v3 helper lines: this service imports only
// kuery's v3 twins (ctxv3 / http/fiberv3 / recoverv3), and module-graph pruning
// means `go mod tidy` records only what the imported packages actually need.
//
// ⚠️ Treat a `github.com/gofiber/fiber/v2 // indirect` line appearing here as a
// TRIPWIRE, not as noise: it means something in this service started importing a
// kuery **v2** package. The usual cause is a file switched from `kuery/ctxv3` to
// `kuery/ctx` — which fails to build at first (no go.sum entry for fiber/v2), then
// builds, vets and tests perfectly green once someone reflexively runs
// `go mod tidy`, while silently writing an empty `created_by`/`updated_by` on
// every row. Find the v2 import and change it back; do not "fix" the go.mod line.
require (
	github.com/gofiber/fiber/v3 v3.5.0
	github.com/gofiber/template/html/v3 v3.0.8
	github.com/joho/godotenv v1.5.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/sarulabs/di/v2 v2.5.2
	github.com/uptrace/bun v1.3.0
	github.com/vukyn/kuery v1.71.0
)

// Transitive, pinned forward for vulnerability fixes rather than for a direct
// import. fasthttp (under Fiber) pulls both compressors, and the versions it
// would otherwise resolve to carry GO-2026-6888 (brotli: excessive memory when
// decompressing crafted streams) and GO-2026-6865 (compress: panic on two-block
// frames), both REACHABLE per govulncheck. Minimal version selection takes the
// maximum, so these lines raise them. Drop them once fasthttp requires
// brotli v1.2.7 and compress v1.20.2 or later itself.
require (
	github.com/andybalholm/brotli v1.2.7 // indirect
	github.com/klauspost/compress v1.20.2 // indirect
)
