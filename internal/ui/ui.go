package ui

import (
	"strings"
	"sync"

	"kindlecord/internal/display"
)

const (
	SIDEBAR_W = 80
	HEADER_H  = 44
	CONTENT_X = SIDEBAR_W + 2

	BG_SIDEBAR  = 0x33
	BG_CONTENT  = 0xF2
	BG_HEADER   = 0x22
	BG_SELECTED = 0x55
	BG_ICON     = 0x44
	BG_DM       = 0x58
	BG_CARD     = 0xFF
	GCARD_ALT   = 0xF6
	BG_BTN      = 0x33

	FG_WHITE   = 0xFF
	FG_BLACK   = 0x11
	FG_MUTED   = 0x88
	FG_DIVIDER = 0xBB
	FG_BORDER  = 0xCC
)

const (
	FT_ICON  = 32
	FT_TITLE = 28
	FT_BTN   = 24
	FT_LABEL = 22
	FT_MSG   = 22
	FT_SMALL = 18
)

func trunc(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return "~"
	}
	return s[:max-1] + "~"
}

type Component interface {
	Render(d *display.Display)
	Tap(x, y int) bool
	Contains(x, y int) bool
}

// ── Sidebar ──────────────────────────────────────────────────────────

type Sidebar struct {
	Servers         []string
	SelectedDM      bool
	ServerIdx       int
	OnDMClick       func()
	OnServerClick   func(idx int)
	OnUpdateClick   func()
	OnSettingsClick func()
	contentH        int
	scroll          int

	iconSize  int
	iconGap   int
	iconStart int
}

const (
	sbIconSize  = 44
	sbIconGap   = 12
	sbSettingsY = 16
	sbDMY       = 64
	sbDividerY  = 120
)

func (s *Sidebar) iconY(i int) int {
	return s.iconStart + i*(sbIconSize+sbIconGap)
}

func (s *Sidebar) serverTop() int {
	return sbDividerY + 12
}

func (s *Sidebar) fitIcons(cmds int) int {
	return (sbIconSize + sbIconGap) * cmds
}

func (s *Sidebar) Render(d *display.Display) {
	h := d.Height
	if s.contentH > 0 {
		h = s.contentH
	}
	d.FillRect(0, 0, SIDEBAR_W, h, BG_SIDEBAR)

	// settings icon (top)
	d.FillRoundRect((SIDEBAR_W-sbIconSize)/2, sbSettingsY, sbIconSize, sbIconSize, sbIconSize/2, BG_SIDEBAR)
	d.FillRoundRect((SIDEBAR_W-sbIconSize+4)/2, sbSettingsY+2, sbIconSize-4, sbIconSize-4, (sbIconSize-4)/2, BG_ICON)
	s.drawGear(d, SIDEBAR_W/2, sbSettingsY+sbIconSize/2, 24)

	// DM icon
	var dmBg uint8 = BG_SIDEBAR
	if s.SelectedDM {
		dmBg = BG_SELECTED
	}
	d.FillRoundRect((SIDEBAR_W-sbIconSize)/2, sbDMY, sbIconSize, sbIconSize, sbIconSize/2, dmBg)
	d.FillRoundRect((SIDEBAR_W-sbIconSize+4)/2, sbDMY+2, sbIconSize-4, sbIconSize-4, (sbIconSize-4)/2, BG_DM)
	d.DrawTextPixel(SIDEBAR_W/2-7, sbDMY+9, FT_ICON, "D", FG_WHITE, BG_DM)

	d.FillRect(SIDEBAR_W/2-14, sbDividerY, 28, 2, 0x55)

	// compute how many icons fit
	s.iconStart = s.serverTop()
	maxVisible := (h - s.iconStart) / (sbIconSize + sbIconGap)
	if maxVisible < 1 {
		maxVisible = 1
	}
	maxScroll := len(s.Servers) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scroll > maxScroll {
		s.scroll = maxScroll
	}

	y := s.iconStart
	for i := s.scroll; i < len(s.Servers); i++ {
		name := s.Servers[i]
		if y+sbIconSize > h {
			break
		}
		letter := "?"
		if len(name) > 0 {
			letter = strings.ToUpper(name[:1])
		}
		var ibg uint8 = BG_ICON
		if !s.SelectedDM && s.ServerIdx == i {
			d.FillRoundRect((SIDEBAR_W-sbIconSize)/2, y, sbIconSize, sbIconSize, sbIconSize/2, BG_SELECTED)
		}
		d.FillRoundRect((SIDEBAR_W-sbIconSize+4)/2, y+2, sbIconSize-4, sbIconSize-4, (sbIconSize-4)/2, ibg)
		d.DrawTextPixel(SIDEBAR_W/2-7, y+9, FT_ICON, letter, FG_WHITE, ibg)
		// selection indicator bar
		if !s.SelectedDM && s.ServerIdx == i {
			d.FillRoundRect(4, y+6, 4, sbIconSize-12, 2, FG_WHITE)
		}
		y += sbIconSize + sbIconGap
	}

	// scroll arrows at top/bottom of sidebar
	if s.scroll > 0 {
		d.FillRoundRect(6, s.serverTop()-6, SIDEBAR_W-12, 10, 3, BG_SIDEBAR)
		d.DrawTextPixel(SIDEBAR_W/2-4, s.serverTop()-6, FT_SMALL, "^", FG_WHITE, BG_SIDEBAR)
	}
	if s.scroll+maxVisible < len(s.Servers) {
		ay := h - 16
		d.DrawTextPixel(SIDEBAR_W/2-4, ay, FT_SMALL, "v", FG_WHITE, BG_SIDEBAR)
	}

	// update icon at very bottom
	uy := h - sbIconSize - 8
	if uy > s.serverTop() {
		d.FillRoundRect((SIDEBAR_W-sbIconSize)/2, uy, sbIconSize, sbIconSize, sbIconSize/2, BG_SIDEBAR)
		d.DrawTextPixel(SIDEBAR_W/2-7, uy+9, FT_ICON, "U", FG_WHITE, BG_SIDEBAR)
	}
}

func (s *Sidebar) drawGear(d *display.Display, cx, cy, size int) {
	half := size / 2
	tooth := size / 5
	halfTooth := tooth / 2
	tall := size / 3
	// four teeth
	d.FillRect(cx-halfTooth, cy-half, tooth, tall, FG_WHITE)
	d.FillRect(cx-halfTooth, cy+half-tall, tooth, tall, FG_WHITE)
	d.FillRect(cx-half, cy-halfTooth, tall, tooth, FG_WHITE)
	d.FillRect(cx+half-tall, cy-halfTooth, tall, tooth, FG_WHITE)
	// wheel + hole
	d.FillRoundRect(cx-size/4, cy-size/4, size/2, size/2, size/4, FG_WHITE)
	d.FillRoundRect(cx-size/10, cy-size/10, size/5, size/5, 2, BG_ICON)
}

func (s *Sidebar) HandleTouch(x, y int) {
	if x >= SIDEBAR_W {
		return
	}
	if y >= sbSettingsY && y <= sbSettingsY+sbIconSize {
		if s.OnSettingsClick != nil {
			s.OnSettingsClick()
		}
		return
	}
	if y >= sbDMY && y <= sbDMY+sbIconSize {
		if s.OnDMClick != nil {
			s.OnDMClick()
		}
		return
	}
	// server scroll controls
	maxScroll := len(s.Servers)
	if maxScroll > 0 {
		maxVisible := (s.contentH - s.serverTop()) / (sbIconSize + sbIconGap)
		if maxVisible < 1 {
			maxVisible = 1
		}
		maxScroll = maxScroll - maxVisible
		if maxScroll < 0 {
			maxScroll = 0
		}
	}
	if y >= s.serverTop()-10 && y <= s.serverTop() && s.scroll > 0 {
		s.scroll--
		return
	}
	if y >= s.contentH-16 && y <= s.contentH && s.scroll < maxScroll {
		s.scroll++
		return
	}
	// update icon at bottom
	if s.OnUpdateClick != nil {
		uy := s.contentH - sbIconSize - 8
		if y >= uy && y <= uy+sbIconSize {
			s.OnUpdateClick()
			return
		}
	}
	// server icon
	for i := s.scroll; i < len(s.Servers); i++ {
		iy := s.iconY(i)
		if iy == 0 {
			s.iconStart = s.serverTop()
		}
		if y >= iy && y <= iy+sbIconSize {
			if s.OnServerClick != nil {
				s.OnServerClick(i)
			}
			return
		}
	}
}

// ── Button ───────────────────────────────────────────────────────────

type Button struct {
	X, Y, W, H int
	Text       string
	Callback   func()
}

func NewButton(x, y int, text string, cb func()) *Button {
	w := len(text)*10 + 24
	return &Button{X: x, Y: y, W: w, H: 36, Text: text, Callback: cb}
}

func (b *Button) Render(d *display.Display) {
	d.FillRoundRect(b.X, b.Y, b.W, b.H, 8, BG_BTN)
	tx := b.X + (b.W-len(b.Text)*10)/2
	ty := b.Y + (b.H-FT_BTN)/2
	d.DrawTextPixel(tx, ty, FT_BTN, b.Text, FG_WHITE, BG_BTN)
}

func (b *Button) Contains(x, y int) bool {
	return x >= b.X && x < b.X+b.W && y >= b.Y && y < b.Y+b.H
}

func (b *Button) Tap(x, y int) bool {
	if b.Contains(x, y) && b.Callback != nil {
		b.Callback()
		return true
	}
	return false
}

// ── Label ────────────────────────────────────────────────────────────

type Label struct {
	X, Y int
	Text string
	Size int
	FG   uint8
}

func NewLabel(x, y int, text string, size int, fg uint8) *Label {
	if size == 0 {
		size = FT_LABEL
	}
	return &Label{X: x, Y: y, Text: text, Size: size, FG: fg}
}

func (l *Label) Render(d *display.Display) {
	d.DrawTextPixel(l.X, l.Y, l.Size, l.Text, l.FG, BG_CONTENT)
}

func (l *Label) Contains(x, y int) bool { return false }

// ── Box (bordered highlighted box: URL / code snippet) ───────────────

type Box struct {
	X, Y, W, H int
	Text       string
	Size       int
	Fill       uint8
	FG         uint8
}

func NewBox(x, y, w, h int, text string, size int, fill, fg uint8) *Box {
	return &Box{X: x, Y: y, W: w, H: h, Text: text, Size: size, Fill: fill, FG: fg}
}

func (b *Box) Render(d *display.Display) {
	d.FillRect(b.X, b.Y, b.W, b.H, FG_BORDER)
	d.FillRoundRect(b.X+1, b.Y+1, b.W-2, b.H-2, 8, b.Fill)
	d.DrawTextPixel(b.X+8, b.Y+6, b.Size, b.Text, b.FG, b.Fill)
}

func (b *Box) Contains(x, y int) bool { return false }
func (b *Box) Tap(x, y int) bool      { return false }
func (l *Label) Tap(x, y int) bool    { return false }

// ── ListItem ─────────────────────────────────────────────────────────

type ListItem struct {
	X, Y, W, H int
	Text       string
	Size       int
	FG         uint8
	BG         uint8
	Callback   func()
}

func NewListItem(x, y, w, h int, text string, fg, bg uint8, cb func()) *ListItem {
	return &ListItem{X: x, Y: y, W: w, H: h, Text: text, Size: FT_LABEL, FG: fg, BG: bg, Callback: cb}
}

func (li *ListItem) Render(d *display.Display) {
	// card with 1px border (built from 2 fills instead of a ring + text)
	bg := li.BG
	if bg == BG_CONTENT {
		bg = BG_CARD
	}
	d.FillRect(li.X, li.Y, li.W, li.H, FG_BORDER)
	d.FillRoundRect(li.X+1, li.Y+1, li.W-2, li.H-2, 5, bg)
	d.DrawTextPixel(li.X+14, li.Y+(li.H-li.Size)/2, li.Size, li.Text, li.FG, bg)
}

func (li *ListItem) Contains(x, y int) bool {
	return x >= li.X && x < li.X+li.W && y >= li.Y && y < li.Y+li.H
}

func (li *ListItem) Tap(x, y int) bool {
	if li.Contains(x, y) && li.Callback != nil {
		li.Callback()
		return true
	}
	return false
}

// ── ScrollBar ────────────────────────────────────────────────────────

type ScrollBar struct {
	X, Y, H, Total, Visible, Offset int
}

func (sb *ScrollBar) Render(d *display.Display) {
	if sb.Total <= sb.Visible {
		return
	}
	d.FillRect(sb.X, sb.Y, 3, sb.H, FG_DIVIDER)
	barH := sb.H * sb.Visible / sb.Total
	if barH < 12 {
		barH = 12
	}
	barY := sb.Y + (sb.H-barH)*sb.Offset/(sb.Total-sb.Visible)
	d.FillRoundRect(sb.X, barY, 3, barH, 1, FG_MUTED)
}

// ── App ──────────────────────────────────────────────────────────────

type App struct {
	mu      sync.Mutex
	Display *display.Display
	Screens map[string]Screen
	Current string
	Running bool
}

type Screen interface {
	SetApp(a *App)
	Render(d *display.Display)
	OnTouch(x, y int) bool
	OnShow(args map[string]interface{})
}

func NewApp(d *display.Display) *App {
	return &App{Display: d, Screens: make(map[string]Screen), Running: true}
}

func (a *App) Add(name string, s Screen) {
	s.SetApp(a)
	a.Screens[name] = s
}

func (a *App) Show(name string, args map[string]interface{}) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Current = name
	scr := a.Screens[name]
	if scr != nil {
		scr.OnShow(args)
		scr.Render(a.Display)
	}
}

func (a *App) Touch(x, y int) {
	a.mu.Lock()
	if a.Current == "" {
		a.mu.Unlock()
		return
	}
	scr := a.Screens[a.Current]
	a.mu.Unlock()

	if scr != nil && scr.OnTouch(x, y) {
		a.mu.Lock()
		scr.Render(a.Display)
		a.mu.Unlock()
	}
}

func (a *App) Stop() { a.Running = false }

// ── LoginScreen ──────────────────────────────────────────────────────
type LoginScreen struct {
	App     *App
	URL     string
	SSHInfo string
	OnQuit  func()
	OnFinish func()
	page    int
	items   []Component
}

func NewLoginScreen(url string, onQuit func()) *LoginScreen {
	return &LoginScreen{URL: url, OnQuit: onQuit, page: 0}
}

func (s *LoginScreen) SetApp(a *App) { s.App = a }

func (s *LoginScreen) SetPage(page int) {
	s.page = page
	s.build()
}

func (s *LoginScreen) OnShow(args map[string]interface{}) {
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
	if v, ok := args["on_finish"]; ok {
		if f, ok := v.(func()); ok {
			s.OnFinish = f
		}
	}
	if v, ok := args["page"]; ok {
		if p, ok := v.(int); ok {
			s.page = p
		}
	}
	s.build()
}

func (s *LoginScreen) build() {
	d := s.App.Display
	s.items = nil

	urlStr := s.URL
	if urlStr == "" {
		urlStr = "http://0.0.0.0:8080"
	}

	switch s.page {
	case 0:
		// Step 1: Welcome
		s.buildWelcome(d, urlStr)
	case 1:
		// Step 2: Configure via browser
		s.buildConfigure(d, urlStr)
	case 2:
		// Step 3: Success - finish setup
		s.buildSuccess(d)
	}
}

func (s *LoginScreen) buildWelcome(d *display.Display, urlStr string) {
	y := 70
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Bem-vindo ao KindleCord", FT_TITLE, FG_BLACK))
	y += 38

	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Um cliente Discord para o seu", FT_LABEL, FG_BLACK))
	y += 26
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Kindle e-ink.", FT_LABEL, FG_BLACK))
	y += 36

	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Navegue em servidores, canais", FT_SMALL, FG_MUTED))
	y += 22
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "e leia mensagens na tela", FT_SMALL, FG_MUTED))
	y += 22
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "eletrônica, sem distrações.", FT_SMALL, FG_MUTED))
	y += 36

	btnX := CONTENT_X + (d.Width-CONTENT_X-100)/2
	s.items = append(s.items, NewButton(btnX+20, d.Height-70, "Próximo >", func() { s.page = 1; s.build() }))
}

func (s *LoginScreen) buildConfigure(d *display.Display, urlStr string) {
	y := 70
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Acesse no Browser para Configurar", FT_TITLE, FG_BLACK))
	y += 38

	s.items = append(s.items, NewLabel(CONTENT_X+12, y, urlStr, FT_TITLE, FG_BLACK))
	y += 40

	if s.SSHInfo != "" {
		s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Ou SSH:", FT_SMALL, FG_MUTED))
		y += 24
		s.items = append(s.items, NewLabel(CONTENT_X+12, y, s.SSHInfo, FT_SMALL, FG_BLACK))
		y += 36
	}

	y += 10
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Abra a URL acima no seu", FT_LABEL, FG_BLACK))
	y += 26
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "computador/celular (mesmo Wi-Fi)", FT_LABEL, FG_BLACK))
	y += 26
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Cole o token e clique Log in", FT_LABEL, FG_BLACK))
	y += 36

	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Aguardando token...", FT_SMALL, FG_MUTED))

	btnX := CONTENT_X + (d.Width-CONTENT_X-100)/2
	s.items = append(s.items, NewButton(btnX-100, d.Height-70, "< Voltar", func() { s.page = 0; s.build() }))
	s.items = append(s.items, NewButton(btnX+20, d.Height-70, "Próximo >", func() { s.page = 2; s.build() }))
}

func (s *LoginScreen) buildSuccess(d *display.Display) {
	y := 90
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Tudo certo!", FT_TITLE, FG_BLACK))
	y += 40

	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Token recebido com sucesso.", FT_LABEL, FG_BLACK))
	y += 26
	s.items = append(s.items, NewLabel(CONTENT_X+12, y, "Clique abaixo para entrar no app.", FT_LABEL, FG_MUTED))
	y += 40

	btnX := CONTENT_X + (d.Width-CONTENT_X-100)/2
	s.items = append(s.items, NewButton(btnX, d.Height-70, "Concluir setup", func() {
		if s.OnFinish != nil {
			s.OnFinish()
		}
	}))
}

func (s *LoginScreen) Render(d *display.Display) {
	d.Clear(BG_CONTENT)
	d.FillRect(0, 0, d.Width, HEADER_H, BG_HEADER)
	d.DrawTextPixel(12, 12, FT_TITLE, "KindleCord", FG_WHITE, BG_HEADER)
	for _, c := range s.items {
		c.Render(d)
	}
	d.Refresh()
}

func (s *LoginScreen) OnTouch(x, y int) bool {
	if y < HEADER_H {
		return false
	}
	for _, c := range s.items {
		if c.Tap(x, y) {
			return true
		}
	}
	return false
}

// ── HomeScreen (sidebar + DM/channel list) ───────────────────────────

type HomeScreen struct {
	App        *App
	Sidebar    Sidebar
	Title      string
	Items      []*ListItem
	OnBack     func()
	scroll     int
	TotalItems int
}

func NewHomeScreen() *HomeScreen {
	return &HomeScreen{}
}

func (s *HomeScreen) SetApp(a *App) { s.App = a }

func (s *HomeScreen) OnShow(args map[string]interface{}) {
	if v, ok := args["servers"]; ok {
		if vv, ok := v.([]string); ok {
			s.Sidebar.Servers = vv
		}
	}
	if v, ok := args["selected_dm"]; ok {
		if vv, ok := v.(bool); ok {
			s.Sidebar.SelectedDM = vv
		}
	}
	if v, ok := args["server_idx"]; ok {
		if vv, ok := v.(int); ok {
			s.Sidebar.ServerIdx = vv
		}
	}
	if v, ok := args["on_dm_click"]; ok {
		if f, ok := v.(func()); ok {
			s.Sidebar.OnDMClick = f
		}
	}
	if v, ok := args["on_server_click"]; ok {
		if f, ok := v.(func(int)); ok {
			s.Sidebar.OnServerClick = f
		}
	}
	if v, ok := args["on_update_click"]; ok {
		if f, ok := v.(func()); ok {
			s.Sidebar.OnUpdateClick = f
		}
	}
	if v, ok := args["on_settings_click"]; ok {
		if f, ok := v.(func()); ok {
			s.Sidebar.OnSettingsClick = f
		}
	}
	if v, ok := args["title"]; ok {
		if vv, ok := v.(string); ok {
			s.Title = vv
		}
	}
	if v, ok := args["items"]; ok {
		if vv, ok := v.([]string); ok {
			s.Items = nil
			for _, item := range vv {
				item := item
				s.Items = append(s.Items, NewListItem(0, 0, 0, 0, item, FG_BLACK, BG_CONTENT, nil))
			}
		}
	}
	if v, ok := args["on_select"]; ok {
		if f, ok := v.(func(int)); ok {
			for i := range s.Items {
				idx := i
				s.Items[i].Callback = func() { f(idx) }
			}
		}
	}
	if v, ok := args["on_back"]; ok {
		if f, ok := v.(func()); ok {
			s.OnBack = f
		}
	}
	s.scroll = 0
	s.TotalItems = len(s.Items)
}

func (s *HomeScreen) Render(d *display.Display) {
	d.Clear(BG_CONTENT)

	s.Sidebar.contentH = d.Height
	s.Sidebar.Render(d)

	d.FillRect(SIDEBAR_W, 0, d.Width-SIDEBAR_W, HEADER_H, BG_HEADER)
	d.DrawTextPixel(SIDEBAR_W+12, 12, FT_TITLE, s.Title, FG_WHITE, BG_HEADER)

	contentW := d.Width - CONTENT_X - 6
	startY := HEADER_H + 8
	itemH := 48
	itemGap := 4
	visible := (d.Height - HEADER_H - 56) / (itemH + itemGap)
	if visible < 1 {
		visible = 1
	}

	if s.scroll > 0 {
		upY := startY
		d.FillRoundRect(CONTENT_X, upY, contentW, 32, 6, BG_CARD)
		d.DrawTextPixel(CONTENT_X+contentW/2-8, upY+8, FT_SMALL, "  ^  ", FG_MUTED, BG_CARD)
		startY += 36
		visible--
	}

	maxScroll := s.TotalItems - visible
	if maxScroll < 0 {
		maxScroll = 0
	}

	endIdx := s.scroll + visible
	if endIdx > s.TotalItems {
		endIdx = s.TotalItems
	}

	y := startY
	for i := s.scroll; i < endIdx; i++ {
		s.Items[i].X = CONTENT_X
		s.Items[i].Y = y
		s.Items[i].W = contentW
		s.Items[i].H = itemH
		s.Items[i].Render(d)
		y += itemH + itemGap
	}

	if s.TotalItems > visible {
		sb := &ScrollBar{
			X:       CONTENT_X + contentW - 6,
			Y:       startY,
			H:       (visible * (itemH + itemGap)) - itemGap,
			Total:   s.TotalItems,
			Visible: visible,
			Offset:  s.scroll,
		}
		sb.Render(d)
	}

	if s.TotalItems > visible && endIdx < s.TotalItems {
		downY := d.Height - 52
		d.FillRoundRect(CONTENT_X, downY, contentW, 32, 6, BG_CARD)
		d.DrawTextPixel(CONTENT_X+contentW/2-8, downY+8, FT_SMALL, "  v  ", FG_MUTED, BG_CARD)
	}

	d.Refresh()
}

func (s *HomeScreen) OnTouch(x, y int) bool {
	if x < SIDEBAR_W {
		s.Sidebar.HandleTouch(x, y)
		return true
	}

	if y < HEADER_H {
		if s.OnBack != nil {
			s.OnBack()
		}
		return false
	}

	d := s.App.Display
	itemH := 48
	itemGap := 4
	startY := HEADER_H + 8
	visible := (d.Height - HEADER_H - 56) / (itemH + itemGap)
	if visible < 1 {
		visible = 1
	}

	if s.scroll > 0 && y >= startY && y < startY+32 {
		s.scroll--
		return true
	}

	if s.TotalItems > visible {
		downY := d.Height - 52
		if y >= downY && y < downY+32 {
			s.scroll++
			maxScroll := s.TotalItems - visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if s.scroll > maxScroll {
				s.scroll = maxScroll
			}
			return true
		}
	}

	if s.scroll > 0 {
		startY += 36
	}

	clickedIdx := -1
	itemY := startY
	for i := s.scroll; i < s.TotalItems && i < s.scroll+visible; i++ {
		if y >= itemY && y < itemY+itemH {
			clickedIdx = i
			break
		}
		itemY += itemH + itemGap
	}

	if clickedIdx >= 0 && clickedIdx < len(s.Items) && s.Items[clickedIdx].Callback != nil {
		s.Items[clickedIdx].Callback()
		return false
	}

	return false
}

// ── MessageScreen (sidebar + messages) ───────────────────────────────

type MessageScreen struct {
	App      *App
	Sidebar  Sidebar
	Title    string
	Messages []map[string]interface{}
	OnBack   func()
	scroll   int
}

func NewMessageScreen() *MessageScreen {
	return &MessageScreen{}
}

func (s *MessageScreen) SetApp(a *App) { s.App = a }

func (s *MessageScreen) OnShow(args map[string]interface{}) {
	if v, ok := args["servers"]; ok {
		if vv, ok := v.([]string); ok {
			s.Sidebar.Servers = vv
		}
	}
	if v, ok := args["selected_dm"]; ok {
		if vv, ok := v.(bool); ok {
			s.Sidebar.SelectedDM = vv
		}
	}
	if v, ok := args["server_idx"]; ok {
		if vv, ok := v.(int); ok {
			s.Sidebar.ServerIdx = vv
		}
	}
	if v, ok := args["on_dm_click"]; ok {
		if f, ok := v.(func()); ok {
			s.Sidebar.OnDMClick = f
		}
	}
	if v, ok := args["on_server_click"]; ok {
		if f, ok := v.(func(int)); ok {
			s.Sidebar.OnServerClick = f
		}
	}
	if v, ok := args["on_update_click"]; ok {
		if f, ok := v.(func()); ok {
			s.Sidebar.OnUpdateClick = f
		}
	}
	if v, ok := args["on_settings_click"]; ok {
		if f, ok := v.(func()); ok {
			s.Sidebar.OnSettingsClick = f
		}
	}
	if v, ok := args["title"]; ok {
		if vv, ok := v.(string); ok {
			s.Title = vv
		}
	}
	if v, ok := args["messages"]; ok {
		if vv, ok := v.([]map[string]interface{}); ok {
			s.Messages = vv
		}
	}
	if v, ok := args["on_back"]; ok {
		if f, ok := v.(func()); ok {
			s.OnBack = f
		}
	}
	s.scroll = 0
}

func (s *MessageScreen) Render(d *display.Display) {
	d.Clear(BG_CONTENT)

	s.Sidebar.contentH = d.Height
	s.Sidebar.Render(d)

	d.FillRect(SIDEBAR_W, 0, d.Width-SIDEBAR_W, HEADER_H, BG_HEADER)
	titleText := "#" + strings.TrimPrefix(s.Title, "#")
	d.DrawTextPixel(SIDEBAR_W+12, 12, FT_TITLE, titleText, FG_WHITE, BG_HEADER)

	contentX := CONTENT_X + 4
	contentW := d.Width - contentX - 10
	startY := HEADER_H + 10
	msgH := 60
	msgGap := 10

	type msgView struct {
		author  string
		content string
	}
	var views []msgView
	for _, msg := range s.Messages {
		author := ""
		if a, ok := msg["author"].(map[string]interface{}); ok {
			if u, ok := a["username"].(string); ok {
				author = u
			}
			if author == "" {
				if g, ok := a["global_name"].(string); ok {
					author = g
				}
			}
			if author == "" {
				if g, ok := a["display_name"].(string); ok {
					author = g
				}
			}
		}
		if author == "" {
			author = "Unknown"
		}
		content, _ := msg["content"].(string)
		content = strings.ReplaceAll(content, "\n", " ")
		views = append(views, msgView{author: author, content: content})
	}

	totalLines := len(views)
	if totalLines == 0 {
		d.DrawTextPixel(contentX+12, startY+20, FT_LABEL, "No messages", FG_MUTED, BG_CONTENT)
		d.Refresh()
		return
	}

	visible := (d.Height - HEADER_H - 80) / (msgH + msgGap)
	if visible < 1 {
		visible = 1
	}

	if s.scroll > 0 {
		d.FillRoundRect(contentX, startY, contentW, 32, 6, BG_CARD)
		d.DrawTextPixel(contentX+contentW/2-8, startY+8, FT_SMALL, "  ^  ", FG_MUTED, BG_CARD)
		startY += 40
	}

	maxScroll := totalLines - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scroll > maxScroll {
		s.scroll = maxScroll
	}

	endIdx := s.scroll + visible
	if endIdx > totalLines {
		endIdx = totalLines
	}

	y := startY
	for i := s.scroll; i < endIdx; i++ {
		v := views[i]
		// message card (1px border via 2 fills)
		var bg uint8 = BG_CARD
		if i%2 == 1 {
			bg = GCARD_ALT
		}
		d.FillRect(contentX, y, contentW, msgH, FG_BORDER)
		d.FillRoundRect(contentX+1, y+1, contentW-2, msgH-2, 7, bg)
		// author badge + name
		d.FillRoundRect(contentX+12, y+8, 14, 14, 7, BG_DM)
		d.DrawTextPixel(contentX+34, y+8, FT_LABEL, v.author, FG_BLACK, bg)
		// content
		if len(v.content) > 0 {
			maxChars := contentW / 9
			txt := v.content
			if len(txt) > maxChars {
				txt = txt[:maxChars-1] + "~"
			}
			d.DrawTextPixel(contentX+12, y+36, FT_SMALL, txt, FG_BLACK, bg)
		}
		y += msgH + msgGap
	}

	if totalLines > visible && endIdx < totalLines {
		downY := d.Height - 52
		d.FillRoundRect(contentX, downY, contentW, 32, 6, BG_CARD)
		d.DrawTextPixel(contentX+contentW/2-8, downY+8, FT_SMALL, "  v  ", FG_MUTED, BG_CARD)
	}

	d.Refresh()
}

func (s *MessageScreen) OnTouch(x, y int) bool {
	if x < SIDEBAR_W {
		s.Sidebar.HandleTouch(x, y)
		return true
	}

	if y < HEADER_H {
		if s.OnBack != nil {
			s.OnBack()
		}
		return false
	}

	d := s.App.Display
	msgH := 60
	msgGap := 10
	startY := HEADER_H + 10
	totalLines := len(s.Messages)
	visible := (d.Height - HEADER_H - 80) / (msgH + msgGap)
	if visible < 1 {
		visible = 1
	}

	if s.scroll > 0 && y >= startY && y < startY+32 {
		s.scroll--
		return true
	}

	if totalLines > visible {
		downY := d.Height - 52
		if y >= downY && y < downY+32 {
			s.scroll++
			maxScroll := totalLines - visible
			if maxScroll < 0 {
				maxScroll = 0
			}
			if s.scroll > maxScroll {
				s.scroll = maxScroll
			}
			return true
		}
	}

	return false
}

// ── LoadingScreen ─────────────────────────────────────────────────────

type LoadingScreen struct {
	App *App
	Msg string
}

func NewLoadingScreen() *LoadingScreen { return &LoadingScreen{} }

func (s *LoadingScreen) SetApp(a *App) { s.App = a }

func (s *LoadingScreen) OnShow(args map[string]interface{}) {
	s.Msg = "Loading..."
	if v, ok := args["message"].(string); ok && v != "" {
		s.Msg = v
	}
}

func (s *LoadingScreen) Render(d *display.Display) {
	d.Clear(BG_CONTENT)
	d.FillRect(0, 0, d.Width, HEADER_H, BG_HEADER)
	d.DrawTextPixel(12, 12, FT_TITLE, "KindleCord", FG_WHITE, BG_HEADER)

	d.FillRoundRect(CONTENT_X+40, d.Height/2-24, d.Width-CONTENT_X-80, 48, 10, BG_CARD)
	d.FillRect(CONTENT_X+40, d.Height/2-24, d.Width-CONTENT_X-80, 48, FG_BORDER)
	txt := s.Msg
	tx := CONTENT_X + 40 + (d.Width-CONTENT_X-80-len(txt)*10)/2
	d.DrawTextPixel(tx, d.Height/2-14, FT_LABEL, txt, FG_BLACK, BG_CARD)
	d.Refresh()
}

func (s *LoadingScreen) OnTouch(x, y int) bool { return false }

// ── Dialog ───────────────────────────────────────────────────────────

type Dialog struct {
	App      *App
	Title    string
	Message  string
	OnOK     func()
	OnCancel func()
	items    []Component
}

func NewDialog(title, message string, onOK func()) *Dialog {
	return &Dialog{Title: title, Message: message, OnOK: onOK}
}

func (d2 *Dialog) SetApp(a *App) { d2.App = a }

func (d2 *Dialog) OnShow(args map[string]interface{}) {
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
	if v, ok := args["on_cancel"]; ok {
		if vv, ok := v.(func()); ok {
			d2.OnCancel = vv
		}
	}
	d2.build()
}

func (d2 *Dialog) build() {
	d := d2.App.Display
	d2.items = nil

	y := 64
	for _, line := range strings.Split(d2.Message, "\n") {
		d2.items = append(d2.items, NewLabel(CONTENT_X+12, y, line, FT_LABEL, FG_BLACK))
		y += 26
	}

	if d2.OnCancel != nil {
		btnX := CONTENT_X + (d.Width-CONTENT_X-80)/2
		d2.items = append(d2.items, NewButton(btnX-85, y+12, "Cancel", d2.OnCancel))
		d2.items = append(d2.items, NewButton(btnX+45, y+12, "OK", d2.OnOK))
	} else {
		btnX := CONTENT_X + (d.Width-CONTENT_X-80)/2
		d2.items = append(d2.items, NewButton(btnX, y+12, "OK", d2.OnOK))
	}
}

func (d2 *Dialog) Render(d *display.Display) {
	d.Clear(BG_CONTENT)
	d.FillRect(0, 0, d.Width, HEADER_H, BG_HEADER)
	titleText := d2.Title
	if len(titleText) > 30 {
		titleText = titleText[:29] + "~"
	}
	d.DrawTextPixel(12, 12, FT_TITLE, titleText, FG_WHITE, BG_HEADER)
	for _, c := range d2.items {
		c.Render(d)
	}
	d.Refresh()
}

func (d2 *Dialog) OnTouch(x, y int) bool {
	for _, c := range d2.items {
		if c.Tap(x, y) {
			return false
		}
	}
	return false
}

// ── ErrorScreen ──────────────────────────────────────────────────────

type ErrorScreen struct {
	App    *App
	Title  string
	Lines  []string
	OnQuit func()
	items  []Component
}

func NewErrorScreen(title string, lines []string, onQuit func()) *ErrorScreen {
	return &ErrorScreen{Title: title, Lines: lines, OnQuit: onQuit}
}

func (s *ErrorScreen) SetApp(a *App) { s.App = a }

func (s *ErrorScreen) OnShow(args map[string]interface{}) {
	if v, ok := args["title"]; ok {
		if vv, ok := v.(string); ok {
			s.Title = vv
		}
	}
	if v, ok := args["lines"]; ok {
		if vv, ok := v.([]string); ok {
			s.Lines = vv
		}
	}
	if v, ok := args["on_quit"]; ok {
		if f, ok := v.(func()); ok {
			s.OnQuit = f
		}
	}
	s.build()
}

func (s *ErrorScreen) build() {
	d := s.App.Display
	s.items = nil

	y := 64
	for _, line := range s.Lines {
		if line == "" {
			y += 18
			continue
		}
		s.items = append(s.items, NewLabel(CONTENT_X+12, y, line, FT_SMALL, FG_BLACK))
		y += 26
	}

	btnX := CONTENT_X + (d.Width-CONTENT_X-80)/2
	s.items = append(s.items, NewButton(btnX, d.Height-60, "Exit", s.OnQuit))
}

func (s *ErrorScreen) Render(d *display.Display) {
	d.Clear(BG_CONTENT)
	d.FillRect(0, 0, d.Width, HEADER_H, BG_HEADER)
	titleText := s.Title
	if len(titleText) > 30 {
		titleText = titleText[:29] + "~"
	}
	d.DrawTextPixel(12, 12, FT_TITLE, titleText, FG_WHITE, BG_HEADER)
	for _, c := range s.items {
		c.Render(d)
	}
	d.Refresh()
}

func (s *ErrorScreen) OnTouch(x, y int) bool {
	for _, c := range s.items {
		if c.Tap(x, y) {
			return false
		}
	}
	return false
}
