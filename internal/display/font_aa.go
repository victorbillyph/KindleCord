package display

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os/exec"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/math/fixed"
)

var (
	goFontRegular *truetype.Font
	goFontBold    *truetype.Font
)

func init() {
	var err error
	goFontRegular, err = freetype.ParseFont(goregular.TTF)
	if err != nil {
		log.Printf("[FONT] parse regular fail: %v", err)
	}
	goFontBold, err = freetype.ParseFont(gobold.TTF)
	if err != nil {
		log.Printf("[FONT] parse bold fail: %v", err)
	}
}

// drawTextAA renders anti-aliased text to the display's buffer.
func (d *Display) DrawTextAA(cx, cy int, text string, fg, bg uint8, bold bool) {
	if d.buf != nil && len(d.buf) > 0 && !d.simulate {
		d.drawTextAABuf(cx, cy, text, fg, bg, bold)
		return
	}
	if d.useFbink {
		// fallback to fbink bitmap
		px := cx * CellSize
		py := cy * CellSize
		fgName := grayName(fg)
		bgName := grayName(bg)
		cmd := exec.Command(d.fbink, "-q", "-b", "-S", "3", "-F", "VGA", "-C", fgName, "-B", bgName, "-X", fmt.Sprintf("%d", px), "-Y", fmt.Sprintf("%d", py), text)
		if err := cmd.Run(); err != nil {
			cmd2 := exec.Command(d.fbink, "-q", "-b", "-S", "3", "-C", fgName, "-B", bgName, "-X", fmt.Sprintf("%d", px), "-Y", fmt.Sprintf("%d", py), text)
			_ = cmd2.Run()
		}
		return
	}
	d.drawTextAABuf(cx, cy, text, fg, bg, bold)
}

func (d *Display) drawTextAABuf(cx, cy int, text string, fg, bg uint8, bold bool) {
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
	// pixel position
	px := cx * CellSize
	py := cy * CellSize
	// font size: 16px normal, 18 bold - smaller to fit safe area and less off-screen
	fontSize := 16.0
	if bold {
		fontSize = 18.0
	}
	// Use freetype if available
	if goFontRegular == nil {
		// fallback bitmap
		for i, ch := range text {
			d.drawChar(px+i*CellSize, py, byte(ch), fg, bg)
		}
		return
	}
	f := goFontRegular
	if bold && goFontBold != nil {
		f = goFontBold
	}
	estW := len(text)*10 + 8
	estH := CellSize
	if estW > d.Width-px {
		estW = d.Width - px
	}
	// Ensure we don't exceed cols width in pixels
	maxPxW := (d.Cols - cx) * CellSize
	if estW > maxPxW {
		estW = maxPxW
	}
	if estW <= 0 || estH <= 0 {
		return
	}
	img := image.NewRGBA(image.Rect(0, 0, estW, estH))
	bgCol := color.RGBA{bg, bg, bg, 255}
	for y := 0; y < estH; y++ {
		for x := 0; x < estW; x++ {
			img.SetRGBA(x, y, bgCol)
		}
	}
	c := freetype.NewContext()
	c.SetDPI(96)
	c.SetFont(f)
	c.SetFontSize(fontSize)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	fgCol := color.RGBA{fg, fg, fg, 255}
	if fg == 0xFF && bg == 0x33 {
		fgCol = color.RGBA{255, 255, 255, 255}
		bgCol = color.RGBA{0x33, 0x33, 0x33, 255}
	}
	c.SetSrc(image.NewUniform(fgCol))
	c.SetHinting(font.HintingFull)
	pt := freetype.Pt(2, 14)
	_, err := c.DrawString(text, pt)
	if err != nil {
		log.Printf("[FONT] draw fail: %v", err)
		return
	}
	// Blit to display buffer with grayscale + alpha blending
	for y := 0; y < estH; y++ {
		for x := 0; x < estW; x++ {
			r, g, b, a := img.RGBAAt(x, y).RGBA()
			// a is alpha of text (0-65535), but we filled bg, so pixel is already blended
			// Convert to gray
			gray := uint8((r>>8 + g>>8 + b>>8) / 3)
			// Blend with bg based on alpha? Already blended in img
			_ = a
			dx := px + x
			dy := py + y
			if dx >= d.Width || dy >= d.Height {
				continue
			}
			// Simple copy: for anti-alias, gray already contains blended fg/bg
			if d.BPP == 8 {
				off := dy*d.Stride + dx
				if off < len(d.buf) {
					// Convert RGB gray to Kindle grayscale (already 0-255)
					d.buf[off] = gray
				}
			} else {
				off := dy*d.Stride + dx*4
				if off+2 < len(d.buf) {
					d.buf[off] = gray
					d.buf[off+1] = gray
					d.buf[off+2] = gray
				}
			}
		}
	}
}

// DrawTextAASimple is a wrapper for ui
func (d *Display) DrawLabelAA(cx, cy int, text string, fg, bg uint8) {
	d.DrawTextAA(cx, cy, text, fg, bg, false)
}
func (d *Display) DrawTitleAA(cx, cy int, text string, fg, bg uint8) {
	d.DrawTextAA(cx, cy, text, fg, bg, true)
}

// Ensure freetype import used
var _ = fixed.Int26_6(0)
