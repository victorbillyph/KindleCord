# KindleCord

A full-screen Discord client for Kindle e-ink devices (PW3, PW4, etc.).

> **DISCLAIMER:** This project is **not affiliated, associated, authorized, endorsed by, or in any way officially connected with** Discord Inc., Amazon.com, Inc., or any of their subsidiaries or affiliates. The name "Discord" is a registered trademark of Discord Inc. Use at your own risk.

## Features

- Full-screen native Discord client on Kindle e-ink
- Anti-aliased UI — Go TTF (Go Regular/Bold 18/20px via freetype) + rounded rectangles, not 8x8 bitmap
- Framebuffer direct — offscreen 8bpp buffer, `mmap` when possible + bundled `fbink` for refresh (no KOReader required)
- Touch navigation — tap guild → channel → messages, scroll arrows
- Double-press power button to exit cleanly (unfreeze awesome, re-enable pillow)
- Web login — paste Discord token via phone/PC on `http://<kindle-ip>:8080`
- Single static Go binary — no Python, no pip

## Screenshots

*(Coming soon — contributions welcome!)*

## Requirements

- **Jailbroken Kindle** (PW3, PW4, etc.)
- No extra dependencies — `fbink` ARM binary is bundled in `bin/fbink` (KOReader optional)
- Network — Kindle and phone/PC on same Wi-Fi

## Installation

### From release (recommended)

1. Download `KindleCord.zip` from [Releases](https://github.com/victorbillyph/KindleCord/releases)
2. Extract and copy to Kindle via USB:
   ```
   cp -r KindleCord /mnt/us/extensions/
   ```
3. Eject Kindle safely.

### From source

```
git clone https://github.com/victorbillyph/KindleCord
cd KindleCord
make build-arm   # -> build/kindlecord-arm (ARM, Kindle)
make build       # -> build/kindlecord (x64, PC sim)
```

Copy `build/kindlecord-arm` to Kindle as `extensions/KindleCord/kindlecord`:
```
cp build/kindlecord-arm /run/media/<Kindle>/extensions/KindleCord/kindlecord
cp bin/fbink /run/media/<Kindle>/extensions/KindleCord/bin/fbink
```

## Usage

1. Open **KUAL** on Kindle — tap **KindleCord**.
2. Screen freezes, shows URL like `http://192.168.1.50:8080`.
3. Open that URL on phone/PC, paste Discord token, tap **Log in**.
4. Browse guilds → `Tap` a server → `Tap` a `#channel` to read messages.
5. Scroll: `/\` (top) and `\/` (bottom). Tap `Exit`/`Back` or double-press **power**.

### Getting your Discord token

1. Open Discord in a browser (not desktop app)
2. `Ctrl+Shift+I` → Console
3. `localStorage.getItem('token')`
4. Copy value (`mfa_...` or `ND...`)

> **⚠️ Security:** Token is stored at `data/token.txt` on Kindle and only sent to `discord.com/api/v10`. Treat it like a password.

## Project Structure

```
KindleCord/
├── cmd/kindlecord/main.go      # App flow, navigation, event loop
├── internal/
│   ├── display/                # Display engine
│   │   ├── display.go          # offscreen 8bpp buf + fbink/mmap + GC16
│   │   ├── font.go             # legacy 8x8 bitmap
│   │   ├── font_aa.go          # TTF AA (Go Regular/Bold, freetype 18/20px)
│   │   └── rounded.go          # FillRoundRect AA
│   ├── input/                  # evdev /dev/input/eventX + PowerWatcher
│   ├── discord/                # Discord REST v10 (correct auth, rate-limit)
│   ├── server/                 # net/http token server
│   └── ui/                     # Win95-style UI (Button/Label/TitleBar/Screens)
├── bin/
│   ├── fbink                   # bundled FBInk ARM (NiLuJe)
│   ├── start.sh                # launcher (prefers Go, fallback Python)
│   └── stop.sh
├── kindlecord/                 # legacy Python (fallback)
├── build/                      # CI artifacts
├── .github/workflows/build.yml # CI: vet + x64 + ARM
├── data/                       # runtime (token, config.example.json)
├── config.xml                  # KUAL metadata (v0.2.0)
├── menu.json
└── README.md
```

## How It Works

### Rendering

Offscreen 8bpp grayscale buffer (`1448x1072` → `60 cols × 44 rows` at `24px` cells). All drawing (`FillRect`, `FillRoundRect`, `DrawTextAA`) writes to buffer with grayscale AA (freetype `Go Regular` 18px, `Bold` 20px, `96 DPI`, `FullHinting`). On refresh, buffer is blitted to `/dev/fb0` (`mmap` if available, else `pwrite` with stride handling) then `fbink -q -s -f -W GC16 -w` triggers e-ink GC16 update. No per-draw `fork`.

```
UI (AA TTF) → Display.buf (8bpp) → /dev/fb0 (mmap/pwrite) → fbink GC16 → EPDC
```

### Bundled fbink

`bin/fbink` is checked first (`extensions/KindleCord/bin/fbink`), then `koreader/fbink`. Standalone — KOReader not required. `display.go:findFbink()` handles fallback.

### Touch & Power

Single-FD `evdev` reader (`/dev/input/event1` → fallback `event0`) parses `EV_ABS MT_POSITION_X/Y` + `EV_KEY BTN_TOUCH`. `PowerWatcher` on `event0` (non-blocking) detects double `KEY_POWER`/`KEY_SLEEP` within 500ms to exit.

### Auth

`internal/discord` uses `net/http` + `InsecureSkipVerify` (Kindle CA outdated), correct `Authorization` (no `Bot ` prefix for user tokens), `Host` header, `429` retry.

## Build & CI

```
make build      # x64 7-8M
make build-arm  # ARM 7.4M Kindle (CGO_ENABLED=0 GOARCH=arm GOARM=7)
make test       # go vet
```

GitHub Actions `.github/workflows/build.yml` (Go 1.25) builds both on push/tags/releases and uploads `KindleCord.zip`.

## Troubleshooting

- **White screen / top bar only** — old mmap without fbink. Update to `v0.2.0+` (fbink bundled).
- **Closes immediately** — was `pkill -f kindlecord` killing `kindlecord.sh`; fixed to `killall`+`pkill -x`.
- **Address already in use :8080** — `start.sh` now `killall`+`fuser -k 8080/tcp` before launch.
- **401 Invalid token** — token with quotes/spaces is trimmed; invalid is removed, tap `Exit` to retry.
- **No fbink found** — ensure `bin/fbink` is `chmod +x` (start.sh does it).

## Technical Details

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25 static binary, `-ldflags="-s -w"` |
| Rendering | Offscreen 8bpp → `/dev/fb0` + `fbink` GC16 |
| Font | Go Regular/Bold TTF 18/20px, freetype AA |
| UI | Rounded rects (r=8), AA text |
| Display | 1448×1072 landscape, 24px cells (60×44) |
| Touch | `/dev/input/event1` evdev, single FD |
| Network | Discord REST v10, `net/http` :8080 |
| Bundle | `bin/fbink` ARM included |

## Limitations

- Read-only — reading messages, no send/reactions yet (API supports it)
- REST polling, no Gateway WebSocket
- Token via web (no on-device keyboard)

## License

MIT © 2026
