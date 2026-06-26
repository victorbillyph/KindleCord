# -*- coding: utf-8 -*-
"""Graphical UI framework for Kindle e-ink display."""

# ── colors ──────────────────────────────────────────────────────────
BLACK = 0x00
DARK = 0x44
MID = 0x88
LIGHT = 0xBB
WHITE = 0xFF

BTN_BG = 0x33
BTN_FG = WHITE
BTN_BORDER = 0x66
TITLE_BG = BLACK
TITLE_FG = WHITE
ITEM_BG = 0xEE
ITEM_BORDER = 0xCC
BG = WHITE
TEXT = BLACK
HINT = 0x66

_CELL = 24

# ── helpers ─────────────────────────────────────────────────────────
def _cx(cell_x):
    return cell_x * _CELL


def _cy(cell_y):
    return cell_y * _CELL


def _trunc(text, max_len):
    if len(text) <= max_len:
        return text
    return text[:max_len - 1] + "~"


# ── components ──────────────────────────────────────────────────────
class Button(object):
    """Styled button with dark background, border, and centered text."""

    def __init__(self, cx, cy, text, callback=None, width=None):
        self.cx = cx
        self.cy = cy
        self.text = text
        self.callback = callback
        self.cw = width if width else len(text) + 4
        self.w = self.cw * _CELL
        self.h = 44
        self.highlighted = False

    def render(self, display):
        x = _cx(self.cx)
        y = _cy(self.cy)
        bg = BTN_BG if not self.highlighted else 0x55
        fg = BTN_FG
        border = BTN_BORDER if not self.highlighted else WHITE

        # shadow edge (1px dark band at bottom/right for depth)
        display.fill_rect(x + 2, y + self.h, self.w - 2, 1, 0x22)
        display.fill_rect(x + self.w, y + 2, 1, self.h - 2, 0x22)
        # body
        display.fill_rect(x, y, self.w, self.h, bg)
        # border
        display.rect(x, y, self.w, self.h, border, 1)

        # centered text
        text_w = len(self.text)
        cell_cx = self.cx + (self.cw - text_w) // 2
        cell_cy = self.cy + (self.h - _CELL) // _CELL
        display.draw_text(cell_cx, cell_cy, self.text, fg, bg)

    def contains(self, px, py):
        x = _cx(self.cx)
        y = _cy(self.cy)
        return x <= px < x + self.w and y <= py < y + self.h

    def tap(self, px, py):
        if self.contains(px, py) and self.callback:
            self.callback()


class Label(object):
    """Static text."""

    def __init__(self, cx, cy, text, width=None, fg=TEXT, bg=BG):
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


class TitleBar(object):
    """Dark header bar with title text."""

    def __init__(self, title, show_back=False, back_cb=None):
        self.title = title
        self.show_back = show_back
        self.back_cb = back_cb

    def render(self, display):
        cols = display.cols
        w = display.width
        # solid dark bar
        display.fill_rect(0, 0, w, 48, TITLE_BG)
        # accent line below
        display.fill_rect(0, 47, w, 1, 0x66)
        # title text (first row)
        display.draw_text(1, 0, _trunc(self.title, cols - 2), TITLE_FG, TITLE_BG)
        # subtitle row with thin separator
        display.draw_text(1, 1, "─" * (cols - 2), 0x88, TITLE_BG)

    def tap(self, px, py):
        if self.show_back and py < 48 and self.back_cb:
            if px < 48:
                self.back_cb()


# ── app / screen base ───────────────────────────────────────────────
class App(object):
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


class Screen(object):
    def __init__(self):
        self.app = None
        self.components = []

    def on_show(self, **kwargs):
        self.components = []

    def render(self, display):
        display.engine.clear(BG)
        for comp in self.components:
            comp.render(display)
        display.refresh()

    def on_touch(self, px, py):
        for comp in self.components:
            if hasattr(comp, 'tap'):
                comp.tap(px, py)


# ── screens ─────────────────────────────────────────────────────────
class LoginScreen(Screen):
    """Login screen with centered layout and quit button at bottom."""

    def __init__(self, url, on_quit=None):
        super(LoginScreen, self).__init__()
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

        # Title
        title = "KindleCord"
        cx = (cols - len(title)) // 2
        self.components.append(Label(cx, 2, title, fg=DARK))

        # Subtitle
        sub = "Discord para Kindle"
        cx = (cols - len(sub)) // 2
        self.components.append(Label(cx, 4, sub, fg=HINT))

        # thin separator
        sep = "─" * min(cols, 40)
        cx = (cols - len(sep)) // 2
        self.components.append(Label(cx, 5, sep, fg=LIGHT))

        # URL block
        self.components.append(Label(0, 8, "Abra no celular:", fg=TEXT))
        url = self.url or "http://0.0.0.0:8080"
        cx = (cols - len(url)) // 2
        self.components.append(Label(cx, 10, url, width=cols, fg=DARK))
        self.components.append(Label(0, 12, "Cole seu token do Discord", fg=HINT))
        self.components.append(Label(0, 14, "para fazer login.", fg=HINT))
        self.components.append(Label(0, 16, "Aguardando token...", fg=DARK))

        # Quit button at bottom
        rows = d.rows
        quit_btn = Button((cols - 8) // 2, rows - 3, "Sair",
                          callback=self._on_quit, width=8)
        self.components.append(quit_btn)

    def on_touch(self, px, py):
        for comp in self.components:
            if hasattr(comp, 'tap'):
                comp.tap(px, py)


class ListScreen(Screen):
    """Scrollable list screen with styled items."""

    def __init__(self, title="", items=None, on_select=None, on_back=None,
                 back_label="Voltar", show_title_bar=True):
        super(ListScreen, self).__init__()
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
            self.components.append(TitleBar(self._title))
            row = 2

        items = self._items
        total = len(items)
        max_visible = d.rows - row - 3
        max_visible = min(max_visible, total - self._scroll)

        # scroll-up indicator
        if self._scroll > 0:
            self.components.append(_ScrollArrow(row, cols, up=True))
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
                self.components.append(_RowBg(row + visible_n, cols))
            self.components.append(Label(1, row + visible_n, " " + txt,
                                         width=cols - 2))
            visible_n += 1

        # scroll-down indicator
        if visible_end < total:
            self.components.append(_ScrollArrow(row + visible_n, cols, up=False))

        if self._on_back:
            self.components.append(
                Button(2, d.rows - 2, self._back_label,
                       callback=self._on_back, width=len(self._back_label) + 4))

    def on_touch(self, px, py):
        d = self.app.display
        start_row = 2 if self._show_title_bar else 0
        max_visible = d.rows - start_row - 3
        total = len(self._items)
        row = py // _CELL

        # back button
        if row >= d.rows - 2 and self._on_back:
            self._on_back()
            return False

        # title bar
        if row < 2 and self._show_title_bar:
            return False

        # scroll zone: top row inside content area
        content_top = start_row
        if self._scroll > 0 and row == content_top:
            self._scroll -= 1
            return True

        # scroll zone: bottom visible content row
        if self._scroll + max_visible < total:
            last_content = d.rows - 3  # one above back button
            if row == last_content:
                self._scroll += 1
                return True

        # item selection
        if start_row <= row < d.rows - 2:
            idx = self._scroll + (row - start_row)
            if 0 <= idx < total and self._on_select:
                self._on_select(idx)
                return False

        return False


class _RowBg(object):
    """Alternating row background for list items."""

    def __init__(self, cy, cols):
        self.cy = cy
        self.cols = cols

    def render(self, display):
        display.fill_rect(0, _cy(self.cy), display.width, _CELL, ITEM_BG)


class MessageScreen(Screen):
    """Screen showing messages with styled layout."""

    def __init__(self, title="", messages=None, on_back=None):
        super(MessageScreen, self).__init__()
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

        self.components.append(TitleBar(self._title))

        msgs = self._messages
        total = len(msgs)
        max_visible = (d.rows - 4) // 2
        row = 2

        # scroll-up indicator
        if self._scroll > 0:
            self.components.append(_ScrollArrow(row, cols, up=True))
            row += 2
            max_visible -= 1

        visible_end = self._scroll + max_visible
        visible_end = min(visible_end, total)
        visible_n = 0
        for i in range(self._scroll, visible_end):
            msg = msgs[i]
            author = msg.get("author", {}).get("username", "?")
            content = msg.get("content", "")
            self.components.append(Label(1, row + visible_n * 2, author,
                                         fg=DARK))
            content_line = " " + _trunc(content, cols - 3)
            self.components.append(Label(1, row + visible_n * 2 + 1,
                                         content_line, fg=TEXT))
            if i < visible_end - 1:
                self.components.append(_Divider(row + visible_n * 2 + 2,
                                                cols))
            visible_n += 1

        # scroll-down indicator
        if visible_end < total:
            sd_row = row + visible_n * 2
            self.components.append(_ScrollArrow(sd_row, cols, up=False))

        self.components.append(
            Button(2, d.rows - 2, "  Voltar  ",
                   callback=self._on_back, width=8))

    def on_touch(self, px, py):
        d = self.app.display
        total = len(self._messages)
        row = py // _CELL

        if row >= d.rows - 2 and self._on_back:
            self._on_back()
            return False
        if row < 2:
            return False

        max_visible = (d.rows - 4) // 2
        # scroll up on first content row
        if self._scroll > 0 and row == 2:
            self._scroll -= 1
            return True
        # scroll down on last content row
        if self._scroll + max_visible < total:
            last = d.rows - 3
            if row == last:
                self._scroll += 1
                return True

        return False


class _Divider(object):
    """Thin horizontal separator line."""

    def __init__(self, cy, cols):
        self.y = _cy(cy)
        self.w = cols * _CELL

    def render(self, display):
        display.fill_rect(4, self.y, self.w - 8, 1, ITEM_BORDER)


class _ScrollArrow(object):
    """Up/down arrow indicator for scrolling."""

    def __init__(self, cy, cols, up=True):
        self.cy = cy
        self.cols = cols
        self.up = up

    def render(self, display):
        y = _cy(self.cy)
        w = self.cols * _CELL
        display.fill_rect(0, y, w, _CELL, 0xEE)
        label = "/\\" if self.up else "\\/"
        cx = (self.cols - 2) // 2
        display.draw_text(cx, self.cy, label, 0x66, 0xEE)
