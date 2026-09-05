# KindleCord

A full-screen Discord client for Kindle e-ink devices (PW3, PW4, etc.).

> **DISCLAIMER:** This project is **not affiliated, associated, authorized, endorsed by, or in any way officially connected with** Discord Inc., Amazon.com, Inc., or any of their subsidiaries or affiliates. The name "Discord" is a registered trademark of Discord Inc. Use at your own risk.

## Features

- Full-screen native Discord client running directly on your Kindle
- Graphical UI rendered via framebuffer (KOReader's FbInk)
- Touch-enabled (tap to navigate, scroll lists)
- View your servers (guilds), channels, and messages
- Server selection → channel selection → message reading flow
- Safe power-off: double-press the power button to exit
- No framework hacks — runs alongside the stock Kindle UI via pillow freeze

## Screenshots

*(Coming soon — contributions welcome!)*

## Requirements

- **Jailbroken Kindle** (PW3, PW4, or similar)
- **No Python needed** — v0.2.0+ is a native Go static binary (no KOReader/fbink required, direct `/dev/fb0` mmap). Legacy Python version still available as fallback.

## Installation

1. **Copy the extension** to your Kindle via USB:

   ```
   cp -r KindleCord /mnt/us/extensions/
   ```

2. **Open KUAL** on your Kindle — "KindleCord" should appear in the menu.

3. **Tap to launch.**

4. The screen will freeze and KindleCord will display a URL like `http://192.168.x.x:8080`.

5. **Open that URL** on your phone or computer.

6. **Paste your Discord token** into the web page (see *Getting your token* below).

7. The app loads your servers and you're in!

## Getting your Discord token

1. Open Discord in a browser (not the desktop app)
2. Press `Ctrl + Shift + I` (DevTools)
3. Go to the **Console** tab
4. Type: `localStorage.getItem('token')`
5. Copy the value (it starts with `mfa_` or `ND...`)

> **⚠️ Security:** Your token is like a password. Anyone with it has full access to your account. The token is stored locally on your Kindle in `data/token.txt` and is never sent anywhere except to Discord's API.

## Usage

- **Login →** URL shown on screen, paste token via web
- **Guild list →** tap a server name to see its channels
- **Channel list →** tap a channel (`#name`) to read messages
- **Message view →** read messages, tap top/bottom to scroll
- **Scroll indicators:**
  - `/\\` at top → tap to scroll up
  - `\\/` at bottom → tap to scroll down
- **Exit →** double-press the physical power button

## Project Structure (v0.2.0 Go)

```
KindleCord/
├── cmd/kindlecord/main.go   # Entry point
├── internal/
│   ├── display/             # mmap /dev/fb0 + font 8x8@3x (no fbink)
│   ├── input/               # evdev touch + power watcher
│   ├── discord/             # Discord REST v10 (fix Bot prefix)
│   ├── server/              # net/http token server
│   └── ui/                  # Win95 UI (scroll fix)
├── kindlecord/              # Legacy Python (fallback)
├── bin/
│   ├── start.sh             # Prefers Go binary, fallback Python
│   └── stop.sh
├── build/                   # CI artifacts (kindlecord, kindlecord-arm)
├── .github/workflows/build.yml
├── data/                    # Runtime data (token, config)
├── config.xml
├── menu.json
└── README.md
```

## How It Works

### Rendering (v0.2.0 Go)

Direct `mmap("/dev/fb0")` with in-RAM buffer (no fork per draw). 8x8 bitmap font scaled 3x to 24x24 cells (60x44 grid at 1448x1072). Fallback to `fbink -W GC16` if available, else `MXCFB_SEND_UPDATE` ioctl.

### Display Architecture

```
Go (UI) → Display.FillRect/DrawText → []byte buffer → /dev/fb0 mmap → GC16 refresh → e-ink panel
```

### Touch Input

Touch events are read directly from `/dev/input/event1` (or `event0` as fallback). The input reader parses `EV_ABS` (MT position) + `EV_KEY` (BTN_TOUCH) events and translates them to UI coordinates, which are matched against component bounding boxes.

### Power Off

A `PowerWatcher` thread monitors `/dev/input/event0` for power button presses. Two presses within 500 ms trigger a clean exit (unfreeze awesome, re-enable pillow, restore the Kindle UI).

## Technical Details

| Component | Technology |
|-----------|-----------|
| Language | **Go 1.21** static binary (no Python) |
| Rendering | `mmap /dev/fb0` direct (fbink fallback) |
| Font | 8×8 bitmap, 3× scaled (24×24 cells) |
| Display | 1448×1072 landscape (Kindle PW3) |
| Touch | `/dev/input/event1` evdev, single-FD |
| Network | Discord REST API v10, `net/http` on :8080 |
| Frame buffer | 8-bit grayscale, GC16 waveform |

## Limitations

- **Read-only (for now):** You can read messages, but sending messages and reactions are not yet wired in the UI (the Discord API client supports them).
- **No Gateway / real-time:** Messages are fetched via REST (polling). No WebSocket/presence yet.
- **No keyboard:** Token entry is done via a web page on your phone/PC.
- **Scrolling only** — no pull-to-refresh.
- **Single language:** UI is in Portuguese (Brazilian). Contributions for i18n welcome.

## License

MIT © 2026
