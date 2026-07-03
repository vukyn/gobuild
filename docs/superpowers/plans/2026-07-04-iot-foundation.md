# IoT Platform Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-driven-development (recommended) or executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `kuino` (a shared PlatformIO/C++ library, IoT's `kuery`) plus a gobuild `iot` preset that scaffolds minimal ESP32 firmware consuming it, so future IoT repos onboard with one command.

**Architecture:** `kuino` extracts the four reusable subsystems from rainybox's monolithic `src/main.cpp` (WiFi connect, HTTPS+JSON client, U8g2 display helpers, button debounce) into namespaced, dependency-injected modules under `src/kuino/`. It is versioned via git tags and consumed through `lib_deps` git URL — the exact reuse model `kuery` uses for Go services. The gobuild `iot` preset renders a thin `main.cpp` that includes `kuino` modules; gobuild's Go-specific post-render step (`go mod tidy`) is gated behind a `go.mod` existence check so non-Go presets are safe.

**Tech Stack:** C++/Arduino framework · PlatformIO · ESP32-S3 (espressif32) · U8g2 · ArduinoJson v7 · Go (gobuild CLI, `text/template` + `embed.FS`).

**Spec:** `gobuild/docs/superpowers/specs/2026-07-03-iot-foundation-kuino-preset-design.md`

---

## File Structure

### New repo — `pet-platform/kuino/`
| Path | Responsibility |
|------|----------------|
| `library.json` | PlatformIO manifest: name, version, deps (U8g2, ArduinoJson), framework/platform scope |
| `src/kuino/diag.h` | `KLOG(...)` macro — verbose serial logging compiled out unless `-DKUINO_DIAG` |
| `src/kuino/button.h` / `.cpp` | Debounced falling-edge detector (pure input helper) |
| `src/kuino/httpjson.h` / `.cpp` | `getJson()` — HTTPS GET + ArduinoJson filtered parse, retries |
| `src/kuino/wifi.h` / `.cpp` | `connect()` — STA connect (headless), timing, public-DNS override |
| `src/kuino/display.h` / `.cpp` | U8g2 text helpers: `fitWidth`/`drawHeader`/`drawScroll`/`toast` (oled + fonts injected) |
| `examples/wifi_hello/wifi_hello.ino` | Minimal WiFi-connect sketch (compile gate for `wifi`) |
| `examples/poll_display/poll_display.ino` | Exercises httpjson + display + button (compile gate for the rest) |
| `README.md` / `CLAUDE.md` | Usage, module API, "non-Go firmware lib — platform Go template N/A" note |
| `.gitignore` / `LICENSE` | `.pio/` ignore; MIT license |

### Modified/new in `pet-platform/gobuild/`
| Path | Change |
|------|--------|
| `templates/iot/platformio.ini.tmpl` | Create — single esp32-s3 env + kuino git dep |
| `templates/iot/src/main.cpp.tmpl` | Create — thin firmware wired to `kuino::wifi` |
| `templates/iot/include/config.h.example.tmpl` | Create — WiFi creds + API base + OLED pins |
| `templates/iot/gitignore.tmpl` | Create — maps to `.gitignore` (`include/config.h`, `.pio/`) |
| `templates/iot/README.md.tmpl` | Create — build/flash + kuino sync note |
| `templates/iot/CLAUDE.md.tmpl` | Create — marks repo non-Go firmware |
| `main.go:176-185` | Modify — gate `go mod tidy` behind `go.mod` existence (new helper `hasGoMod`) |
| `main.go:36` | Modify — `--preset` usage string add `iot` |
| `gobuild_test.go:22` | Modify — add `"iot"` to `presets`; new unit test for `hasGoMod` |
| `testdata/golden/iot/**` | Create — golden fixtures via `go test . -update` |
| `README.md` (repo root) | Modify — document `iot` preset |
| `CLAUDE.md` (repo) | Modify — document `iot` preset + non-Go emission |

### Platform root (not a git repo — edits only, no commit)
| Path | Change |
|------|--------|
| `pet-platform/CLAUDE.md` | Add `kuino/` to repo list; update gobuild entry (non-Go presets); note IoT shared-lib rule |
| `.claude/onboarding/kuino.json` | New onboarding record for kuino |

---

## PART 1 — `kuino` shared library

> Build order is dependency-driven: `kuino` must exist and be tagged **before** the gobuild preset can pin it.

### Task 1: Create kuino repo skeleton

**Files:**
- Create: `pet-platform/kuino/library.json`
- Create: `pet-platform/kuino/.gitignore`
- Create: `pet-platform/kuino/LICENSE`

- [ ] **Step 1: Create the directory and manifest**

Create `pet-platform/kuino/library.json`:

```json
{
  "name": "kuino",
  "version": "0.1.0",
  "description": "Shared ESP32/Arduino helpers for vukyn IoT firmware: WiFi connect, HTTPS+JSON client, U8g2 display helpers, button debounce.",
  "keywords": ["esp32", "arduino", "iot", "wifi", "json", "u8g2", "oled"],
  "repository": { "type": "git", "url": "https://github.com/vukyn/kuino.git" },
  "authors": [{ "name": "vukyn" }],
  "license": "MIT",
  "frameworks": ["arduino"],
  "platforms": ["espressif32"],
  "dependencies": {
    "olikraus/U8g2": "^2.35.30",
    "bblanchon/ArduinoJson": "^7.2.0"
  }
}
```

PlatformIO adds `src/` to the include path when it exists, so headers at `src/kuino/*.h` resolve as `#include <kuino/*.h>`.

- [ ] **Step 2: Create `.gitignore`**

Create `pet-platform/kuino/.gitignore`:

```gitignore
.pio/
.vscode/
*.o
*.bin
```

- [ ] **Step 3: Create `LICENSE`** (MIT, matching the platform's other repos)

Create `pet-platform/kuino/LICENSE` with the standard MIT text, `Copyright (c) 2026 vukyn`.

- [ ] **Step 4: Initialise git**

```bash
cd pet-platform/kuino && git init && git add -A && git commit -m "chore: kuino repo skeleton + library.json"
```
Expected: `git init` prints "Initialized empty Git repository"; commit succeeds.

---

### Task 2: `diag` + `button` modules

`button` is pure input logic — the smallest module, extracted verbatim from rainybox `pressed()` (main.cpp:384-392) with a configurable debounce interval.

**Files:**
- Create: `pet-platform/kuino/src/kuino/diag.h`
- Create: `pet-platform/kuino/src/kuino/button.h`
- Create: `pet-platform/kuino/src/kuino/button.cpp`

- [ ] **Step 1: Create `diag.h`** (shared logging macro, used by later modules)

```cpp
#pragma once
#include <Arduino.h>

// Verbose serial diagnostics, compiled out entirely unless -DKUINO_DIAG is set
// in the consumer's build_flags. Mirrors rainybox's DLOG.
#ifdef KUINO_DIAG
  #define KLOG(...) Serial.printf(__VA_ARGS__)
#else
  #define KLOG(...) do {} while (0)
#endif
```

- [ ] **Step 2: Create `button.h`**

```cpp
#pragma once
#include <Arduino.h>

namespace kuino {
namespace button {

// One debounced falling-edge (press) detector. The caller owns the three state
// variables — declare one set per physical button, initialising reading/stable
// to HIGH and lastEdge to 0. Returns true exactly once per debounced HIGH->LOW
// transition (active-low button to GND with INPUT_PULLUP).
bool pressed(int pin, int &reading, int &stable, unsigned long &lastEdge,
             unsigned long debounceMs = 50);

} // namespace button
} // namespace kuino
```

- [ ] **Step 3: Create `button.cpp`**

```cpp
#include "kuino/button.h"

namespace kuino {
namespace button {

bool pressed(int pin, int &reading, int &stable, unsigned long &lastEdge,
             unsigned long debounceMs) {
  int r = digitalRead(pin);
  if (r != reading) {
    reading = r;
    lastEdge = millis();
  }
  if (millis() - lastEdge > debounceMs && r != stable) {
    stable = r;
    if (stable == LOW) return true;
  }
  return false;
}

} // namespace button
} // namespace kuino
```

- [ ] **Step 4: Commit**

```bash
cd pet-platform/kuino && git add src/kuino/diag.h src/kuino/button.h src/kuino/button.cpp && git commit -m "feat: kuino diag macro + button debounce module"
```

---

### Task 3: `httpjson` module

Extracted from rainybox `getJson()` (main.cpp:81-106). Generalised: `retries` param, `DLOG`→`KLOG`, filter defaults to null.

**Files:**
- Create: `pet-platform/kuino/src/kuino/httpjson.h`
- Create: `pet-platform/kuino/src/kuino/httpjson.cpp`

- [ ] **Step 1: Create `httpjson.h`**

```cpp
#pragma once
#include <Arduino.h>
#include <ArduinoJson.h>

namespace kuino {
namespace httpjson {

// HTTPS GET `url` and deserialize the body into `doc`. When `filter` is non-null
// it is applied as an ArduinoJson deserialization filter (parses only wanted
// fields, keeping RAM small). Returns true on a 2xx response that parsed
// cleanly. Retries up to `retries` times with a 500ms backoff — fly.dev
// cold-starts can EOF the TLS handshake and hotspot DNS can transiently miss,
// both of which usually succeed on retry. Certificate validation is skipped
// (setInsecure) — intended for hobby devices, not sensitive endpoints.
bool getJson(const String &url, JsonDocument &doc,
             const JsonDocument *filter = nullptr, int retries = 3);

} // namespace httpjson
} // namespace kuino
```

- [ ] **Step 2: Create `httpjson.cpp`**

```cpp
#include "kuino/httpjson.h"
#include "kuino/diag.h"

#include <HTTPClient.h>
#include <WiFiClientSecure.h>

namespace kuino {
namespace httpjson {

bool getJson(const String &url, JsonDocument &doc, const JsonDocument *filter,
             int retries) {
  for (int attempt = 0; attempt < retries; attempt++) {
    WiFiClientSecure client;
    client.setInsecure();           // skip cert validation (hobby device)
    client.setHandshakeTimeout(15); // seconds; cold-start handshakes are slow
    HTTPClient https;
    https.setConnectTimeout(8000);
    https.setTimeout(8000);
    if (https.begin(client, url)) {
      int code = https.GET();
      KLOG("GET %s -> %d (%s) try=%d heap=%u\n", url.c_str(), code,
           HTTPClient::errorToString(code).c_str(), attempt, ESP.getFreeHeap());
      if (code >= 200 && code < 300) {
        DeserializationError err =
            filter ? deserializeJson(doc, https.getStream(),
                                     DeserializationOption::Filter(*filter))
                   : deserializeJson(doc, https.getStream());
        https.end();
        if (!err) return true;
      } else {
        https.end();
      }
    }
    delay(500); // brief backoff before retry
  }
  return false;
}

} // namespace httpjson
} // namespace kuino
```

- [ ] **Step 3: Commit**

```bash
cd pet-platform/kuino && git add src/kuino/httpjson.h src/kuino/httpjson.cpp && git commit -m "feat: kuino httpjson client module"
```

---

### Task 4: `wifi` module

Extracted from rainybox `connectWifi()` (main.cpp:264-344). **Key generalisation:** rainybox drew an OLED spinner *inside* the connect loop — the biggest coupling. In `kuino` the module is **headless**; the app passes an optional `onTick(frame)` callback to render its own spinner. Timing state becomes module-static; `WOKWI` branch and OLED diag screen are dropped (app concerns).

**Files:**
- Create: `pet-platform/kuino/src/kuino/wifi.h`
- Create: `pet-platform/kuino/src/kuino/wifi.cpp`

- [ ] **Step 1: Create `wifi.h`**

```cpp
#pragma once
#include <Arduino.h>

namespace kuino {
namespace wifi {

// Connect in STA mode to ssid/pass and block until associated + DHCP-bound.
// Tunes the radio for fast/reliable association, forces public DNS
// (8.8.8.8 / 1.1.1.1) while keeping the DHCP IP/gateway, and never persists
// creds to NVS. Headless: draws nothing. Pass an optional `onTick` callback to
// render a spinner — it is invoked once per ~90ms wait iteration with an
// incrementing frame counter. Timing is logged under -DKUINO_DIAG.
void connect(const char *ssid, const char *pass,
             void (*onTick)(int frame) = nullptr);

// True when the station interface is connected.
bool connected();

// Association time (radio linked) and DHCP time (IP obtained) in ms from the
// most recent connect() — 0 until measured. For diagnostics/on-screen timing.
unsigned long assocMs();
unsigned long dhcpMs();

} // namespace wifi
} // namespace kuino
```

- [ ] **Step 2: Create `wifi.cpp`**

```cpp
#include "kuino/wifi.h"
#include "kuino/diag.h"

#include <WiFi.h>

namespace kuino {
namespace wifi {

namespace {
// Connect timing (ms since begin) — separates association vs DHCP.
volatile unsigned long tBegin = 0, tAssoc = 0, tGotIP = 0;
volatile int lastReason = 0; // last STA_DISCONNECTED reason code
} // namespace

void connect(const char *ssid, const char *pass, void (*onTick)(int frame)) {
  // persistent(false): never write creds/static-config to NVS. Persisting a
  // static IP/DNS previously caused reason-2 (AUTH_EXPIRE) loops that survived
  // reboots and network changes.
  WiFi.persistent(false);
  WiFi.mode(WIFI_STA);
  WiFi.setSleep(false);                          // no modem sleep during connect
  WiFi.setTxPower(WIFI_POWER_19_5dBm);           // max TX (helps the handshake)
  WiFi.setScanMethod(WIFI_ALL_CHANNEL_SCAN);     // see every AP...
  WiFi.setSortMethod(WIFI_CONNECT_AP_BY_SIGNAL); // ...join the strongest

  WiFi.onEvent([](arduino_event_id_t e) { tAssoc = millis(); },
               ARDUINO_EVENT_WIFI_STA_CONNECTED);
  WiFi.onEvent([](arduino_event_id_t e) { tGotIP = millis(); },
               ARDUINO_EVENT_WIFI_STA_GOT_IP);
  WiFi.onEvent(
      [](arduino_event_id_t e, arduino_event_info_t info) {
        lastReason = info.wifi_sta_disconnected.reason;
        KLOG("\nWiFi disconnect reason=%d\n", lastReason);
      },
      ARDUINO_EVENT_WIFI_STA_DISCONNECTED);

  tBegin = millis();
  WiFi.begin(ssid, pass);

  int frame = 0;
  while (WiFi.status() != WL_CONNECTED) {
    if (onTick) onTick(frame);
    KLOG(".");
    frame++;
    delay(90);
  }

  // Force public DNS (keep DHCP IP/gateway). Router/hotspot DNS that fails to
  // resolve surfaces as WiFiClientSecure "start_ssl_client: -1".
  WiFi.config(WiFi.localIP(), WiFi.gatewayIP(), WiFi.subnetMask(),
              IPAddress(8, 8, 8, 8), IPAddress(1, 1, 1, 1));

  KLOG("\nWiFi ok: %s | assoc=%lums dhcp=%lums total=%lums\n",
       WiFi.localIP().toString().c_str(), assocMs(), dhcpMs(),
       (tGotIP > tBegin) ? tGotIP - tBegin : 0);
}

bool connected() { return WiFi.status() == WL_CONNECTED; }

unsigned long assocMs() { return (tAssoc > tBegin) ? tAssoc - tBegin : 0; }
unsigned long dhcpMs() { return (tGotIP > tAssoc) ? tGotIP - tAssoc : 0; }

} // namespace wifi
} // namespace kuino
```

- [ ] **Step 3: Commit**

```bash
cd pet-platform/kuino && git add src/kuino/wifi.h src/kuino/wifi.cpp && git commit -m "feat: kuino headless wifi connect module"
```

---

### Task 5: `display` module

Extracted from rainybox `fitWidth`/`drawHeader`/`drawScroll`/`toast` (main.cpp:165-353). **Key generalisation:** rainybox used a global `oled` object and hardcoded fonts. In `kuino` the `U8G2&` and font pointers are injected as parameters, and marquee offset is caller-owned (passed in).

**Files:**
- Create: `pet-platform/kuino/src/kuino/display.h`
- Create: `pet-platform/kuino/src/kuino/display.cpp`

- [ ] **Step 1: Create `display.h`**

```cpp
#pragma once
#include <U8g2lib.h>

namespace kuino {
namespace display {

// Truncate UTF-8 `s` to fit `maxW` px under the font currently set on `oled`,
// appending ".." when truncated. Cuts on codepoint boundaries so multi-byte
// (e.g. Vietnamese) characters never split.
String fitWidth(U8G2 &oled, const String &s, int maxW);

// Draw `s` as a top header (baseline y=13). ASCII text steps a font ladder
// (vnFont -> 7x13B -> 6x12 -> 5x8) to fit `width`; non-ASCII always uses
// `vnFont` and truncates via fitWidth if still too wide.
void drawHeader(U8G2 &oled, const String &s, const uint8_t *vnFont,
                int width = 128);

// Draw `text` at baseline `y` using `vnFont`: centered if it fits `width`,
// otherwise marquee-scrolled by `marqueeOffset` px (caller owns/advances it).
void drawScroll(U8G2 &oled, int y, const String &text, const uint8_t *vnFont,
                int marqueeOffset, int width = 128);

// Clear the screen and show a one- or two-line toast, then send the buffer.
void toast(U8G2 &oled, const uint8_t *vnFont, const uint8_t *smallFont,
           const char *line1, const char *line2 = nullptr);

} // namespace display
} // namespace kuino
```

- [ ] **Step 2: Create `display.cpp`**

```cpp
#include "kuino/display.h"

namespace kuino {
namespace display {

String fitWidth(U8G2 &oled, const String &s, int maxW) {
  if (oled.getUTF8Width(s.c_str()) <= maxW) return s;
  int limit = maxW - oled.getUTF8Width("..");
  String out;
  int i = 0, n = s.length();
  while (i < n) {
    int j = i + 1;
    while (j < n && ((uint8_t)s[j] & 0xC0) == 0x80) j++; // skip continuation
    String cand = out + s.substring(i, j);
    if (oled.getUTF8Width(cand.c_str()) > limit) break;
    out = cand;
    i = j;
  }
  return out + "..";
}

void drawHeader(U8G2 &oled, const String &s, const uint8_t *vnFont, int width) {
  bool ascii = true;
  for (size_t i = 0; i < s.length(); i++)
    if ((uint8_t)s[i] & 0x80) { ascii = false; break; }

  const uint8_t *font = vnFont;
  if (ascii) {
    const uint8_t *ladder[] = {vnFont, u8g2_font_7x13B_tr, u8g2_font_6x12_tr,
                               u8g2_font_5x8_tr};
    font = ladder[3]; // smallest as fallback
    for (auto cand : ladder) {
      oled.setFont(cand);
      if (oled.getUTF8Width(s.c_str()) <= width) { font = cand; break; }
    }
  }
  oled.setFont(font);
  oled.drawUTF8(0, 13, fitWidth(oled, s, width).c_str());
}

void drawScroll(U8G2 &oled, int y, const String &text, const uint8_t *vnFont,
                int marqueeOffset, int width) {
  oled.setFont(vnFont);
  int w = oled.getUTF8Width(text.c_str());
  if (w <= width) {
    oled.drawUTF8((width - w) / 2, y, text.c_str()); // fits: center
  } else {
    int span = w + 24; // marquee with gap
    int x = -(marqueeOffset % span);
    oled.drawUTF8(x, y, text.c_str());
    oled.drawUTF8(x + span, y, text.c_str());
  }
}

void toast(U8G2 &oled, const uint8_t *vnFont, const uint8_t *smallFont,
           const char *line1, const char *line2) {
  oled.clearBuffer();
  oled.setFont(vnFont);
  oled.drawUTF8(0, 30, line1);
  if (line2) {
    oled.setFont(smallFont);
    oled.drawStr(0, 48, line2);
  }
  oled.sendBuffer();
}

} // namespace display
} // namespace kuino
```

- [ ] **Step 3: Commit**

```bash
cd pet-platform/kuino && git add src/kuino/display.h src/kuino/display.cpp && git commit -m "feat: kuino display helpers module"
```

---

### Task 6: Example sketches + compile verification

Firmware modules cannot be unit-tested off-hardware (they depend on `WiFi`/`U8g2`/`HTTPClient`); per the spec, **the acceptance test is that example sketches compile against the library.** `pio ci` compiles a source file with the local library linked.

**Files:**
- Create: `pet-platform/kuino/examples/wifi_hello/wifi_hello.ino`
- Create: `pet-platform/kuino/examples/poll_display/poll_display.ino`

- [ ] **Step 1: Create `examples/wifi_hello/wifi_hello.ino`**

```cpp
#include <Arduino.h>
#include <kuino/wifi.h>

void setup() {
  Serial.begin(115200);
  kuino::wifi::connect("your-ssid", "your-pass");
  Serial.printf("online: assoc=%lums dhcp=%lums\n", kuino::wifi::assocMs(),
                kuino::wifi::dhcpMs());
}

void loop() { delay(1000); }
```

- [ ] **Step 2: Create `examples/poll_display/poll_display.ino`** (exercises httpjson + display + button)

```cpp
#include <Arduino.h>
#include <U8g2lib.h>
#include <Wire.h>
#include <kuino/wifi.h>
#include <kuino/httpjson.h>
#include <kuino/display.h>
#include <kuino/button.h>

U8G2_SSD1306_128X64_NONAME_F_HW_I2C oled(U8G2_R0, U8X8_PIN_NONE);
#define FONT_VN u8g2_font_unifont_t_vietnamese2
#define FONT_SMALL u8g2_font_6x12_tr

int btnReading = HIGH, btnStable = HIGH;
unsigned long btnEdge = 0;
int marquee = 0;

void setup() {
  Serial.begin(115200);
  Wire.begin(8, 9);
  oled.begin();
  pinMode(4, INPUT_PULLUP);
  kuino::wifi::connect("your-ssid", "your-pass");
}

void loop() {
  if (kuino::button::pressed(4, btnReading, btnStable, btnEdge))
    kuino::display::toast(oled, FONT_VN, FONT_SMALL, ">> pressed");

  JsonDocument doc;
  if (kuino::httpjson::getJson(
          "https://rainy.fly.dev/api/v1/public/stations/live", doc)) {
    oled.clearBuffer();
    kuino::display::drawHeader(oled, "kuino demo", FONT_VN);
    kuino::display::drawScroll(oled, 48, "streaming ok", FONT_VN, marquee);
    oled.sendBuffer();
  }
  marquee += 1;
  delay(40);
}
```

- [ ] **Step 3: Compile-verify `wifi_hello`**

Run (from `pet-platform/kuino`):
```bash
pio ci examples/wifi_hello/wifi_hello.ino -l . -b esp32-s3-devkitc-1
```
Expected: ends with `[SUCCESS]`. (`pio ci` copies the sketch into a temp project, links the local lib via `-l .`, and resolves the lib's `library.json` dependencies.)

- [ ] **Step 4: Compile-verify `poll_display`**

Run:
```bash
pio ci examples/poll_display/poll_display.ino -l . -b esp32-s3-devkitc-1
```
Expected: ends with `[SUCCESS]`. If include resolution fails, confirm headers are at `src/kuino/*.h` (PlatformIO adds `src/` to the include path).

- [ ] **Step 5: Commit**

```bash
cd pet-platform/kuino && git add examples && git commit -m "test: kuino example sketches (compile gate for all modules)"
```

---

### Task 7: kuino README + CLAUDE.md, publish, tag v0.1.0

**Files:**
- Create: `pet-platform/kuino/README.md`
- Create: `pet-platform/kuino/CLAUDE.md`

- [ ] **Step 1: Create `README.md`**

Contents (fill exactly):
- One-line intro: "Shared ESP32/Arduino helpers for vukyn IoT firmware."
- **Install** section:
  ```ini
  lib_deps =
      https://github.com/vukyn/kuino.git#v0.1.0
      olikraus/U8g2@^2.35.30
      bblanchon/ArduinoJson@^7.2.0
  ```
- **Modules** table: `kuino/wifi` (headless STA connect + timing), `kuino/httpjson` (`getJson`), `kuino/display` (U8g2 text helpers), `kuino/button` (debounced press). One-line API per module copied from the header doc-comments.
- **Diagnostics**: add `-DKUINO_DIAG` to `build_flags` for verbose serial logs.
- **Versioning**: git tags, keep only the 5 newest (mirrors kuery); bump minor per change.

- [ ] **Step 2: Create `CLAUDE.md`**

Contents:
- Header: "CLAUDE.md — kuino".
- "**Not a Go platform service** — C++/Arduino PlatformIO **library** (not an app). The platform clean-arch template, gobuild presets, DI, domains, and the kuery shared-pkg rule DO NOT apply. This IS the IoT equivalent of kuery: reusable firmware code goes here, versioned, imported via `lib_deps`."
- **Stack**: PlatformIO, Arduino framework, espressif32, U8g2, ArduinoJson v7.
- **Layout**: `src/kuino/*.{h,cpp}` (public headers = include path root), `examples/` compile gates.
- **Build/verify**: `pio ci examples/<name>/<name>.ino -l . -b esp32-s3-devkitc-1`.
- **Versioning rule**: tag minor bumps; keep 5 newest tags (delete older local+remote); consumers pin `#vX.Y.Z`.

- [ ] **Step 3: Commit docs**

```bash
cd pet-platform/kuino && git add README.md CLAUDE.md && git commit -m "docs: kuino README + CLAUDE.md"
```

- [ ] **Step 4: Create the GitHub remote and push (bootstrap exception)**

> Per platform rule, commits go via PR — **except** bootstrapping a brand-new empty remote, which is this case. Confirm with the user before running, since it creates a public repo and pushes to `main`.

```bash
cd pet-platform/kuino
gh repo create vukyn/kuino --public --source=. --remote=origin --description "Shared ESP32/Arduino helpers for vukyn IoT firmware"
git branch -M main
git push -u origin main
```
Expected: repo created, `main` pushed.

- [ ] **Step 5: Tag v0.1.0**

```bash
cd pet-platform/kuino && git tag -a v0.1.0 -m "kuino v0.1.0 — wifi/httpjson/display/button" && git push origin v0.1.0
```
Expected: tag pushed. The gobuild preset (Part 2) pins this exact tag.

---

## PART 2 — gobuild `iot` preset

> All gobuild changes land via PR (committer / `/pet-commit`) — no direct pushes.

### Task 8: Gate `go mod tidy` behind `go.mod` existence

The current `generateProject` (main.go:176-185) always runs `go mod tidy`, which errors and prints noise for a non-Go preset. Fix is preset-agnostic: only run it when a `go.mod` was rendered.

**Files:**
- Modify: `gobuild/main.go:176-185`
- Test: `gobuild/gobuild_test.go`

- [ ] **Step 1: Write the failing unit test**

Add to `gobuild/gobuild_test.go`:

```go
func TestHasGoMod(t *testing.T) {
	dir := t.TempDir()
	if hasGoMod(dir) {
		t.Fatal("hasGoMod(empty dir) = true, want false")
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if !hasGoMod(dir) {
		t.Fatal("hasGoMod(dir with go.mod) = false, want true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd gobuild && go test . -run TestHasGoMod`
Expected: FAIL — `undefined: hasGoMod`.

- [ ] **Step 3: Add the helper + gate the tidy call**

In `gobuild/main.go`, add near `generateProject`:

```go
// hasGoMod reports whether the rendered project contains a go.mod (i.e. it is a
// Go preset). Non-Go presets (e.g. iot) skip the go mod tidy step.
func hasGoMod(projectDir string) bool {
	_, err := os.Stat(filepath.Join(projectDir, "go.mod"))
	return err == nil
}
```

Replace the existing tidy block (main.go:176-185) with:

```go
	// Run go mod tidy only for Go presets (those that rendered a go.mod).
	projectDir := filepath.Join(currentDir, projectName)
	if hasGoMod(projectDir) {
		goModTidyCmd := exec.Command("go", "mod", "tidy")
		goModTidyCmd.Dir = projectDir
		goModTidyCmd.Stdout = os.Stdout
		goModTidyCmd.Stderr = os.Stderr
		fmt.Println("Running go mod tidy...")
		if err := goModTidyCmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to run go mod tidy: %v\n", err)
		}
	} else {
		fmt.Println("No go.mod (non-Go preset) — skipping go mod tidy")
	}
```

(The `git init` block that follows stays unchanged and still uses `projectDir`.)

- [ ] **Step 4: Run test to verify it passes**

Run: `cd gobuild && go test . -run TestHasGoMod && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 5: Commit**

```bash
cd gobuild && git add main.go gobuild_test.go && git commit -m "feat(gobuild): gate go mod tidy behind go.mod existence for non-Go presets"
```

---

### Task 9: Create the `iot` preset template tree

Every file under a preset is run through `text/template` with `missingkey=error`. C++/ini files contain no literal `{{`, so plain `.tmpl` suffixes are fine. `gitignore.tmpl` maps to `.gitignore` via `dotfileNames`; `config.h.example.tmpl` → `config.h.example` (only `.tmpl` is stripped).

**Files:**
- Create: `gobuild/templates/iot/platformio.ini.tmpl`
- Create: `gobuild/templates/iot/src/main.cpp.tmpl`
- Create: `gobuild/templates/iot/include/config.h.example.tmpl`
- Create: `gobuild/templates/iot/gitignore.tmpl`
- Create: `gobuild/templates/iot/README.md.tmpl`
- Create: `gobuild/templates/iot/CLAUDE.md.tmpl`

- [ ] **Step 1: `platformio.ini.tmpl`**

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

- [ ] **Step 2: `src/main.cpp.tmpl`**

```cpp
// {{.ProjectName}} — ESP32-S3 firmware scaffold (gobuild iot preset)
// Consumes the shared kuino library. Fill in your device logic in loop().
#include <Arduino.h>
#include <kuino/wifi.h>
#include "config.h"

void setup() {
  Serial.begin(115200);
  kuino::wifi::connect(WIFI_SSID, WIFI_PASSWORD);
  Serial.println("{{.ProjectName}} online");
}

void loop() {
  // Add your poll/render logic here. Example building blocks from kuino:
  //   #include <kuino/httpjson.h>  -> kuino::httpjson::getJson(url, doc)
  //   #include <kuino/display.h>   -> kuino::display::drawHeader(oled, ...)
  //   #include <kuino/button.h>    -> kuino::button::pressed(pin, ...)
  delay(1000);
}
```

- [ ] **Step 3: `include/config.h.example.tmpl`**

```cpp
#pragma once
// Copy to config.h and fill in. config.h is gitignored (keeps WiFi creds out of repo).

#define WIFI_SSID       "your-wifi"
#define WIFI_PASSWORD   "your-pass"

// Backend API base (no trailing slash). Replace with your service host.
#define API_BASE        "https://example.fly.dev"

// Poll interval (ms)
#define POLL_INTERVAL_MS 10000

// OLED I2C pins (ESP32-S3). Match your wiring: SDA/SCL. Remove if headless.
#define OLED_SDA 8
#define OLED_SCL 9
```

- [ ] **Step 4: `gitignore.tmpl`** (renders to `.gitignore`)

```gitignore
# secrets — copied from include/config.h.example
include/config.h

# PlatformIO build output
.pio/
.vscode/
```

- [ ] **Step 5: `README.md.tmpl`**

```markdown
# {{.ProjectName}}

ESP32-S3 firmware scaffolded by gobuild (`iot` preset). Consumes the shared
[kuino](https://github.com/vukyn/kuino) library.

## Setup
```bash
cp include/config.h.example include/config.h   # then edit WiFi creds + API base
pio run                 # build (pulls kuino from git)
pio run -t upload       # flash
pio device monitor      # serial log (115200)
```

## kuino
Firmware helpers (WiFi, HTTPS+JSON, OLED display, buttons) live in kuino and are
pinned by tag in `platformio.ini` (`kuino.git#vX.Y.Z`). Bump the pin to adopt a
new kuino release.
```

- [ ] **Step 6: `CLAUDE.md.tmpl`**

```markdown
# CLAUDE.md — {{.ProjectName}}

ESP32-S3 firmware (gobuild `iot` preset).

**Not a Go platform service** — C++/Arduino/PlatformIO firmware. The platform
clean-arch template, gobuild Go presets, DI, domains, and the kuery shared-pkg
rule DO NOT apply here. Reusable firmware code belongs in the shared **kuino**
library (imported via `lib_deps`), not copied into this repo.

## Stack
- ESP32-S3 (espressif32), Arduino framework via PlatformIO
- Shared lib: `github.com/vukyn/kuino` (WiFi / HTTPS+JSON / U8g2 display / buttons)

## Build / flash
```bash
cp include/config.h.example include/config.h   # then edit
pio run
pio run -t upload
pio device monitor
```

## config.h
Secrets (WiFi creds, API base) live in `include/config.h` — gitignored. Copy
from `include/config.h.example`.
```

- [ ] **Step 7: Commit**

```bash
cd gobuild && git add templates/iot && git commit -m "feat(gobuild): add iot preset template tree"
```

---

### Task 10: Register the preset — usage string + golden test

**Files:**
- Modify: `gobuild/main.go:36` (usage string)
- Modify: `gobuild/gobuild_test.go:22` (`presets` slice)
- Create: `gobuild/testdata/golden/iot/**`

- [ ] **Step 1: Update the `--preset` usage string**

In `gobuild/main.go`, change the flag Usage (line ~37):
```go
				Usage:   "Project preset (base|fiber|platform-service|iot)",
```

- [ ] **Step 2: Add `iot` to the golden `presets` slice**

In `gobuild/gobuild_test.go` (line 22):
```go
var presets = []string{"base", "fiber", "platform-service", "iot"}
```

- [ ] **Step 3: Run the golden test to verify it fails (no fixtures yet)**

Run: `cd gobuild && go test . -run TestRenderPresetGolden/iot`
Expected: FAIL — tree shape mismatch (golden dir empty/missing).

- [ ] **Step 4: Generate the golden fixtures**

Run: `cd gobuild && go test . -update`
Then verify: `cd gobuild && go test ./...`
Expected: PASS. The rendered `iot` tree now lives under `testdata/golden/iot/` (e.g. `platformio.ini`, `src/main.cpp`, `include/config.h.example`, `.gitignore`, `README.md`, `CLAUDE.md`, all with `ProjectName=testproj`).

- [ ] **Step 5: Commit (force-add golden — the golden `.gitignore` self-ignores siblings)**

```bash
cd gobuild && git add main.go gobuild_test.go && git add -f testdata/golden/iot && git commit -m "test(gobuild): register iot preset + golden fixtures"
```

---

### Task 11: Docs — gobuild + platform root

**Files:**
- Modify: `gobuild/README.md`
- Modify: `gobuild/CLAUDE.md`
- Modify: `pet-platform/CLAUDE.md` (root — no git, edit only)
- Create: `pet-platform/.claude/onboarding/kuino.json`

- [ ] **Step 1: gobuild `README.md`** — add `iot` to the presets list with:
  "**iot** — minimal ESP32-S3 firmware skeleton (C++/PlatformIO, **non-Go**). Renders `platformio.ini` (single esp32-s3 env pinning the shared `kuino` lib), a thin `src/main.cpp` wired to `kuino::wifi`, `include/config.h.example`, `.gitignore`, `README.md`, `CLAUDE.md`. `go mod tidy` is skipped (no `go.mod`)."

- [ ] **Step 2: gobuild `CLAUDE.md`** — under "Presets", add the `iot` entry (same summary as Step 1). In the `--http-template`/`--preset` flag doc, change the value list to `base|fiber|platform-service|iot`. Add a note: "gobuild now emits **non-Go** presets too; `go mod tidy` runs only when a `go.mod` was rendered (`hasGoMod`), `git init` always runs."

- [ ] **Step 3: Platform root `CLAUDE.md`** — two edits:
  - Add to the repo list:
    "**`kuino/`** — Shared **firmware** library `github.com/vukyn/kuino` (C++/Arduino/PlatformIO, **first non-Go shared lib**). IoT's equivalent of kuery: WiFi connect, HTTPS+JSON client, U8g2 display helpers, button debounce. Consumed by IoT repos via `lib_deps` git URL pinned to a tag. Versioned; keep 5 newest tags."
  - Update the `gobuild/` entry: note it now also generates a **non-Go** `iot` preset (minimal ESP32/PlatformIO firmware consuming kuino) and that `/pet-onboard --new <name> iot` can scaffold firmware.
  - Add an "IoT shared-code rule (kuino)" line mirroring the kuery rule, scoped to firmware.

- [ ] **Step 4: Create `pet-platform/.claude/onboarding/kuino.json`**

```json
{
  "repo": "kuino",
  "source": "scaffold",
  "classification": "code",
  "language": "cpp",
  "platform_fit": "shared-firmware-library",
  "notes": "IoT shared library (kuery-equivalent for firmware). Not a Go service; platform clean-arch template N/A. Consumed via PlatformIO lib_deps git URL.",
  "onboarded_at": "2026-07-04"
}
```

- [ ] **Step 5: Commit the gobuild docs (root CLAUDE.md + onboarding json are outside git — leave uncommitted)**

```bash
cd gobuild && git add README.md CLAUDE.md && git commit -m "docs(gobuild): document iot preset + non-Go emission"
```

---

### Task 12: End-to-end smoke test

**Files:** none (verification only)

- [ ] **Step 1: Build gobuild and scaffold a test project**

```bash
cd gobuild && go build -o bin/gobuild . && cd /tmp && rm -rf testiot && /Users/vuky10/vukyn/repo/pet-platform/gobuild/bin/gobuild --preset iot -n testiot
```
Expected: prints "Successfully created testiot project template!", "No go.mod (non-Go preset) — skipping go mod tidy", "Initializing git repository...". Directory `/tmp/testiot` contains `platformio.ini`, `src/main.cpp`, `include/config.h.example`, `.gitignore`, `README.md`, `CLAUDE.md`.

- [ ] **Step 2: Compile the scaffolded firmware (pulls kuino from the pushed tag)**

```bash
cd /tmp/testiot && cp include/config.h.example include/config.h && pio run
```
Expected: PlatformIO downloads `kuino@v0.1.0` + U8g2 + ArduinoJson, build ends `[SUCCESS]`. (Requires network + the v0.1.0 tag pushed in Task 7.)

- [ ] **Step 3: Clean up**

```bash
rm -rf /tmp/testiot
```

- [ ] **Step 4: Open the gobuild PR**

Use the committer agent / `/pet-commit` to push the gobuild branch and open a PR covering Tasks 8-11. Verify the PR diff contains: `main.go` (gate + usage), `gobuild_test.go`, `templates/iot/**`, `testdata/golden/iot/**`, `README.md`, `CLAUDE.md`.

---

## Self-Review

**Spec coverage:**
- Deliverable A (kuino: 4 modules, library.json, versioning, KUINO_DIAG) → Tasks 1-7 ✅
- Deliverable B (iot preset: platformio.ini/main.cpp/config/gitignore/README/CLAUDE, go-mod-tidy gate, usage+README+CLAUDE) → Tasks 8-11 ✅
- Build order (kuino first, tag, then preset pins it) → Part 1 before Part 2; smoke in Task 12 ✅
- Testing (examples compile, golden test, e2e smoke) → Tasks 6, 10, 12 ✅
- Governance (5-tag retention, PR-only for gobuild, bootstrap exception for kuino, root CLAUDE.md + onboarding) → Tasks 7, 11, 12 ✅
- Out-of-scope respected: rainybox NOT refactored onto kuino; single minimal preset; no MQTT/BLE ✅

**Placeholder scan:** No TBD/TODO left as work; the `main.cpp.tmpl` loop comment is intentional scaffold guidance, not a plan placeholder. README/CLAUDE doc steps enumerate exact required content.

**Type/signature consistency:** `kuino::wifi::connect(ssid, pass, onTick=nullptr)` + `connected()`/`assocMs()`/`dhcpMs()`; `kuino::httpjson::getJson(url, doc, filter=nullptr, retries=3)`; `kuino::display::fitWidth/drawHeader/drawScroll/toast(U8G2&, ...)`; `kuino::button::pressed(pin, reading, stable, lastEdge, debounceMs=50)`; `KLOG` macro; Go `hasGoMod(projectDir) bool`. Example sketches (Task 6) and the preset `main.cpp` (Task 9) call only these exact signatures. Consistent.
