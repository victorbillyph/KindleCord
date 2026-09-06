package ui

import (
	"strings"

	"kindlecord/internal/display"
)

const (
	BG        = 0xEE
	CARD      = 0xFF
	PRIMARY   = 0x22
	ACCENT    = 0x44
	TEXT      = 0x11
	TEXTMUTED = 0x77
	BORDER    = 0xCC
	BTNBG     = 0x33
	BTNFG     = 0xFF
	GOVERLAY  = 0xF2
)

const cell = display.CellSize

const (
	FontTitle   = 18
	FontButton  = 14
	FontLabel   = 12
	FontSmall   = 10
	FontClose   = 14
	FontMsg     = 12
	FontAuthor  = 12
)

func px(c int) int { return c * cell }

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "~"
	}
	return s[:max-1] + "~"
}

// Component interface
type Component interface {
	Render(d *display.Display)
	Tap(px, py int) bool
	Contains(px, py int) bool
}

// Button
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
	return &Button95{CX: cx_, CY: cy_, Text: text, Callback: cb, CW: cw, W: cw * cell, H: 40}
}

func (b *Button95) Render(d *display.Display) {
	x := px(b.CX)
	y := px(b.CY)
	r := 8
	if b.H < 20 {
		r = 4
	}
	d.FillRoundRect(x, y, b.W, b.H, r, BTNBG)
	if b.Pressed {
		d.FillRoundRect(x+1, y+1, b.W-2, b.H-2, r-1, ACCENT)
	}
	tx := x + (b.W-len(b.Text)*10)/2
	ty := y + (b.H-14)/2
	_ = tx
	_ = ty
	d.DrawTextSized(b.CX+(b.CW-len(b.Text))/2, b.CY+1, FontButton, b.Text, BTNFG, BTNBG)
}

func (b *Button95) Contains(px_, py_ int) bool {
	x := px(b.CX)
	y := px(b.CY)
	return px_ >= x && px_ < x+b.W && py_ >= y && py_ < y+b.H
}

func (b *Button95) Tap(px_, py_ int) bool {
	if b.Contains(px_, py_) && b.Callback != nil {
		b.Callback()
		return true
	}
	return false
}

// Label
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
	d.DrawTextSized(l.CX, l.CY, FontLabel, txt, l.FG, l.BG)
}

func (l *Label95) Contains(px_, py_ int) bool { return false }
func (l *Label95) Tap(px_, py_ int) bool      { return false }

// TitleBar
type TitleBar95 struct {
	Title   string
	OnClose func()
	rect    [4]int
	BarH    int
}

func NewTitleBar(title string, onClose func()) *TitleBar95 {
	return &TitleBar95{Title: title, OnClose: onClose, BarH: 44}
}

func (t *TitleBar95) Render(d *display.Display) {
	w := d.Width
	d.FillRect(0, 0, w, t.BarH, PRIMARY)
	d.FillRect(0, t.BarH-1, w, 1, BORDER)
	d.DrawTextSized(2, 1, FontTitle, trunc(t.Title, d.Cols-6), CARD, PRIMARY)
	if t.OnClose != nil {
		closeX := d.Cols - 4
		t.rect = [4]int{px(closeX), 6, 3*cell, t.BarH - 12}
		d.FillRoundRect(t.rect[0], t.rect[1], t.rect[2], t.rect[3], 6, ACCENT)
		d.DrawTextSized(closeX+1, 1, FontClose, "X", CARD, ACCENT)
	}
}

func (t *TitleBar95) Contains(px_, py_ int) bool {
	x, y, w, h := t.rect[0], t.rect[1], t.rect[2], t.rect[3]
	return px_ >= x && px_ < x+w && py_ >= y && py_ < y+h
}

func (t *TitleBar95) Tap(px_, py_ int) bool {
	if t.OnClose != nil && t.Contains(px_, py_) {
		t.OnClose()
		return true
	}
	return false
}

// ModernHeader
type ModernHeader struct {
	Title   string
	OnClose func()
	rect    [4]int
	BarH    int
}

func NewModernHeader(title string, onClose func()) *ModernHeader {
	return &ModernHeader{Title: title, OnClose: onClose, BarH: 44}
}

func (m *ModernHeader) Render(d *display.Display) {
	w := d.Width
	d.FillRect(0, 0, w, m.BarH, PRIMARY)
	d.DrawTextSized(2, 1, FontTitle, trunc(m.Title, d.Cols-6), CARD, PRIMARY)
	if m.OnClose != nil {
		closeX := d.Cols - 4
		m.rect = [4]int{px(closeX), 6, 3*cell, m.BarH - 12}
		d.FillRoundRect(m.rect[0], m.rect[1], m.rect[2], m.rect[3], 6, ACCENT)
		d.DrawTextSized(closeX+1, 1, FontClose, "X", CARD, ACCENT)
	}
}

func (m *ModernHeader) Contains(px_, py_ int) bool {
	x, y, w, h := m.rect[0], m.rect[1], m.rect[2], m.rect[3]
	return px_ >= x && px_ < x+w && py_ >= y && py_ < y+h
}

func (m *ModernHeader) Tap(px_, py_ int) bool {
	if m.OnClose != nil && m.Contains(px_, py_) {
		m.OnClose()
		return true
	}
	return false
}

// Card
type Card struct {
	x, y, w, h, radius int
}

func (c *Card) Render(d *display.Display) {
	d.FillRoundRect(c.x, c.y, c.w, c.h, c.radius, CARD)
	d.Rect(c.x, c.y, c.w, c.h, BORDER, 1)
}

func (c *Card) Contains(px_, py_ int) bool { return false }
func (c *Card) Tap(px_, py_ int) bool      { return false }

// Box
type Box struct {
	x, y, w, h, radius int
	bg, border         uint8
}

func (b *Box) Render(d *display.Display) {
	d.FillRoundRect(b.x, b.y, b.w, b.h, b.radius, b.border)
	d.FillRoundRect(b.x+2, b.y+2, b.w-4, b.h-4, b.radius-2, b.bg)
}

func (b *Box) Contains(px_, py_ int) bool { return false }
func (b *Box) Tap(px_, py_ int) bool      { return false }

// rowBg
type rowBg struct{ cy, cols int }

func (r *rowBg) Render(d *display.Display) {
	d.FillRoundRect(8, px(r.cy)+2, d.Width-16, cell-4, 6, GOVERLAY)
}

func (r *rowBg) Contains(px_, py_ int) bool { return false }
func (r *rowBg) Tap(px_, py_ int) bool      { return false }

// divider
type divider struct{ y, w int }

func (d2 *divider) Render(d *display.Display) {
	d.FillRect(12, d2.y, d.Width-24, 1, BORDER)
}

func (d2 *divider) Contains(px_, py_ int) bool { return false }
func (d2 *divider) Tap(px_, py_ int) bool      { return false }

// scrollArrow
type scrollArrow struct {
	cy, cols int
	up       bool
}

func (s *scrollArrow) Render(d *display.Display) {
	y := px(s.cy)
	w := d.Width
	d.FillRoundRect(8, y+2, w-16, cell-4, 6, ACCENT)
	label := "  \\/  "
	if s.up {
		label = "  /\\  "
	}
	d.DrawTextSized(2, s.cy, FontSmall, label, CARD, ACCENT)
}

func (s *scrollArrow) Contains(px_, py_ int) bool { return false }
func (s *scrollArrow) Tap(px_, py_ int) bool      { return false }

// App
type App95 struct {
	Display *display.Display
	Screens map[string]Screen
	Current string
	Running bool
}

type Screen interface {
	SetApp(a *App95)
	Render(d *display.Display)
	OnTouch(px, py int) bool
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

func (a *App95) Touch(px_, py_ int) {
	if a.Current == "" {
		return
	}
	scr := a.Screens[a.Current]
	if scr != nil {
		if scr.OnTouch(px_, py_) {
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
	d.Clear(BG)
	for _, c := range b.Components {
		c.Render(d)
	}
	d.Refresh()
}

func (b *BaseScreen) OnShow(args map[string]interface{}) { b.Components = nil }

func (b *BaseScreen) OnTouch(px_, py_ int) bool {
	for _, c := range b.Components {
		if c.Tap(px_, py_) {
			return false
		}
	}
	return false
}

// LoginScreen
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

	s.Components = append(s.Components, NewTitleBar("KindleCord", s.OnQuit))

	y := 3

	s.Components = append(s.Components, NewLabel(2, y, "Open on your phone:", 0, TEXTMUTED, BG))
	y += 2

	url := s.URL
	if url == "" {
		url = "http://0.0.0.0:8080"
	}
	cx_ := (cols - len(url)) / 2
	if cx_ < 2 {
		cx_ = 2
	}
	s.Components = append(s.Components, NewLabel(cx_, y, url, cols-4, ACCENT, BG))
	y += 2

	if s.SSHInfo != "" {
		sshCx := (cols - len(s.SSHInfo)) / 2
		if sshCx < 2 {
			sshCx = 2
		}
		s.Components = append(s.Components, NewLabel(sshCx, y, s.SSHInfo, cols-4, TEXTMUTED, BG))
		y += 2
	} else {
		y += 1
	}

	y += 1

	s.Components = append(s.Components, NewLabel(2, y, "Paste your Discord token", 0, TEXT, BG))
	y += 2
	s.Components = append(s.Components, NewLabel(2, y, "to log in.", 0, TEXT, BG))
	y += 2
	s.Components = append(s.Components, NewLabel(2, y, "Waiting for token...", 0, TEXTMUTED, BG))

	btnY := d.Rows - 3
	s.Components = append(s.Components, NewButton((cols-8)/2, btnY, "  Exit  ", s.OnQuit, 8))
}

func (s *LoginScreen95) Render(d *display.Display) { s.BaseScreen.Render(d) }

func (s *LoginScreen95) OnTouch(px_, py_ int) bool {
	for _, c := range s.Components {
		c.Tap(px_, py_)
	}
	return false
}

// ListScreen
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
		s.Components = append(s.Components, &Card{x: 4, y: px(row) - 2, w: d.Width - 8, h: (d.Rows - row - 3) * cell, radius: 8})
	}

	total := len(s.Items)
	visibleRows := d.Rows - row - 4
	if visibleRows < 0 {
		visibleRows = 0
	}
	hasUp := s.scroll > 0
	hasDown := false

	if hasUp {
		visibleRows--
	}
	if s.scroll+visibleRows < total {
		hasDown = true
		visibleRows--
		if visibleRows < 0 {
			visibleRows = 0
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
		s.Components = append(s.Components, NewLabel(2, row+vi, txt, cols-4, TEXT, CARD))
	}

	if hasDown {
		s.Components = append(s.Components, &scrollArrow{cy: row + visibleRows, cols: cols, up: false})
	}

	if s.OnBack != nil {
		s.Components = append(s.Components, NewButton(2, d.Rows-3, s.BackLabel, s.OnBack, len(s.BackLabel)+4))
	}

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

func (s *ListScreen95) OnTouch(px_, py_ int) bool {
	d := s.App.Display
	total := len(s.Items)
	row := py_ / cell

	if row >= d.Rows-3 && s.OnBack != nil {
		s.OnBack()
		return false
	}

	if s.ShowTitle && row < 2 {
		for _, c := range s.Components {
			if c.Tap(px_, py_) {
				return false
			}
		}
		return false
	}

	startRow := 2
	if !s.ShowTitle {
		startRow = 0
	}

	if s.scroll > 0 && row == startRow {
		s.scroll--
		s.build()
		return true
	}

	if s.scroll+s.scrollVisible < total {
		last := d.Rows - 4
		if row == last {
			s.scroll++
			s.build()
			return true
		}
	}

	if row >= startRow && row < d.Rows-3 {
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

// MessageScreen
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
		s.Components = append(s.Components, NewLabel(2, row, author, 0, PRIMARY, CARD))
		s.Components = append(s.Components, NewLabel(2, row+1, "  "+trunc(content, cols-5), 0, TEXT, CARD))
		if i < visEnd-1 {
			s.Components = append(s.Components, &divider{y: px(row+2) - 10, w: cols * cell})
		}
		row += 2
	}
	if hasDown {
		s.Components = append(s.Components, &scrollArrow{cy: row, cols: cols, up: false})
	}
	s.Components = append(s.Components, NewButton(2, d.Rows-3, "  OK  ", s.OnBack, 6))
}

func (s *MessageScreen95) OnTouch(px_, py_ int) bool {
	d := s.App.Display
	row := py_ / cell
	if row >= d.Rows-3 && s.OnBack != nil {
		s.OnBack()
		return false
	}
	if row < 2 {
		for _, c := range s.Components {
			if c.Tap(px_, py_) {
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

// Dialog
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
		d2.Components = append(d2.Components, NewLabel(cx_, y, line, 0, TEXT, BG))
		y += 2
	}
	btnY := y + 1
	d2.Components = append(d2.Components, NewButton((cols-6)/2, btnY, "  OK  ", d2.OnOK, 6))
}
