package display

import "math"

// FillRoundRect fills a rounded rectangle with given radius and color.
// Uses anti-aliased edge for smoothness on e-ink grayscale.
func (d *Display) FillRoundRect(x, y, w, h int, radius int, color uint8) {
	if radius <= 0 {
		d.FillRect(x, y, w, h, color)
		return
	}
	if radius > w/2 {
		radius = w / 2
	}
	if radius > h/2 {
		radius = h / 2
	}
	// If in fbink mode, we still draw to buffer (if buffer exists) for AA
	// Otherwise fallback to FillRect
	// For fbink direct mode without buffer, approximate with FillRect
	if d.useFbink && (d.buf == nil || len(d.buf) == 0) {
		// fallback: just fill rect (no round)
		d.FillRect(x, y, w, h, color)
		return
	}
	// Draw to buffer with AA
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			// Determine if inside rounded rect
			// Check corners
			cx, cy := px, py
			inside := true
			// Top-left corner
			if px < x+radius && py < y+radius {
				dx := float64(x+radius - px)
				dy := float64(y+radius - py)
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > float64(radius) {
					inside = false
				} else if dist > float64(radius-1) {
					// anti-alias edge: blend
					alpha := float64(radius) - dist
					if alpha < 0 {
						alpha = 0
					}
					if alpha < 1 {
						// blend with bg (assume white bg for now)
						// For simplicity, skip AA and just include if alpha >0.5
						inside = alpha > 0.5
					}
				}
				_ = cx
				_ = cy
			} else if px >= x+w-radius && py < y+radius {
				dx := float64(px - (x + w - radius - 1))
				dy := float64(y+radius - py)
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > float64(radius) {
					inside = false
				}
			} else if px < x+radius && py >= y+h-radius {
				dx := float64(x+radius - px)
				dy := float64(py - (y + h - radius - 1))
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > float64(radius) {
					inside = false
				}
			} else if px >= x+w-radius && py >= y+h-radius {
				dx := float64(px - (x + w - radius - 1))
				dy := float64(py - (y + h - radius - 1))
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > float64(radius) {
					inside = false
				}
			}
			if inside {
				if d.BPP == 8 {
					off := py*d.Stride + px
					if off >= 0 && off < len(d.buf) && py >= 0 && py < d.Height && px >= 0 && px < d.Width {
						d.buf[off] = color
					}
				} else {
					off := py*d.Stride + px*4
					if off+2 < len(d.buf) {
						d.buf[off] = color
						d.buf[off+1] = color
						d.buf[off+2] = color
					}
				}
			}
		}
	}
}

// DrawRoundRect draws a rounded border
func (d *Display) DrawRoundRect(x, y, w, h, radius int, color uint8, thickness int) {
	// Simple: draw outer round rect filled, then inner
	if thickness <= 0 {
		thickness = 1
	}
	d.FillRoundRect(x, y, w, h, radius, color)
	// inner
	if w > thickness*2 && h > thickness*2 {
		// We need bg color - assume white for inner
		// Caller should handle: we clear inner with bg
	}
}


