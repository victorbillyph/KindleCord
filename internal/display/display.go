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

// Display handles framebuffer access with fbink (Kindle) or direct mmap/sim.
type Display struct {
	Width  int
	Height int
	Stride int
	BPP    int
	Cols   int
	Rows   int

	buf      []byte
	mmaped   bool
	fd       *os.File
	fbink    string
	simulate bool
	useFbink bool
}

const (
	CellSize = 24
	FBInk    = "/mnt/us/koreader/fbink"
)

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

func detectFBParams() (w, h, stride, bpp int) {
	w, h, stride, bpp = 1448, 1072, 1448, 8
	if data, err := os.ReadFile("/sys/class/graphics/fb0/virtual_size"); err == nil {
		parts := strings.Split(strings.TrimSpace(string(data)), ",")
		if len(parts) == 2 {
			if ww, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && ww > 0 {
				w = ww
			}
			if hh, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && hh > 0 {
				h = hh
			}
		}
	}
	if data, err := os.ReadFile("/sys/class/graphics/fb0/mode"); err == nil {
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
	if stride < w && bpp == 8 {
		stride = w
	}
	return
}

// New creates a Display. Uses fbink on Kindle, mmap/sim on PC.
func New() *Display {
	w, h, stride, bpp := detectFBParams()
	d := &Display{
		Width:  w,
		Height: h,
		Stride: stride,
		BPP:    bpp,
		Cols:   w / CellSize,
		Rows:   h / CellSize,
		fbink:  FBInk,
	}
	if _, err := os.Stat(FBInk); err != nil {
		d.fbink = ""
	}
	if d.fbink != "" {
		// Kindle with fbink - use fbink subprocess (handles rotation, no mmap needed)
		d.useFbink = true
		// Still try to open fb0 for refresh ioctl fallback
		if f, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0); err == nil {
			d.fd = f
		}
		log.Printf("[DISPLAY] fbink mode %dx%d stride=%d bpp=%d cols=%d rows=%d", w, h, stride, bpp, d.Cols, d.Rows)
		return d
	}

	// PC / no fbink - try mmap
	f, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0)
	if err != nil {
		log.Printf("[DISPLAY] /dev/fb0 not available (%v), sim mode %dx%d cols=%d rows=%d", err, w, h, d.Cols, d.Rows)
		d.simulate = true
		d.buf = make([]byte, w*h*4)
		d.BPP = 32
		d.Stride = w * 4
		return d
	}
	d.fd = f
	var size int
	if bpp == 8 {
		size = stride * h
	} else if bpp == 32 || bpp == 24 {
		size = stride * h
	} else {
		size = w * h * 4
	}
	if size <= 0 {
		size = w * h * 4
	}
	mmap, err := syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		log.Printf("[DISPLAY] mmap fail %v, using sim buffer", err)
		d.simulate = true
		d.buf = make([]byte, w*h*4)
		d.BPP = 32
		d.Stride = w * 4
	} else {
		d.buf = mmap
		d.mmaped = true
		log.Printf("[DISPLAY] mmap ok %dx%d stride=%d bpp=%d cols=%d rows=%d size=%d", w, h, stride, bpp, d.Cols, d.Rows, size)
	}
	return d
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

func (d *Display) setPixel(x, y int, color uint8) {
	if x < 0 || x >= d.Width || y < 0 || y >= d.Height {
		return
	}
	if d.BPP == 8 {
		off := y*d.Stride + x
		if off < len(d.buf) {
			d.buf[off] = color
		}
	} else {
		off := y*d.Stride + x*4
		if off+3 < len(d.buf) {
			d.buf[off] = color
			d.buf[off+1] = color
			d.buf[off+2] = color
			if len(d.buf) > off+3 {
				d.buf[off+3] = 0xFF
			}
		}
	}
}

// Clear fills entire screen
func (d *Display) Clear(color uint8) {
	if d.useFbink {
		cname := grayName(color)
		cmd := exec.Command(d.fbink, "-q", "-k", "-B", cname, "-b")
		_ = cmd.Run()
		return
	}
	if d.simulate && d.BPP == 32 {
		for i := 0; i < len(d.buf); i += 4 {
			d.buf[i] = color
			d.buf[i+1] = color
			d.buf[i+2] = color
			if i+3 < len(d.buf) {
				d.buf[i+3] = 0xFF
			}
		}
		return
	}
	if d.BPP == 8 {
		for i := 0; i < len(d.buf) && i < d.Stride*d.Height; i++ {
			d.buf[i] = color
		}
	} else {
		for y := 0; y < d.Height; y++ {
			for x := 0; x < d.Width; x++ {
				d.setPixel(x, y, color)
			}
		}
	}
}

func (d *Display) FillRect(x, y, w, h int, color uint8) {
	x, y, w, h = d.clipRect(x, y, w, h)
	if w <= 0 || h <= 0 {
		return
	}
	if d.useFbink {
		cname := grayName(color)
		region := fmt.Sprintf("top=%d,left=%d,width=%d,height=%d", y, x, w, h)
		cmd := exec.Command(d.fbink, "-q", "-k", region, "-B", cname, "-b")
		_ = cmd.Run()
		return
	}
	if d.BPP == 8 {
		for yy := y; yy < y+h; yy++ {
			off := yy*d.Stride + x
			for xx := 0; xx < w; xx++ {
				if off+xx < len(d.buf) {
					d.buf[off+xx] = color
				}
			}
		}
	} else {
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				d.setPixel(xx, yy, color)
			}
		}
	}
}

func (d *Display) HLine(x, y, w int, color uint8) {
	// use FillRect which handles fbink
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
	if d.useFbink {
		d.FillRect(x, y, 1, 1, color)
		return
	}
	d.setPixel(x, y, color)
}

// DrawText draws text at cell coordinates with fg/bg
func (d *Display) DrawText(cx, cy int, text string, fg, bg uint8) {
	if cx >= d.Cols || cy >= d.Rows || text == "" {
		return
	}
	maxChars := d.Cols - cx
	if maxChars <= 0 {
		return
	}
	if len(text) > maxChars {
		text = text[:maxChars]
	}
	if d.useFbink {
		px := cx * CellSize
		py := cy * CellSize
		fgName := grayName(fg)
		bgName := grayName(bg)
		cmd := exec.Command(d.fbink, "-q", "-b", "-S", "3", "-F", "VGA", "-C", fgName, "-B", bgName, "-X", fmt.Sprintf("%d", px), "-Y", fmt.Sprintf("%d", py), text)
		if err := cmd.Run(); err != nil {
			// fallback without -F
			cmd2 := exec.Command(d.fbink, "-q", "-b", "-S", "3", "-C", fgName, "-B", bgName, "-X", fmt.Sprintf("%d", px), "-Y", fmt.Sprintf("%d", py), text)
			_ = cmd2.Run()
		}
		return
	}
	px := cx * CellSize
	py := cy * CellSize
	for i, ch := range text {
		bx := px + i*CellSize
		by := py
		d.drawChar(bx, by, byte(ch), fg, bg)
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

// InvertRect inverts region
func (d *Display) InvertRect(x, y, w, h int) {
	if d.useFbink {
		d.FillRect(x, y, w, h, 0x00)
		return
	}
	x, y, w, h = d.clipRect(x, y, w, h)
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if d.BPP == 8 {
				off := yy*d.Stride + xx
				if off < len(d.buf) {
					d.buf[off] = 0xFF - d.buf[off]
				}
			} else {
				off := yy*d.Stride + xx*4
				if off+2 < len(d.buf) {
					d.buf[off] = 0xFF - d.buf[off]
					d.buf[off+1] = 0xFF - d.buf[off+1]
					d.buf[off+2] = 0xFF - d.buf[off+2]
				}
			}
		}
	}
}

// Refresh triggers e-ink update.
func (d *Display) Refresh() {
	if d.simulate {
		return
	}
	if d.fbink != "" {
		cmd := exec.Command(d.fbink, "-q", "-s", "-f", "-W", "GC16", "-w")
		_ = cmd.Run()
		return
	}
	if d.fd != nil {
		const mxcfbSendUpdate = 0x4044462E
		type mxcfbUpdateData struct {
			UpdateRegion struct{ Top, Left, Width, Height uint32 }
			WaveformMode         uint32
			UpdateMode           uint32
			UpdateMarker         uint32
			Temp                 int32
			Flags                uint32
			DitherMode           int32
			QuantMode            int32
			AltBufferData        struct{ PhysAddr uint32; Width, Height uint32; AltUpdateRegion struct{ Top, Left, Width, Height uint32 } }
		}
		var upd mxcfbUpdateData
		upd.UpdateRegion.Top = 0
		upd.UpdateRegion.Left = 0
		upd.UpdateRegion.Width = uint32(d.Width)
		upd.UpdateRegion.Height = uint32(d.Height)
		upd.WaveformMode = 2
		upd.UpdateMode = 0
		upd.Temp = 1
		_, _, _ = syscall.Syscall(syscall.SYS_IOCTL, d.fd.Fd(), uintptr(mxcfbSendUpdate), uintptr(unsafe.Pointer(&upd)))
	}
}

// Close unmaps and closes fb
func (d *Display) Close() {
	if d.mmaped && d.buf != nil {
		_ = syscall.Munmap(d.buf)
		d.buf = nil
	}
	if d.fd != nil {
		_ = d.fd.Close()
		d.fd = nil
	}
}

// DebugDump dumps sim buffer as ASCII
func (d *Display) DebugDump() string {
	if !d.simulate {
		return ""
	}
	var sb strings.Builder
	for row := 0; row < d.Rows && row < 20; row++ {
		for col := 0; col < d.Cols && col < 80; col++ {
			px := col*CellSize + CellSize/2
			py := row*CellSize + CellSize/2
			off := py*d.Stride + px*4
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
