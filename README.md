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
- **KOReader** installed (`/mnt/us/koreader/fbink` required for framebuffer access)
- **Python 2.7.18** (stock on Kindle) — pure stdlib, no pip/PIL needed

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

## Project Structure

```
KindleCord/
├── bin/
│   ├── start.sh          # Full-screen launcher
│   └── stop.sh           # Cleanup
├── kindlecord/
│   ├── __init__.py
│   ├── __main__.py       # Python entry point
│   ├── main.py           # App flow, navigation, event loop
│   ├── gfx.py            # Framebuffer rendering (FbInk subprocess)
│   ├── display.py        # Thin display wrapper
│   ├── ui.py             # UI framework (Button, Label, Screen, etc.)
│   ├── input.py          # Touch input + PowerWatcher
│   ├── discord.py        # Discord REST API client
│   └── server.py         # HTTP server for token paste
├── data/                 # Runtime data (token, config)
├── config.xml            # KUAL extension metadata
├── menu.json             # KUAL menu entry
├── web/                  # (future) web UI assets
└── README.md
```

## How It Works

### Rendering

KindleCord renders directly to the framebuffer by shelling out to KOReader's **FbInk** binary (`/mnt/us/koreader/fbink`). All drawing — text, rectangles, fills — goes through `fbink` subprocess calls. There is no in-memory pixel buffer; everything is rendered on-the-fly to the hardware framebuffer.

The display is treated as a **24×24 pixel character grid** using FbInk's VGA font at 3× scale (`-S 3`), giving 60 columns × 44 rows of usable text area.

### Display Architecture

```
Python (UI) → GfxEngine.draw_text/fill_rect → subprocess fbink → /dev/fb0 → EPDC → e-ink panel
```

A full `GC16` refresh is triggered once per screen render (no partial updates).

### Touch Input

Touch events are read directly from `/dev/input/event1` (or `event0` as fallback). The input reader parses `EV_ABS` (MT position) + `EV_KEY` (BTN_TOUCH) events and translates them to UI coordinates, which are matched against component bounding boxes.

### Power Off

A `PowerWatcher` thread monitors `/dev/input/event0` for power button presses. Two presses within 500 ms trigger a clean exit (unfreeze awesome, re-enable pillow, restore the Kindle UI).

## Technical Details

| Component | Technology |
|-----------|-----------|
| Language | Python 2.7.18 (stock Kindle) |
| Rendering | FbInk (subprocess to `/mnt/us/koreader/fbink`) |
| Font | VGA 8×8 bitmap, 3× scaled (24×24 cells) |
| Display | 1448×1072 landscape (Kindle PW3) |
| Touch | `/dev/input/event1` (MT protocol) |
| Network | Discord REST API v10, HTTP server on port 8080 |
| Frame buffer | 8-bit grayscale, GC16 waveform |

## Limitations

- **Read-only (for now):** You can read messages, but sending messages and reactions are not yet wired in the UI (the Discord API client supports them).
- **No Gateway / real-time:** Messages are fetched via REST (polling). No WebSocket/presence yet.
- **No keyboard:** Token entry is done via a web page on your phone/PC.
- **Scrolling only** — no pull-to-refresh.
- **Single language:** UI is in Portuguese (Brazilian). Contributions for i18n welcome.

## License

MIT © 2026
