# CLAUDE.md — testproj

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
