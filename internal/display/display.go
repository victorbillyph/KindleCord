package display

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// Display handles framebuffer access through fbink (Kindle) or a sim buffer (PC).
// On Kindle every primitive goes through fbink, which compensates the panel
// rotation (rotate=3) when writing to memory. Writing to /dev/fb0 directly
// does NOT compensate, which is why text appeared mirrored on screen.
type Display struct {
	Width  int
	Height int
	Stride int
	BPP    int
	Cols   int
	Rows   int

	// Sim buffer (PC / no fbink). Filled for debug dumps.
	buf []byte

	fbink    string
	simulate bool
	useFbink bool
	font     string
}

const (
	CellSize = 24
	FBInk    = "/mnt/us/koreader/fbink"
)

var fbFontPaths = []string{
	"/usr/java/lib/fonts/Amazon-Ember-Regular.ttf",
	"/usr/java/lib/fonts/Helvetica_LT_65_Medium.ttf",
	"/usr/java/lib/fonts/LiberationSans-Regular.ttf",
}

func findFont() string {
	for _, p := range fbFontPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

var fbinkPaths = []string{
	"/mnt/us/extensions/KindleCord/bin/fbink",
	"/mnt/us/extensions/KindleCord/fbink",
	"/mnt/us/koreader/fbink",
	"./bin/fbink",
	"./fbink",
	FBInk,
}

func findFbink() string {
	for _, p := range fbinkPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

var grayToFbink = map[uint8]string{
	0x00: "BLACK", 0x11: "GRAY1", 0x22: "GRAY2", 0x33: "GRAY3",
	0x44: "GRAY4", 0x55: "GRAY5", 0x66: "GRAY6", 0x77: "GRAY7",
	0x88: "GRAY8", 0x99: "GRAY9", 0xAA: "GRAYA", 0xBB: "GRAYB",
	0xCC: "GRAYC", 0xDD: "GRAYD", 0xEE: "GRAYE", 0xFF: "WHITE",
}

func grayName(gray uint8) string {
	if v, ok := grayToFbink[gray]; ok {
		return v
	}
	return "WHITE"
}

func getFbSizeIoctl() (w, h, stride, rotate int, ok bool) {
	f, err := os.OpenFile("/dev/fb0", os.O_RDONLY, 0)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	defer f.Close()
	const FBIOGET_VSCREENINFO = 0x4600
	var info [160]byte
	if err := ioctlFBIOInfo(f, &info); err != nil {
		return 0, 0, 0, 0, false
	}
	le := func(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24 }
	w = le(info[0:4])
	h = le(info[4:8])
	stride = le(info[8:12])
	rotate = le(info[136:140])
	if w > 0 && h > 0 && w < 5000 && h < 5000 {
		return w, h, stride, rotate, true
	}
	return 0, 0, 0, 0, false
}

func ioctlFBIOInfo(f *os.File, info *[160]byte) error {
	const FBIOGET_VSCREENINFO = 0x4600
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), FBIOGET_VSCREENINFO, uintptr(unsafe.Pointer(info)))
	if errno != 0 {
		return errno
	}
	return nil
}

func detectFBParams() (w, h, stride, bpp int) {
	w, h, stride, bpp = 1072, 1448, 1088, 8
	if ww, hh, ss, rot, ok := getFbSizeIoctl(); ok {
		w, h = ww, hh
		if ss > 0 {
			stride = ss
		}
		log.Printf("[DISPLAY] ioctl fb %dx%d stride=%d rotate=%d", w, h, stride, rot)
	} else if data, err := os.ReadFile("/sys/class/graphics/fb0/mode"); err == nil {
		m := strings.TrimSpace(string(data))
		if strings.Contains(m, "x") {
			s := m
			if idx := strings.LastIndex(m, ":"); idx >= 0 {
				s = m[idx+1:]
			}
			parts := strings.Split(s, "x")
			if len(parts) == 2 {
				wp := parts[0]
				hp := parts[1]
				if idx := strings.Index(hp, "p"); idx >= 0 {
					hp = hp[:idx]
				}
				if idx := strings.Index(hp, "-"); idx >= 0 {
					hp = hp[:idx]
				}
				if ww, err := strconv.Atoi(wp); err == nil {
					w = ww
				}
				if hh, err := strconv.Atoi(strings.TrimSpace(hp)); err == nil {
					h = hh
				}
			}
		}
	}
	if data, err := os.ReadFile("/sys/class/graphics/fb0/line_length"); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && v > 0 {
			stride = v
		}
	}
	if data, err := os.ReadFile("/sys/class/graphics/fb0/bits_per_pixel"); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && v > 0 {
			bpp = v
		}
	}
	if stride < w {
		stride = w
	}
	return
}

// New creates a Display. On Kindle, routes everything through fbink.
func New() *Display {
	w, h, stride, bpp := detectFBParams()
	d := &Display{
		Width:  w,
		Height: h,
		Stride: stride,
		BPP:    bpp,
		Cols:   w / CellSize,
		Rows:   h / CellSize,
		fbink:  findFbink(),
	}
	if d.fbink != "" {
		if d.Cols < 1 {
			d.Cols = 44
		}
		if d.Rows < 1 {
			d.Rows = 60
		}
		d.useFbink = true
		d.font = findFont()
		log.Printf("[DISPLAY] fbink mode %s %dx%d cols=%d rows=%d font=%s", d.fbink, w, h, d.Cols, d.Rows, d.font)
		return d
	}

	// PC / no fbink - simulate with an 8bpp buffer of the same layout.
	d.simulate = true
	d.buf = make([]byte, w*h)
	d.BPP = 8
	d.Stride = w
	log.Printf("[DISPLAY] sim mode %dx%d cols=%d rows=%d (no fbink)", w, h, d.Cols, d.Rows)
	return d
}

func (d *Display) runFB(args ...string) error {
	if d.fbink == "" {
		return nil
	}
	cmd := exec.Command(d.fbink, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[DISPLAY] fbink %v err=%v out=%s", args, err, strings.TrimSpace(string(out)))
		return err
	}
	return nil
}

func (d *Display) clipRect(x, y, w, h int) (int, int, int, int) {
	if x < 0 {
		w += x
		x = 0
	}
	if y < 0 {
		h += y
		y = 0
	}
	if x+w > d.Width {
		w = d.Width - x
	}
	if y+h > d.Height {
		h = d.Height - y
	}
	if w <= 0 || h <= 0 {
		return 0, 0, 0, 0
	}
	return x, y, w, h
}

// Clear fills entire screen
func (d *Display) Clear(color uint8) {
	if d.useFbink {
		// -k with -B fills the whole screen in the background color
		_ = d.runFB("-q", "-k", "-B", grayName(color), "-b")
		return
	}
	if d.buf != nil {
		for i := 0; i < len(d.buf); i++ {
			d.buf[i] = color
		}
	}
}

// FillRect fills a pixel rectangle
func (d *Display) FillRect(x, y, w, h int, color uint8) {
	x, y, w, h = d.clipRect(x, y, w, h)
	if w <= 0 || h <= 0 {
		return
	}
	if d.useFbink {
		region := fmt.Sprintf("top=%d,left=%d,width=%d,height=%d", y, x, w, h)
		_ = d.runFB("-q", "-k", region, "-B", grayName(color), "-b")
		return
	}
	if d.buf != nil && d.BPP == 8 {
		for yy := y; yy < y+h; yy++ {
			off := yy*d.Stride + x
			for xx := 0; xx < w; xx++ {
				if off+xx < len(d.buf) {
					d.buf[off+xx] = color
				}
			}
		}
	}
}

func (d *Display) HLine(x, y, w int, color uint8) {
	d.FillRect(x, y, w, 1, color)
}

func (d *Display) VLine(x, y, h int, color uint8) {
	d.FillRect(x, y, 1, h, color)
}

func (d *Display) Rect(x, y, w, h int, color uint8, thickness int) {
	for t := 0; t < thickness; t++ {
		d.HLine(x+t, y+t, w-2*t, color)
		d.HLine(x+t, y+h-1-t, w-2*t, color)
		d.VLine(x+t, y+t, h-2*t, color)
		d.VLine(x+w-1-t, y+t, h-2*t, color)
	}
}

func (d *Display) Pixel(x, y int, color uint8) {
	d.FillRect(x, y, 1, 1, color)
}

// DrawText draws text at cell coordinates with fg/bg via fbink.
func (d *Display) DrawText(cx, cy int, text string, fg, bg uint8) {
	d.DrawTextSized(cx, cy, 16, text, fg, bg)
}

// DrawTextSized draws text at cell coordinates with a given font size.
func (d *Display) DrawTextSized(cx, cy, size int, text string, fg, bg uint8) {
	if cx >= d.Cols || cy >= d.Rows || text == "" {
		return
	}
	px := cx * CellSize
	py := cy * CellSize
	d.DrawTextPixel(px, py, size, text, fg, bg)
}

// DrawTextPixel draws text at exact pixel coordinates with a given font size.
func (d *Display) DrawTextPixel(x, y, size int, text string, fg, bg uint8) {
	if text == "" || x >= d.Width || y >= d.Height {
		return
	}
	if d.useFbink {
		font := d.font
		if font == "" {
			font = findFont()
		}
		opts := fmt.Sprintf("regular=%s,px=%d,left=%d,top=%d", font, size, x, y)
		args := []string{"-q", "-b", "-t", opts,
			"-C", grayName(fg), "-B", grayName(bg)}
		_ = d.runFB(append(args, text)...)
		return
	}
	for i, ch := range text {
		bx := x + i*8
		d.drawChar(bx, y, byte(ch), fg, bg)
	}
}

func (d *Display) drawChar(x, y int, ch byte, fg, bg uint8) {
	if ch >= 128 {
		ch = '?'
	}
	glyph := font8x8[ch]
	d.FillRect(x, y, CellSize, CellSize, bg)
	scale := CellSize / 8
	for row := 0; row < 8; row++ {
		bits := glyph[row]
		for col := 0; col < 8; col++ {
			on := (bits & (1 << (7 - col))) != 0
			c := bg
			if on {
				c = fg
			}
			sx := x + col*scale
			sy := y + row*scale
			d.FillRect(sx, sy, scale, scale, c)
		}
	}
}

// InvertRect inverts region (approximated as dark fill in fbink mode)
func (d *Display) InvertRect(x, y, w, h int) {
	if d.useFbink {
		d.FillRect(x, y, w, h, 0x33)
		return
	}
	x, y, w, h = d.clipRect(x, y, w, h)
	if d.buf == nil {
		return
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			off := yy*d.Stride + xx
			if off < len(d.buf) {
				d.buf[off] = 0xFF - d.buf[off]
			}
		}
	}
}

// Refresh triggers e-ink update. Non-flashing GC16 for speed (no full-screen
// flash, no blocking wait).
func (d *Display) Refresh() {
	if d.useFbink {
		_ = d.runFB("-q", "-s", "-W", "GC16")
		return
	}
	// sim mode: nothing to do
}

// Close releases resources
func (d *Display) Close() {}

// DebugDump dumps sim buffer as ASCII
func (d *Display) DebugDump() string {
	if !d.simulate || d.buf == nil {
		return ""
	}
	var sb strings.Builder
	for row := 0; row < d.Rows && row < 20; row++ {
		for col := 0; col < d.Cols && col < 80; col++ {
			px := col*CellSize + CellSize/2
			py := row*CellSize + CellSize/2
			off := py*d.Stride + px
			var v uint8 = 0xFF
			if off < len(d.buf) {
				v = d.buf[off]
			}
			if v < 0x80 {
				sb.WriteByte('#')
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func (d *Display) IsSimulate() bool { return d.simulate }

// GrayToFbink compatibility
func GrayToFbink(gray uint8) string {
	return grayName(gray)
}