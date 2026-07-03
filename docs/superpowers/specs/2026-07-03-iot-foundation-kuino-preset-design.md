# IoT Platform Foundation — `kuino` shared lib + gobuild `iot` preset

**Date:** 2026-07-03
**Status:** Approved design (pending spec review)
**Author:** vukyn

## Goal

Establish a reusable, unified foundation for onboarding multiple IoT firmware
projects into the platform — the IoT analogue of the Go services' `kuery` +
gobuild `platform-service` split.

Two deliverables:

- **A — `kuino`**: a new shared PlatformIO/C++ library repo (IoT's `kuery`).
- **B — gobuild `iot` preset**: a minimal ESP32 firmware skeleton that consumes
  `kuino`, scaffolded via `gobuild --preset iot`.

`rainybox` (the only existing IoT repo) is the source of the reusable code and
the structural template, but it is **not** refactored to consume `kuino` as part
of this work (see Out of Scope).

## Non-goals (YAGNI)

- No OLED/wokwi/CI wiring in the preset — add per project.
- No second `iot-oled` preset — one minimal preset for now.
- No MQTT/BLE/sensor modules in `kuino` until a real repo needs them.
- No migration of `rainybox` onto `kuino` in this spec.

---

## Deliverable A — `kuino` shared library

### What it is

New standalone git repo at `github.com/vukyn/kuino`, living under the platform
root as `pet-platform/kuino/`. A PlatformIO library, consumed by IoT firmware
repos via `lib_deps` git URL pinned to a version tag — the exact reuse model
`kuery` uses for Go services (versioned, imported, never copied per project).

### Layout (PlatformIO library convention)

```
kuino/
  library.json            ; PlatformIO manifest: name, version, deps
  src/
    kuino/
      wifi.h    wifi.cpp
      httpjson.h httpjson.cpp
      display.h display.cpp
      button.h  button.cpp
  examples/                ; one minimal sketch per module
  README.md
  CLAUDE.md
  .gitignore              ; .pio/
  LICENSE
```

Consumers include modules as `#include <kuino/wifi.h>`.

### Seed modules (extracted from rainybox `src/main.cpp`)

All four are lifted from rainybox's 446-line monolith, namespaced, and split
into header + impl:

| Module | Source in rainybox | Responsibility |
|--------|--------------------|----------------|
| `wifi` | `connectWifi()` (~90 lines), `WOKWI` branch, diag timing | Connect + reconnect, assoc/dhcp timing under diag flag, Wokwi-GUEST branch |
| `httpjson` | `getJson()`, `apiBase()` | HTTPS GET via `WiFiClientSecure`+`HTTPClient`, `ArduinoJson` filtered parse |
| `display` | `fitWidth`, `drawHeader`, `drawScroll`, marquee state, `toast` | U8g2 128×64 text-fit, header, scrolling line, marquee, transient toast |
| `button` | `pressed()` edge-debounce, `nextStation()` cycle helper | Active-low debounced edge detect + multi-button cycle helper |

### Manifest & dependencies

`library.json` declares transitive deps so consumers get them automatically:

```json
{
  "name": "kuino",
  "version": "0.1.0",
  "dependencies": {
    "olikraus/U8g2": "^2.35.30",
    "bblanchon/ArduinoJson": "^7.2.0"
  }
}
```

`display` is the only module that needs U8g2; ArduinoJson is needed by
`httpjson`. Both listed at the library level (PlatformIO has no per-source
optional-dep granularity for a single library) — acceptable, small footprint.

### Diagnostics flag

rainybox uses `-DRAINYBOX_DIAG` to gate verbose WiFi/timing logs. In `kuino`
this generalizes to **`-DKUINO_DIAG`**. Consumers opt in via a `build_flags`
env.

### Versioning & governance

- Semantic tags: first tag `v0.1.0`.
- **Keep only the 5 newest tags** (same retention rule as `kuery`); delete older
  tags local + remote after each bump.
- Workflow: edit module → commit → tag (bump minor) → bump `#vX.Y.Z` pin in each
  consumer's `platformio.ini`.

---

## Deliverable B — gobuild `iot` preset

### What it scaffolds

`gobuild --preset iot -n <name>` produces a **minimal** ESP32-S3 firmware
skeleton (philosophically the `base` preset of the IoT world — not a full
rainybox clone):

```
<name>/
  platformio.ini            ; 1 env (esp32-s3-devkitc-1) + kuino git dep + U8g2 + ArduinoJson
  src/main.cpp              ; thin: kuino::wifi connect + serial "hello" + poll stub
  include/config.h.example  ; WIFI_SSID/PASSWORD + API base placeholder + OLED pins
  .gitignore                ; include/config.h + .pio/
  README.md                 ; templated (build/flash, kuino sync note)
  CLAUDE.md                 ; templated; marks repo as non-Go firmware (platform template N/A)
```

### `platformio.ini` (rendered)

```ini
; {{.ProjectName}} — ESP32-S3 firmware
[env:esp32-s3]
platform = espressif32
board = esp32-s3-devkitc-1
framework = arduino
monitor_speed = 115200
build_flags = -DARDUINO_USB_CDC_ON_BOOT=1
lib_deps =
	https://github.com/vukyn/kuino.git#v0.1.0
	olikraus/U8g2@^2.35.30
	bblanchon/ArduinoJson@^7.2.0
```

Single env (no classic/wokwi/diag variants — those are per-project additions).

### `src/main.cpp` (rendered, thin)

Minimal, wired to `kuino`:

```cpp
#include <Arduino.h>
#include <kuino/wifi.h>
#include "config.h"

void setup() {
  Serial.begin(115200);
  kuino::wifi::connect(WIFI_SSID, WIFI_PASSWORD);
  Serial.println("{{.ProjectName}} online");
}

void loop() {
  // poll stub — add kuino::httpjson + your device logic here
  delay(1000);
}
```

Exact `kuino` public API signatures are finalized during implementation (extract
step defines them); the preset's `main.cpp` is updated to match whatever `kuino`
exports.

### `config.h.example` (rendered)

WiFi creds, an API-base placeholder, and OLED I2C pins — mirrors rainybox's,
minus rainy-specific station macros.

### gobuild code change

Current `generateProject` (main.go:176-185) unconditionally runs `go mod tidy`,
which fails/noise for a non-Go preset. Fix is **preset-agnostic**: run
`go mod tidy` only if a `go.mod` was rendered.

```go
// after renderPreset, before/around the existing go mod tidy block:
if _, err := os.Stat(filepath.Join(projectDir, "go.mod")); err == nil {
    // existing go mod tidy block runs here
} else {
    fmt.Println("No go.mod (non-Go preset) — skipping go mod tidy")
}
```

`git init` stays (runs for all presets). Go-version detection stays as-is —
harmless (its template var is simply unused by the `iot` preset).

### Metadata / docs updates

- `--preset` flag usage string: `(base|fiber|platform-service|iot)`
  (main.go:36).
- gobuild `README.md`: document the `iot` preset and that gobuild now emits
  non-Go skeletons.
- Root `pet-platform/CLAUDE.md`: update the gobuild entry (no longer
  "Go project skeletons" exclusively) and add `kuino/` to the repo list with a
  one-line description + the IoT shared-lib rule (mirror of the kuery rule,
  scoped to firmware).

---

## Build order (dependency-driven)

`kuino` must exist and be tagged before the preset can pin it.

1. **Build `kuino`**: create repo, extract the 4 modules from rainybox, write
   `library.json` + README + CLAUDE.md, `git init`, push to GitHub, tag `v0.1.0`.
2. **Build gobuild `iot` preset**: add `templates/iot/`, gate `go mod tidy`,
   update usage string + README.
3. **Docs/governance**: root CLAUDE.md updates; onboard `kuino` via the normal
   flow (`.claude/onboarding/kuino.json`).
4. **Smoke test**: `gobuild --preset iot -n testiot` → `cd testiot` →
   `pio run` compiles clean pulling `kuino` from git.

## Testing

- **kuino**: `examples/` sketches compile per module (`pio run` in each example
  dir, or a wokwi build). No unit-test framework introduced.
- **gobuild**: existing golden-file test pattern (`gobuild` uses golden tests per
  preset — see `gobuild-preset-system` memory) extended with an `iot` golden
  tree. `git add -f` the golden fixtures.
- **End-to-end**: the smoke test in Build order step 4 is the acceptance gate.

## Risks / notes

- **First non-Go artifact in gobuild** — keep the Go-specific post-render steps
  strictly guarded so future non-Go presets inherit the same safety.
- **kuino git-URL pin resolves at `pio run`** — requires network + a pushed tag;
  a floating `#main` pin would be reproducibility risk, so always pin a tag.
- **Commits go via PR** per platform rule (committer agent / `/pet-commit`); no
  direct pushes to `main` on either repo.
