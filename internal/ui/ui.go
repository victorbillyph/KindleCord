package ui

import (
	"strings"

	"kindlecord/internal/display"
)

const (
	Desktop  = 0x55
	W95Gray  = 0xBB
	W95Dark  = 0x44
	W95Light = 0xDD
	W95Blue  = 0x33
	W95Black = 0x00
	W95White = 0xFF
)

const cell = display.CellSize

func cx(c int) int { return c * cell }
func cy(c int) int { return c * cell }

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "~"
	}
	return s[:max-1] + "~"
}

func bevelUp(d *display.Display, x, y, w, h int) {
	// Rounded bevel for less pixelated look
	if d.IsSimulate() || w < 40 {
		// fallback to classic bevel on sim
		d.HLine(x, y, w, W95White)
		d.VLine(x, y, h, W95White)
		d.HLine(x, y+h, w, W95Dark)
		d.VLine(x+w, y, h, W95Dark)
		return
	}
	// Rounded button: fill with rounded rect + soft shadow
	r := 8
	if h < 20 {
		r = 4
	}
	d.FillRoundRect(x, y, w, h, r, W95Gray)
	// highlight top edge
	d.FillRect(x+r, y, w-r*2, 2, W95White)
	d.FillRect(x, y+r, 2, h-r*2, W95White)
	// shadow bottom
	d.FillRect(x+r, y+h-2, w-r*2, 2, W95Dark)
	d.FillRect(x+w-2, y+r, 2, h-r*2, W95Dark)
}

func bevelDown(d *display.Display, x, y, w, h int) {
	if d.IsSimulate() || w < 40 {
		d.HLine(x, y, w, W95Dark)
		d.VLine(x, y, h, W95Dark)
		d.HLine(x, y+h, w, W95White)
		d.VLine(x+w, y, h, W95White)
		return
	}
	r := 8
	if h < 20 {
		r = 4
	}
	d.FillRoundRect(x, y, w, h, r, W95Gray)
	d.FillRect(x+r, y, w-r*2, 2, W95Dark)
	d.FillRect(x, y+r, 2, h-r*2, W95Dark)
}

// Component interface
type Component interface {
	Render(d *display.Display)
	Tap(px, py int) bool // returns true if handled
	Contains(px, py int) bool
}

// Button95
type Button95 struct {
	CX, CY   int
	Text     string
	Callback func()
	CW       int
	W, H     int
	Pressed  bool
}

func NewButton(cx_, cy_ int, text string, cb func(), width int) *Button95 {
	cw := width
	if cw == 0 {
		cw = len(text) + 4
	}
	return &Button95{CX: cx_, CY: cy_, Text: text, Callback: cb, CW: cw, W: cw * cell, H: 44}
}

func (b *Button95) Render(d *display.Display) {
	x := cx(b.CX)
	y := cy(b.CY)
	bg := uint8(W95Gray)
	fg := uint8(W95Black)
	d.FillRect(x, y, b.W, b.H, bg)
	if b.Pressed {
		bevelDown(d, x, y, b.W, b.H)
		d.DrawText(b.CX+(b.CW-len(b.Text))/2+1, b.CY+1, b.Text, fg, bg)
	} else {
		bevelUp(d, x, y, b.W, b.H)
		d.DrawText(b.CX+(b.CW-len(b.Text))/2, b.CY, b.Text, fg, bg)
	}
}
func (b *Button95) Contains(px, py int) bool {
	x := cx(b.CX)
	y := cy(b.CY)
	return px >= x && px < x+b.W && py >= y && py < y+b.H
}
func (b *Button95) Tap(px, py int) bool {
	if b.Contains(px, py) && b.Callback != nil {
		b.Callback()
		return true
	}
	return false
}

// Label95
type Label95 struct {
	CX, CY int
	Text   string
	Width  int
	FG, BG uint8
}

func NewLabel(cx_, cy_ int, text string, width int, fg, bg uint8) *Label95 {
	return &Label95{CX: cx_, CY: cy_, Text: text, Width: width, FG: fg, BG: bg}
}
func (l *Label95) Render(d *display.Display) {
	txt := l.Text
	if l.Width > 0 && len(txt) > l.Width {
		txt = trunc(txt, l.Width)
	}
	d.DrawText(l.CX, l.CY, txt, l.FG, l.BG)
}
func (l *Label95) Contains(px, py int) bool { return false }
func (l *Label95) Tap(px, py int) bool      { return false }

// TitleBar95
type TitleBar95 struct {
	Title   string
	OnClose func()
	rect    [4]int
	BarH    int
}

func NewTitleBar(title string, onClose func()) *TitleBar95 {
	return &TitleBar95{Title: title, OnClose: onClose, BarH: 48}
}
func (t *TitleBar95) Render(d *display.Display) {
	cols := d.Cols
	w := d.Width
	d.FillRect(0, 0, w, t.BarH, W95Blue)
	d.FillRect(0, t.BarH-1, w, 1, W95Black)
	d.DrawText(1, 0, trunc(t.Title, cols-4), W95White, W95Blue)
	closeX := cols - 4
	t.rect = [4]int{cx(closeX), 4, 3 * cell, t.BarH - 10}
	d.FillRect(t.rect[0], t.rect[1], t.rect[2], t.rect[3], W95Gray)
	bevelUp(d, t.rect[0], t.rect[1], t.rect[2], t.rect[3])
	d.DrawText(closeX+1, 0, "X", W95Black, W95Gray)
}
func (t *TitleBar95) Contains(px, py int) bool {
	x, y, w, h := t.rect[0], t.rect[1], t.rect[2], t.rect[3]
	return px >= x && px < x+w && py >= y && py < y+h
}
func (t *TitleBar95) Tap(px, py int) bool {
	if t.OnClose != nil && t.Contains(px, py) {
		t.OnClose()
		return true
	}
	return false
}

// ModernHeader - cleaner, rounded
type ModernHeader struct {
	Title   string
	OnClose func()
	rect    [4]int
	BarH    int
}

func NewModernHeader(title string, onClose func()) *ModernHeader {
	return &ModernHeader{Title: title, OnClose: onClose, BarH: 48}
}
func (m *ModernHeader) Render(d *display.Display) {
	w := d.Width
	// soft header with rounded bottom
	d.FillRect(0, 0, w, m.BarH, W95Blue)
	d.FillRoundRect(0, m.BarH-8, w, 16, 8, W95Blue)
	d.DrawText(1, 0, trunc(m.Title, d.Cols-4), W95White, W95Blue)
	closeX := d.Cols - 4
	m.rect = [4]int{cx(closeX), 4, 3 * cell, m.BarH - 10}
	d.FillRoundRect(m.rect[0], m.rect[1], m.rect[2], m.rect[3], 8, W95Gray)
	bevelUp(d, m.rect[0], m.rect[1], m.rect[2], m.rect[3])
	d.DrawText(closeX+1, 0, "X", W95Black, W95Gray)
}
func (m *ModernHeader) Contains(px, py int) bool {
	x, y, w, h := m.rect[0], m.rect[1], m.rect[2], m.rect[3]
	return px >= x && px < x+w && py >= y && py < y+h
}
func (m *ModernHeader) Tap(px, py int) bool {
	if m.OnClose != nil && m.Contains(px, py) {
		m.OnClose()
		return true
	}
	return false
}

// Card - white rounded container (simple)
type Card struct {
	x, y, w, h, radius int
}

func (c *Card) Render(d *display.Display) {
	d.FillRect(c.x, c.y, c.w, c.h, W95White)
	d.Rect(c.x, c.y, c.w, c.h, W95Dark, 1)
}
func (c *Card) Contains(px, py int) bool { return false }
func (c *Card) Tap(px, py int) bool      { return false }

// Box - rounded bordered box for URL/highlight
type Box struct {
	x, y, w, h, radius int
	bg, border         uint8
}

func (b *Box) Render(d *display.Display) {
	d.FillRoundRect(b.x, b.y, b.w, b.h, b.radius, b.border)
	d.FillRoundRect(b.x+2, b.y+2, b.w-4, b.h-4, b.radius-2, b.bg)
}
func (b *Box) Contains(px, py int) bool { return false }
func (b *Box) Tap(px, py int) bool      { return false }

// Internal helpers
type rowBg struct{ cy, cols int }
func (r *rowBg) Render(d *display.Display) {
	// subtle rounded row
	d.FillRoundRect(cell+2, cy(r.cy)+1, d.Width-cell*2-4, cell-2, 6, 0xF2)
}
func (r *rowBg) Contains(px, py int) bool { return false }
func (r *rowBg) Tap(px, py int) bool      { return false }

type divider struct{ y, w int }
func (d2 *divider) Render(d *display.Display) { d.FillRect(cell, d2.y, d.Width-cell*2, 1, W95Dark) }
func (d2 *divider) Contains(px, py int) bool { return false }
func (d2 *divider) Tap(px, py int) bool      { return false }

type scrollArrow struct{ cy, cols int; up bool }
func (s *scrollArrow) Render(d *display.Display) {
	y := cy(s.cy)
	w := d.Width
	d.FillRoundRect(cell+4, y+2, w-cell*2-8, cell-4, 8, W95Gray)
	bevelUp(d, cell+4, y+2, w-cell*2-8, cell-4)
	label := "  \\/  "
	if s.up {
		label = "  /\\  "
	}
	cx_ := (s.cols - len(label)/2) / 2
	d.DrawText(cx_, s.cy, label, W95Black, W95Gray)
}
func (s *scrollArrow) Contains(px, py int) bool { return false }
func (s *scrollArrow) Tap(px, py int) bool      { return false }

// App95 manages screens
type App95 struct {
	Display *display.Display
	Screens map[string]Screen
	Current string
	Running bool
}

type Screen interface {
	SetApp(a *App95)
	Render(d *display.Display)
	OnTouch(px, py int) bool // true = needs re-render
	OnShow(args map[string]interface{})
}

func NewApp(d *display.Display) *App95 {
	return &App95{Display: d, Screens: make(map[string]Screen), Running: true}
}
func (a *App95) Add(name string, s Screen) {
	s.SetApp(a)
	a.Screens[name] = s
}
func (a *App95) Show(name string, args map[string]interface{}) {
	a.Current = name
	scr := a.Screens[name]
	if scr != nil {
		scr.OnShow(args)
		scr.Render(a.Display)
	}
}
func (a *App95) Touch(px, py int) {
	if a.Current == "" {
		return
	}
	scr := a.Screens[a.Current]
	if scr != nil {
		if scr.OnTouch(px, py) {
			scr.Render(a.Display)
		}
	}
}
func (a *App95) Stop() { a.Running = false }

// BaseScreen
type BaseScreen struct {
	App        *App95
	Components []Component
}
func (b *BaseScreen) SetApp(a *App95) { b.App = a }
func (b *BaseScreen) Render(d *display.Display) {
	d.Clear(W95White)
	for _, c := range b.Components {
		c.Render(d)
	}
	d.Refresh()
}
func (b *BaseScreen) OnShow(args map[string]interface{}) { b.Components = nil }
func (b *BaseScreen) OnTouch(px, py int) bool {
	for _, c := range b.Components {
		if c.Tap(px, py) {
			return false
		}
	}
	return false
}

// LoginScreen95
type LoginScreen95 struct {
	BaseScreen
	URL     string
	SSHInfo string
	OnQuit  func()
}

func NewLoginScreen(url string, onQuit func()) *LoginScreen95 {
	return &LoginScreen95{URL: url, OnQuit: onQuit}
}
func (s *LoginScreen95) OnShow(args map[string]interface{}) {
	if v, ok := args["url"]; ok {
		if u, ok := v.(string); ok {
			s.URL = u
		}
	}
	if v, ok := args["ssh_info"]; ok {
		if u, ok := v.(string); ok {
			s.SSHInfo = u
		}
	}
	if v, ok := args["on_quit"]; ok {
		if f, ok := v.(func()); ok {
			s.OnQuit = f
		}
	}
	s.build()
}
func (s *LoginScreen95) build() {
	if s.App == nil {
		return
	}
	d := s.App.Display
	cols := d.Cols
	s.Components = nil
	s.Components = append(s.Components, NewTitleBar("KindleCord Setup", s.OnQuit))
	y := 4
	s.Components = append(s.Components, NewLabel(2, y, "Open on your phone:", 0, W95Black, W95White))
	y += 2
	url := s.URL
	if url == "" {
		url = "http://0.0.0.0:8080"
	}
	cx_ := (cols - len(url)) / 2
	if cx_ < 0 {
		cx_ = 0
	}
	s.Components = append(s.Components, NewLabel(cx_, y, url, cols-4, W95Blue, W95White))
	y += 2
	if s.SSHInfo != "" {
		sshCx := (cols - len(s.SSHInfo)) / 2
		if sshCx < 0 {
			sshCx = 0
		}
		s.Components = append(s.Components, NewLabel(sshCx, y, s.SSHInfo, cols-4, W95Dark, W95White))
		y += 2
	} else {
		y += 1
	}
	s.Components = append(s.Components, NewLabel(2, y, "Paste your Discord token", 0, W95Black, W95White))
	y += 2
	s.Components = append(s.Components, NewLabel(2, y, "to log in.", 0, W95Black, W95White))
	y += 2
	s.Components = append(s.Components, NewLabel(2, y, "Waiting for token...", 0, W95Black, W95White))
	btnY := d.Rows - 3
	s.Components = append(s.Components, NewButton((cols-8)/2, btnY, "Exit", s.OnQuit, 8))
}
func (s *LoginScreen95) Render(d *display.Display) { s.BaseScreen.Render(d) }
func (s *LoginScreen95) OnTouch(px, py int) bool {
	for _, c := range s.Components {
		c.Tap(px, py)
	}
	return false
}

// ListScreen95
type ListScreen95 struct {
	BaseScreen
	Title         string
	Items         []string
	OnSelect      func(idx int)
	OnBack        func()
	BackLabel     string
	ShowTitle     bool
	scroll        int
	scrollHasUp   bool
	scrollHasDown bool
	scrollVisible int
	scrollRowStart int
}

func NewListScreen(title string, items []string, onSelect func(int), onBack func(), backLabel string, showTitle bool) *ListScreen95 {
	if backLabel == "" {
		backLabel = "Back"
	}
	return &ListScreen95{Title: title, Items: items, OnSelect: onSelect, OnBack: onBack, BackLabel: backLabel, ShowTitle: showTitle}
}
func (s *ListScreen95) OnShow(args map[string]interface{}) {
	if v, ok := args["items"]; ok {
		if vv, ok := v.([]string); ok {
			s.Items = vv
		}
	}
	if v, ok := args["title"]; ok {
		if vv, ok := v.(string); ok {
			s.Title = vv
		}
	}
	if v, ok := args["on_select"]; ok {
		if vv, ok := v.(func(int)); ok {
			s.OnSelect = vv
		}
	}
	if v, ok := args["on_back"]; ok {
		if vv, ok := v.(func()); ok {
			s.OnBack = vv
		}
	}
	s.scroll = 0
	s.build()
}
func (s *ListScreen95) build() {
	if s.App == nil {
		return
	}
	d := s.App.Display
	cols := d.Cols
	s.Components = nil
	row := 0
	if s.ShowTitle {
		s.Components = append(s.Components, NewModernHeader(s.Title, s.OnBack))
		row = 2
		// subtle card behind list
		s.Components = append(s.Components, &Card{ x: cell, y: cy(row)-4, w: d.Width-cell*2, h: (d.Rows-row-3)*cell, radius: 12 })
	}
	total := len(s.Items)
	visibleRows := d.Rows - row - 4
	if visibleRows < 0 {
		visibleRows = 0
	}
	hasUp := s.scroll > 0
	hasDown := false
	// reserve space for arrows
	if hasUp {
		visibleRows--
	}
	// compute if we need down arrow
	if s.scroll+visibleRows < total {
		hasDown = true
		if visibleRows > 0 {
			// down arrow takes one row
			// check after accounting
			if s.scroll+visibleRows < total {
				// keep as is, we will reserve
			}
		}
		if hasDown {
			visibleRows--
			if visibleRows < 0 {
				visibleRows = 0
			}
		}
	}
	if hasUp {
		s.Components = append(s.Components, &scrollArrow{cy: row, cols: cols, up: true})
		row++
	}
	visEnd := s.scroll + visibleRows
	if visEnd > total {
		visEnd = total
	}
	for i := s.scroll; i < visEnd; i++ {
		vi := i - s.scroll
		if vi%2 == 0 {
			s.Components = append(s.Components, &rowBg{cy: row + vi, cols: cols})
		}
		txt := "  " + s.Items[i]
		s.Components = append(s.Components, NewLabel(2, row+vi, txt, cols-4, W95Black, W95White))
	}
	if hasDown {
		s.Components = append(s.Components, &scrollArrow{cy: row + visibleRows, cols: cols, up: false})
	}
	if s.OnBack != nil {
		s.Components = append(s.Components, NewButton(2, d.Rows-3, s.BackLabel, s.OnBack, len(s.BackLabel)+4))
	}
	// store for touch
	s.scrollHasUp = hasUp
	s.scrollHasDown = hasDown
	s.scrollVisible = visibleRows
	s.scrollRowStart = row - boolToInt(hasUp)
	if s.ShowTitle {
		s.scrollRowStart = 2
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}



func (s *ListScreen95) OnTouch(px, py int) bool {
	d := s.App.Display
	total := len(s.Items)
	row := py / cell
	// back button
	if row >= d.Rows-3 && s.OnBack != nil {
		s.OnBack()
		return false
	}
	if s.ShowTitle && row < 2 {
		for _, c := range s.Components {
			if c.Tap(px, py) {
				return false
			}
		}
		return false
	}
	startRow := 2
	if !s.ShowTitle {
		startRow = 0
	}
	// up arrow
	if s.scroll > 0 && row == startRow {
		s.scroll--
		s.build()
		return true
	}
	// down arrow
	if s.scroll+s.scrollVisible < total {
		last := d.Rows - 4
		if row == last {
			s.scroll++
			s.build()
			return true
		}
	}
	// item select
	if row >= startRow && row < d.Rows-3 {
		// adjust for up arrow offset
		effectiveStart := startRow
		if s.scrollHasUp {
			effectiveStart++
		}
		if row < effectiveStart {
			return false
		}
		idx := s.scroll + (row - effectiveStart)
		if idx >= 0 && idx < total && s.OnSelect != nil {
			s.OnSelect(idx)
			return false
		}
	}
	return false
}



// MessageScreen95
type MessageScreen95 struct {
	BaseScreen
	Title    string
	Messages []map[string]interface{}
	OnBack   func()
	scroll   int
	hasUp    bool
	hasDown  bool
	visible  int
}

func NewMessageScreen(title string, msgs []map[string]interface{}, onBack func()) *MessageScreen95 {
	return &MessageScreen95{Title: title, Messages: msgs, OnBack: onBack}
}
func (s *MessageScreen95) OnShow(args map[string]interface{}) {
	if v, ok := args["messages"]; ok {
		if vv, ok := v.([]map[string]interface{}); ok {
			s.Messages = vv
		}
	}
	if v, ok := args["title"]; ok {
		if vv, ok := v.(string); ok {
			s.Title = vv
		}
	}
	if v, ok := args["on_back"]; ok {
		if vv, ok := v.(func()); ok {
			s.OnBack = vv
		}
	}
	s.scroll = 0
	s.build()
}
func (s *MessageScreen95) build() {
	if s.App == nil {
		return
	}
	d := s.App.Display
	cols := d.Cols
	t := strings.TrimPrefix(s.Title, "#")
	total := len(s.Messages)
	// calc visible messages: each msg 2 rows, arrows 1 row each
	baseVisible := (d.Rows - 5) / 2
	if baseVisible < 0 {
		baseVisible = 0
	}
	hasUp := s.scroll > 0
	hasDown := false
	vis := baseVisible
	if hasUp {
		vis--
	}
	if s.scroll+vis < total {
		hasDown = true
		vis--
	}
	if vis < 0 {
		vis = 0
	}
	s.hasUp = hasUp
	s.hasDown = hasDown
	s.visible = vis

	s.Components = nil
	s.Components = append(s.Components, NewTitleBar("#"+t, s.OnBack))
	row := 2
	if hasUp {
		s.Components = append(s.Components, &scrollArrow{cy: row, cols: cols, up: true})
		row++
	}
	visEnd := s.scroll + vis
	if visEnd > total {
		visEnd = total
	}
	for i := s.scroll; i < visEnd; i++ {
		msg := s.Messages[i]
		author := "?"
		if a, ok := msg["author"].(map[string]interface{}); ok {
			if u, ok := a["username"].(string); ok {
				author = u
			}
		}
		content, _ := msg["content"].(string)
		s.Components = append(s.Components, NewLabel(2, row, author, 0, W95Blue, W95White))
		s.Components = append(s.Components, NewLabel(2, row+1, "  "+trunc(content, cols-5), 0, W95Black, W95White))
		if i < visEnd-1 {
			s.Components = append(s.Components, &divider{y: cy(row+2) - 12, w: cols * cell})
		}
		row += 2
	}
	if hasDown {
		s.Components = append(s.Components, &scrollArrow{cy: row, cols: cols, up: false})
	}
	s.Components = append(s.Components, NewButton(2, d.Rows-3, "  OK  ", s.OnBack, 6))
}
func (s *MessageScreen95) OnTouch(px, py int) bool {
	d := s.App.Display
	row := py / cell
	if row >= d.Rows-3 && s.OnBack != nil {
		s.OnBack()
		return false
	}
	if row < 2 {
		for _, c := range s.Components {
			if c.Tap(px, py) {
				return false
			}
		}
		return false
	}
	if s.hasUp && row == 2 {
		s.scroll--
		s.build()
		return true
	}
	if s.hasDown {
		last := d.Rows - 4
		if row == last {
			s.scroll++
			s.build()
			return true
		}
	}
	return false
}

// Dialog95
type Dialog95 struct {
	BaseScreen
	Title   string
	Message string
	OnOK    func()
}

func NewDialog(title, message string, onOK func()) *Dialog95 {
	return &Dialog95{Title: title, Message: message, OnOK: onOK}
}
func (d2 *Dialog95) OnShow(args map[string]interface{}) {
	if v, ok := args["title"]; ok {
		if vv, ok := v.(string); ok {
			d2.Title = vv
		}
	}
	if v, ok := args["message"]; ok {
		if vv, ok := v.(string); ok {
			d2.Message = vv
		}
	}
	if v, ok := args["on_ok"]; ok {
		if vv, ok := v.(func()); ok {
			d2.OnOK = vv
		}
	}
	d2.build()
}
func (d2 *Dialog95) build() {
	if d2.App == nil {
		return
	}
	d := d2.App.Display
	cols := d.Cols
	d2.Components = nil
	d2.Components = append(d2.Components, NewTitleBar(d2.Title, d2.OnOK))
	lines := strings.Split(d2.Message, "\n")
	y := 4
	for _, line := range lines {
		cx_ := (cols - len(line)) / 2
		if cx_ < 2 {
			cx_ = 2
		}
		d2.Components = append(d2.Components, NewLabel(cx_, y, line, 0, W95Black, W95White))
		y += 2
	}
	btnY := y + 1
	d2.Components = append(d2.Components, NewButton((cols-6)/2, btnY, "  OK  ", d2.OnOK, 6))
}
