# -*- coding: utf-8 -*-
"""Display wrapper — delegates to pixel-based GfxEngine."""

from kindlecord.gfx import GfxEngine


class Display(object):
    def __init__(self):
        self.engine = GfxEngine()

    @property
    def width(self):
        return self.engine.width

    @property
    def height(self):
        return self.engine.height

    @property
    def cols(self):
        return self.engine.cols

    @property
    def rows(self):
        return self.engine.rows

    def clear(self):
        self.engine.clear(0xFF)
        self.engine.flush()

    def refresh(self):
        self.engine.flush()

    def pixel(self, x, y, color):
        self.engine.pixel(x, y, color)

    def fill_rect(self, x, y, w, h, color):
        self.engine.fill_rect(x, y, w, h, color)

    def rect(self, x, y, w, h, color, thickness=1):
        self.engine.rect(x, y, w, h, color, thickness)

    def draw_text(self, cx, cy, text, fg=0x00, bg=0xFF):
        self.engine.draw_text(cx, cy, text, fg, bg)

    def hline(self, x, y, w, color):
        self.engine.hline(x, y, w, color)

    def vline(self, x, y, h, color):
        self.engine.vline(x, y, h, color)

    def invert_rect(self, x, y, w, h):
        self.engine.fill_rect(x, y, w, h, 0x00)

    def close(self):
        self.engine.close()
