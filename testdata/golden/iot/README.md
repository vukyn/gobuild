# testproj

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
