# -*- coding: utf-8 -*-
import os
import fcntl
import struct
import subprocess

FBINK = "/mnt/us/koreader/fbink"
CELL = 24
# Fonts tested: IBM (default), VGA, TOPAZ, MICROKNIGHT, COZETTE
FBINK_FONT = "VGA"

_GRAY_TO_FBINK = {
    0x00: "BLACK",
    0x11: "GRAY1",
    0x22: "GRAY2",
    0x33: "GRAY3",
    0x44: "GRAY4",
    0x55: "GRAY5",
    0x66: "GRAY6",
    0x77: "GRAY7",
    0x88: "GRAY8",
    0x99: "GRAY9",
    0xAA: "GRAYA",
    0xBB: "GRAYB",
    0xCC: "GRAYC",
    0xDD: "GRAYD",
    0xEE: "GRAYE",
    0xFF: "WHITE",
}


def _gray_to_fbink(gray):
    return _GRAY_TO_FBINK.get(gray & 0xFF, "WHITE")


def _detect_fb_params():
    w, h = 1448, 1072
    stride = 1088
    bpp = 8
    try:
        with open("/sys/class/graphics/fb0/mode") as f:
            m = f.read().strip()
            if "x" in m:
                parts = m.split(":")[-1].split("x")
                if len(parts) == 2:
                    wp = parts[0]
                    hp = parts[1].split("p")[0] if "p" in parts[1] else parts[1]
                    w, h = int(wp), int(hp)
    except (IOError, OSError, ValueError):
        pass
    try:
        with open("/sys/class/graphics/fb0/line_length") as f:
            v = int(f.read().strip())
            if v > 0:
                stride = v
    except (IOError, OSError, ValueError):
        pass
    try:
        with open("/sys/class/graphics/fb0/bits_per_pixel") as f:
            bpp = int(f.read().strip())
    except (IOError, OSError, ValueError):
        pass
    # The mode gives DISPLAY dimensions (landscape). fbink coordinates
    # are in display space. Use w=1448, h=1072 for layout.
    return w, h, stride, bpp


class GfxEngine(object):
    def __init__(self):
        w, h, stride, bpp = _detect_fb_params()
        self._fb_fd = None
        try:
            fd = os.open("/dev/fb0", os.O_RDWR)
            self._fb_fd = fd
        except Exception as e:
            print("[GFX] open /dev/fb0 fail: %s" % e)

        self.width = w
        self.height = h
        self.stride = stride
        self.bpp = bpp
        self.cols = w // CELL
        self.rows = h // CELL
        self.fb_xres = w
        self.fb_yres = h

        if self._fb_fd is not None:
            print("[GFX] fb %dx%d stride=%d bpp=%d cols=%d rows=%d" %
                  (w, h, stride, bpp, self.cols, self.rows))

    def clear(self, color=0xFF):
        """Fill entire screen with color via fbink -k."""
        cname = _gray_to_fbink(color)
        cmd = [FBINK, "-q", "-k", "-B", cname, "-b"]
        try:
            subprocess.check_call(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except Exception:
            pass

    def fill_rect(self, x, y, w, h, color):
        """Fill rectangle via fbink -k with region."""
        if x < 0:
            w += x
            x = 0
        if y < 0:
            h += y
            y = 0
        if w <= 0 or h <= 0:
            return
        cname = _gray_to_fbink(color)
        region = "top=%d,left=%d,width=%d,height=%d" % (y, x, w, h)
        cmd = [FBINK, "-q", "-k", region, "-B", cname, "-b"]
        try:
            subprocess.check_call(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except Exception:
            pass

    def pixel(self, x, y, color):
        """Single pixel via fbink -k 1x1 rect."""
        cname = _gray_to_fbink(color)
        region = "top=%d,left=%d,width=1,height=1" % (y, x)
        cmd = [FBINK, "-q", "-k", region, "-B", cname, "-b"]
        try:
            subprocess.check_call(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except Exception:
            pass

    def draw_text(self, cx, cy, text, fg=0x00, bg=0xFF):
        if not text:
            return
        px = cx * CELL
        py = cy * CELL
        fg_name = _gray_to_fbink(fg)
        bg_name = _gray_to_fbink(bg)
        cmd = [FBINK, "-q", "-b", "-S", "3",
               "-F", FBINK_FONT,
               "-C", fg_name, "-B", bg_name,
               "-X", str(px), "-Y", str(py),
               text]
        try:
            subprocess.check_call(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except Exception:
            # Fallback: default IBM font
            cmd2 = [FBINK, "-q", "-b", "-S", "3",
                    "-C", fg_name, "-B", bg_name,
                    "-X", str(px), "-Y", str(py),
                    text]
            try:
                subprocess.check_call(cmd2, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            except Exception:
                pass

    def rect(self, x, y, w, h, color, thickness=1):
        for t in xrange(thickness):
            self.hline(x + t, y + t, w - 2 * t, color)
            self.hline(x + t, y + h - 1 - t, w - 2 * t, color)
            self.vline(x + t, y + t, h - 2 * t, color)
            self.vline(x + w - 1 - t, y + t, h - 2 * t, color)

    def hline(self, x, y, w, color):
        if y < 0 or y >= self.height:
            return
        x1 = max(0, x)
        x2 = min(self.width, x + w)
        if x2 <= x1:
            return
        self.fill_rect(x1, y, x2 - x1, 1, color)

    def vline(self, x, y, h, color):
        if x < 0 or x >= self.width:
            return
        y1 = max(0, y)
        y2 = min(self.height, y + h)
        if y2 <= y1:
            return
        self.fill_rect(x, y1, 1, y2 - y1, color)

    def flush(self):
        """Full-screen refresh via fbink."""
        cmd = [FBINK, "-q", "-s", "-f", "-W", "GC16", "-w"]
        try:
            subprocess.check_call(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        except Exception:
            pass

    def close(self):
        if self._fb_fd is not None:
            try:
                os.close(self._fb_fd)
            except Exception:
                pass
            self._fb_fd = None
