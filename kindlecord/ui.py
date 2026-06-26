# -*- coding: utf-8 -*-
"""Windows 95 themed UI framework for Kindle e-ink display."""

# ── Win95 grayscale palette ───────────────────────────────────────
DESKTOP = 0x55      # teal equivalent
W95_GRAY = 0xBB     # button face (C0C0C0 in color)
W95_DARK = 0x44     # shadow / dark border
W95_LIGHT = 0xDD    # highlight / light border
W95_BLUE = 0x33     # title bar blue equivalent
W95_BLACK = 0x00
W95_WHITE = 0xFF

_CELL = 24

# ── helpers ───────────────────────────────────────────────────────
def _cx(cell_x):
    return cell_x * _CELL

def _cy(cell_y):
    return cell_y * _CELL

def _trunc(text, max_len):
    if len(text) <= max_len:
        return text
    return text[:max_len - 1] + "~"

# ── 3D bevel primitives ──────────────────────────────────────────
def _bevel_up(display, x, y, w, h):
    """Raised bevel (button up)."""
    display.hline(x, y, w, W95_WHITE)
    display.vline(x, y, h, W95_WHITE)
    display.hline(x, y + h, w, W95_DARK)
    display.vline(x + w, y, h, W95_DARK)

def _bevel_down(display, x, y, w, h):
    """Sunken bevel (button down / pressed)."""
    display.hline(x, y, w, W95_DARK)
    display.vline(x, y, h, W95_DARK)
    display.hline(x, y + h, w, W95_WHITE)
    display.vline(x + w, y, h, W95_WHITE)

def _bevel_frame(display, x, y, w, h):
    """Window frame bevel: outer dark, inner white."""
    display.hline(x, y, w, W95_DARK)
    display.vline(x, y, h, W95_DARK)
    display.hline(x + 1, y + 1, w - 2, W95_WHITE)
    display.vline(x + 1, y + 1, h - 2, W95_WHITE)
    display.hline(x, y + h, w, W95_WHITE)
    display.vline(x + w, y, h, W95_WHITE)
    display.hline(x + 1, y + h - 1, w - 2, W95_DARK)
    display.vline(x + w - 1, y + 1, h - 2, W95_DARK)

# ── components ────────────────────────────────────────────────────
class Button95(object):
    """Win95-style 3D bevel button."""

    def __init__(self, cx, cy, text, callback=None, width=None):
        self.cx = cx
        self.cy = cy
        self.text = text
        self.callback = callback
        self.cw = width if width else len(text) + 4
        self.w = self.cw * _CELL
        self.h = 44
        self.pressed = False

    def render(self, display):
        x = _cx(self.cx)
        y = _cy(self.cy)
        bg = W95_GRAY
        fg = W95_BLACK

        # face
        display.fill_rect(x, y, self.w, self.h, bg)
        # bevel
        if self.pressed:
            _bevel_down(display, x, y, self.w, self.h)
            ox, oy = 1, 1
        else:
            _bevel_up(display, x, y, self.w, self.h)
            ox, oy = 0, 0

        # centered text
        text_w = len(self.text)
        cell_cx = self.cx + (self.cw - text_w) // 2
        cell_cy = self.cy + (self.h - _CELL) // _CELL
        display.draw_text(cell_cx + ox, cell_cy + oy, self.text, fg, bg)

    def contains(self, px, py):
        x = _cx(self.cx)
        y = _cy(self.cy)
        return x <= px < x + self.w and y <= py < y + self.h

    def tap(self, px, py):
        if self.contains(px, py) and self.callback:
            self.callback()


class Label95(object):
    """Static text."""

    def __init__(self, cx, cy, text, width=None, fg=W95_BLACK, bg=W95_WHITE):
        self.cx = cx
        self.cy = cy
        self.text = text
        self.width = width
        self.fg = fg
        self.bg = bg

    def render(self, display):
        txt = self.text
        if self.width and len(txt) > self.width:
            txt = _trunc(txt, self.width)
        display.draw_text(self.cx, self.cy, txt, self.fg, self.bg)


class TitleBar95(object):
    """Win95 window title bar (blue, white text, close button)."""

    BAR_H = 48

    def __init__(self, title, on_close=None):
        self.title = title
        self.on_close = on_close

    def render(self, display):
        cols = display.cols
        w = display.width
        # blue bar
        display.fill_rect(0, 0, w, self.BAR_H, W95_BLUE)
        # inset line at bottom for depth
        display.fill_rect(0, self.BAR_H - 1, w, 1, W95_BLACK)
        # title
        display.draw_text(1, 0, _trunc(self.title, cols - 4), W95_WHITE, W95_BLUE)
        display.draw_text(1, 1, _trunc(self.title, cols - 4), W95_WHITE, W95_BLUE)
        # close button (X) on the right
        close_x = cols - 4
        display.fill_rect(_cx(close_x), 4, 3 * _CELL, self.BAR_H - 10, W95_GRAY)
        _bevel_up(display, _cx(close_x), 4, 3 * _CELL, self.BAR_H - 10)
        display.draw_text(close_x + 1, 0, "X", W95_BLACK, W95_GRAY)
        self._close_rect = (_cx(close_x), 4, 3 * _CELL, self.BAR_H - 10)

    def tap(self, px, py):
        if self.on_close and hasattr(self, '_close_rect'):
            x, y, w, h = self._close_rect
            if x <= px < x + w and y <= py < y + h:
                self.on_close()


# ── app / screen base ─────────────────────────────────────────────
class App95(object):
    def __init__(self, display, discord):
        self.display = display
        self.discord = discord
        self.screens = {}
        self.current = None
        self.running = True

    def add(self, name, screen):
        screen.app = self
        self.screens[name] = screen

    def show(self, name, **kwargs):
        self.current = name
        scr = self.screens[name]
        if hasattr(scr, 'on_show'):
            scr.on_show(**kwargs)
        scr.render(self.display)

    def touch(self, px, py):
        if self.current:
            scr = self.screens[self.current]
            if hasattr(scr, 'on_touch'):
                if scr.on_touch(px, py):
                    scr.render(self.display)

    def stop(self):
        self.running = False


class Screen95(object):
    def __init__(self):
        self.app = None
        self.components = []

    def on_show(self, **kwargs):
        self.components = []

    def render(self, display):
        display.engine.clear(DESKTOP)
        # small frame margin (2 cells top, 1 cell sides)
        margin_x = _CELL
        margin_y = 0
        cx = margin_x
        cy = margin_y
        cw = display.width - margin_x * 2
        ch = display.height - margin_y
        display.fill_rect(cx, cy, cw, ch, W95_WHITE)
        _bevel_frame(display, cx, cy, cw, ch)
        for comp in self.components:
            comp.render(display)
        display.refresh()

    def on_touch(self, px, py):
        for comp in self.components:
            if hasattr(comp, 'tap'):
                comp.tap(px, py)


# ── screens ───────────────────────────────────────────────────────
class LoginScreen95(Screen95):
    """Login window on teal desktop."""

    def __init__(self, url, on_quit=None):
        super(LoginScreen95, self).__init__()
        self.url = url
        self._on_quit = on_quit

    def on_show(self, **kwargs):
        if 'url' in kwargs:
            self.url = kwargs['url']
        if 'on_quit' in kwargs:
            self._on_quit = kwargs['on_quit']
        self._build_components()

    def _build_components(self):
        self.components = []
        d = self.app.display
        cols = d.cols

        self.components.append(TitleBar95("KindleCord Login",
                                 on_close=self._on_quit))

        y = 4
        self.components.append(Label95(2, y, "Abra no celular:"))
        y += 2
        url = self.url or "http://0.0.0.0:8080"
        cx = (cols - len(url)) // 2
        self.components.append(Label95(cx, y, url, width=cols - 4, fg=W95_BLUE))
        y += 3
        self.components.append(Label95(2, y, "Cole seu token do Discord"))
        y += 2
        self.components.append(Label95(2, y, "para fazer login."))
        y += 2
        self.components.append(Label95(2, y, "Aguardando token..."))

        # buttons at bottom
        btn_y = d.rows - 4
        self.components.append(
            Button95((cols - 8) // 2, btn_y, "Sair",
                     callback=self._on_quit, width=8))

    def on_touch(self, px, py):
        for comp in self.components:
            if hasattr(comp, 'tap'):
                comp.tap(px, py)


class ListScreen95(Screen95):
    """Scrollable list with Win95 styling."""

    def __init__(self, title="", items=None, on_select=None, on_back=None,
                 back_label="Voltar", show_title_bar=True):
        super(ListScreen95, self).__init__()
        self._title = title
        self._items = items or []
        self._on_select = on_select
        self._on_back = on_back
        self._back_label = back_label
        self._scroll = 0
        self._show_title_bar = show_title_bar

    def on_show(self, **kwargs):
        for k in ('items', 'title', 'on_select', 'on_back'):
            if k in kwargs:
                setattr(self, '_' + k, kwargs[k])
        self._scroll = 0
        self._build_components()

    def _build_components(self):
        self.components = []
        d = self.app.display
        cols = d.cols
        row = 0

        if self._show_title_bar:
            self.components.append(TitleBar95(self._title,
                                     on_close=self._on_back))
            row = 2

        items = self._items
        total = len(items)
        max_visible = d.rows - row - 4
        max_visible = min(max_visible, total - self._scroll)

        if self._scroll > 0:
            self.components.append(_ScrollArrow95(row, cols, up=True))
            row += 1
            max_visible -= 1

        visible_end = self._scroll + max_visible
        visible_n = 0
        for i in range(max_visible):
            idx = self._scroll + i
            if idx >= total:
                break
            txt = items[idx]
            if visible_n % 2 == 0:
                self.components.append(_RowBg95(row + visible_n, cols))
            self.components.append(Label95(2, row + visible_n, "  " + txt,
                                           width=cols - 4))
            visible_n += 1

        if visible_end < total:
            self.components.append(_ScrollArrow95(row + visible_n, cols,
                                                  up=False))

        if self._on_back:
            self.components.append(
                Button95(2, d.rows - 3, self._back_label,
                         callback=self._on_back,
                         width=len(self._back_label) + 4))

    def on_touch(self, px, py):
        d = self.app.display
        start_row = 2 if self._show_title_bar else 0
        max_visible = d.rows - start_row - 4
        total = len(self._items)
        row = py // _CELL

        if row >= d.rows - 3 and self._on_back:
            self._on_back()
            return False
        if row < 2 and self._show_title_bar:
            for comp in self.components:
                if hasattr(comp, 'tap'):
                    comp.tap(px, py)
            return False

        if self._scroll > 0 and row == start_row:
            self._scroll -= 1
            return True

        if self._scroll + max_visible < total:
            last = d.rows - 4
            if row == last:
                self._scroll += 1
                return True

        if start_row <= row < d.rows - 3:
            idx = self._scroll + (row - start_row)
            if 0 <= idx < total and self._on_select:
                self._on_select(idx)
                return False

        return False


class MessageScreen95(Screen95):
    """Messages with Win95 styling."""

    def __init__(self, title="", messages=None, on_back=None):
        super(MessageScreen95, self).__init__()
        self._title = title
        self._messages = messages or []
        self._on_back = on_back
        self._scroll = 0

    def on_show(self, **kwargs):
        for k in ('messages', 'title', 'on_back'):
            if k in kwargs:
                setattr(self, '_' + k, kwargs[k])
        self._scroll = 0
        self._build_components()

    def _build_components(self):
        self.components = []
        d = self.app.display
        cols = d.cols

        self.components.append(TitleBar95("#" + self._title,
                                 on_close=self._on_back))

        msgs = self._messages
        total = len(msgs)
        max_visible = (d.rows - 5) // 2
        row = 2

        if self._scroll > 0:
            self.components.append(_ScrollArrow95(row, cols, up=True))
            row += 2
            max_visible -= 1

        visible_end = self._scroll + max_visible
        visible_end = min(visible_end, total)
        visible_n = 0
        for i in range(self._scroll, visible_end):
            msg = msgs[i]
            author = msg.get("author", {}).get("username", "?")
            content = msg.get("content", "")
            self.components.append(Label95(2, row + visible_n * 2,
                                           author, fg=W95_BLUE))
            content_line = "  " + _trunc(content, cols - 5)
            self.components.append(Label95(2, row + visible_n * 2 + 1,
                                           content_line))
            if i < visible_end - 1:
                self.components.append(
                    _Divider95(row + visible_n * 2 + 2, cols))
            visible_n += 1

        if visible_end < total:
            sd_row = row + visible_n * 2
            self.components.append(_ScrollArrow95(sd_row, cols, up=False))

        self.components.append(
            Button95(2, d.rows - 3, "  OK  ",
                     callback=self._on_back, width=6))

    def on_touch(self, px, py):
        d = self.app.display
        total = len(self._messages)
        row = py // _CELL

        if row >= d.rows - 3 and self._on_back:
            self._on_back()
            return False
        if row < 2:
            for comp in self.components:
                if hasattr(comp, 'tap'):
                    comp.tap(px, py)
            return False

        max_visible = (d.rows - 5) // 2
        if self._scroll > 0 and row == 2:
            self._scroll -= 1
            return True
        if self._scroll + max_visible < total:
            last = d.rows - 4
            if row == last:
                self._scroll += 1
                return True

        return False


class Dialog95(Screen95):
    """Win95-style modal dialog box."""

    def __init__(self, title, message, on_ok=None):
        super(Dialog95, self).__init__()
        self._title = title
        self._message = message
        self._on_ok = on_ok

    def on_show(self, **kwargs):
        for k in ('title', 'message', 'on_ok'):
            if k in kwargs:
                setattr(self, '_' + k, kwargs[k])
        self._build_components()

    def _build_components(self):
        self.components = []
        d = self.app.display
        cols = d.cols

        self.components.append(TitleBar95(self._title,
                                 on_close=self._on_ok))

        lines = self._message.split("\n")
        y = 4
        for line in lines:
            cx = (cols - len(line)) // 2
            self.components.append(Label95(max(2, cx), y, line))
            y += 2

        btn_y = y + 1
        self.components.append(
            Button95((cols - 6) // 2, btn_y, "  OK  ",
                     callback=self._on_ok, width=6))


# ── internal helpers ──────────────────────────────────────────────
class _RowBg95(object):
    def __init__(self, cy, cols):
        self.cy = cy
        self.cols = cols

    def render(self, display):
        display.fill_rect(_CELL, _cy(self.cy),
                          display.width - _CELL * 2, _CELL, 0xF0)


class _Divider95(object):
    def __init__(self, cy, cols):
        self.y = _cy(cy)
        self.w = cols * _CELL

    def render(self, display):
        display.fill_rect(_CELL, self.y, display.width - _CELL * 2, 1,
                          W95_DARK)


class _ScrollArrow95(object):
    def __init__(self, cy, cols, up=True):
        self.cy = cy
        self.cols = cols
        self.up = up

    def render(self, display):
        y = _cy(self.cy)
        w = display.width
        display.fill_rect(_CELL, y, w - _CELL * 2, _CELL, W95_GRAY)
        _bevel_down(display, _CELL, y, w - _CELL * 2, _CELL)
        label = "/\\" if self.up else "\\/"
        cx = (self.cols - 2) // 2
        display.draw_text(cx, self.cy, label, W95_BLACK, W95_GRAY)
