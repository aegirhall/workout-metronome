// gen-icon generates the app icon for Workout Metronome as a 1024x1024 PNG.
// Run with: go run ./cmd/gen-icon
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const size = 1024

func main() {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	cx, cy := float64(size)/2, float64(size)/2

	// Colors
	bgColor := color.RGBA{R: 28, G: 28, B: 32, A: 255}         // near-black background
	ringBg := color.RGBA{R: 55, G: 55, B: 65, A: 255}          // dark ring track
	ringFill := color.RGBA{R: 52, G: 152, B: 255, A: 255}      // app blue
	ringFillEnd := color.RGBA{R: 30, G: 100, B: 200, A: 255}   // deeper blue for lower arc
	dumbbellColor := color.RGBA{R: 230, G: 230, B: 240, A: 255} // near-white dumbbell

	// Fill background with rounded rectangle
	cornerRadius := float64(size) * 0.18
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if inRoundedRect(float64(x), float64(y), 0, 0, size, size, cornerRadius) {
				img.SetRGBA(x, y, bgColor)
			}
		}
	}

	// Ring dimensions
	outerR := cx * 0.72
	innerR := outerR * 0.68

	// Draw ring: ~75% filled (progress arc from top, clockwise)
	fillFraction := 0.72
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !inRoundedRect(float64(x), float64(y), 0, 0, size, size, cornerRadius) {
				continue
			}
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < innerR || dist > outerR {
				continue
			}
			angle := math.Atan2(dy, dx) + math.Pi/2
			if angle < 0 {
				angle += 2 * math.Pi
			}
			norm := angle / (2 * math.Pi)
			if norm <= fillFraction {
				// Interpolate from ringFill (top) to ringFillEnd (bottom) based on norm
				t := norm / fillFraction
				c := lerpColor(ringFill, ringFillEnd, t)
				img.SetRGBA(x, y, c)
			} else {
				img.SetRGBA(x, y, ringBg)
			}
		}
	}

	// Round end caps on the arc
	capR := (outerR - innerR) / 2
	midR := (outerR + innerR) / 2

	// Start cap: top (angle = -π/2 in standard coords, i.e. straight up)
	startAngle := -math.Pi / 2
	startCapX := cx + midR*math.Cos(startAngle)
	startCapY := cy + midR*math.Sin(startAngle)
	drawFilledCircle(img, startCapX, startCapY, capR, ringFill,
		func(x, y int) bool {
			return inRoundedRect(float64(x), float64(y), 0, 0, size, size, cornerRadius)
		})

	// End cap: at fillFraction angle
	endAngle := fillFraction*2*math.Pi - math.Pi/2
	endCapX := cx + midR*math.Cos(endAngle)
	endCapY := cy + midR*math.Sin(endAngle)
	drawFilledCircle(img, endCapX, endCapY, capR, ringFillEnd,
		func(x, y int) bool {
			return inRoundedRect(float64(x), float64(y), 0, 0, size, size, cornerRadius)
		})

	// Dumbbell in the center
	drawDumbbell(img, cx, cy, dumbbellColor,
		func(x, y int) bool {
			return inRoundedRect(float64(x), float64(y), 0, 0, size, size, cornerRadius)
		})

	f, err := os.Create("cmd/metronome-gui/Icon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// drawDumbbell draws a simple horizontal dumbbell centered at (cx, cy).
func drawDumbbell(img *image.RGBA, cx, cy float64, c color.RGBA, clip func(int, int) bool) {
	barHalfLen := float64(size) * 0.155
	barHalfH := float64(size) * 0.028
	plateHalfW := float64(size) * 0.048
	plateHalfH := float64(size) * 0.135
	collarHalfW := float64(size) * 0.025
	collarHalfH := float64(size) * 0.062

	// Bar
	fillRect(img, cx-barHalfLen, cy-barHalfH, cx+barHalfLen, cy+barHalfH, c, clip)

	// Left plate pair
	fillRect(img, cx-barHalfLen-plateHalfW*2, cy-plateHalfH, cx-barHalfLen-plateHalfW, cy+plateHalfH, c, clip)
	fillRect(img, cx-barHalfLen-plateHalfW*4, cy-plateHalfH*0.78, cx-barHalfLen-plateHalfW*3, cy+plateHalfH*0.78, c, clip)
	// Left collar
	fillRect(img, cx-barHalfLen-collarHalfW, cy-collarHalfH, cx-barHalfLen+collarHalfW, cy+collarHalfH, c, clip)

	// Right plate pair
	fillRect(img, cx+barHalfLen+plateHalfW, cy-plateHalfH, cx+barHalfLen+plateHalfW*2, cy+plateHalfH, c, clip)
	fillRect(img, cx+barHalfLen+plateHalfW*3, cy-plateHalfH*0.78, cx+barHalfLen+plateHalfW*4, cy+plateHalfH*0.78, c, clip)
	// Right collar
	fillRect(img, cx+barHalfLen-collarHalfW, cy-collarHalfH, cx+barHalfLen+collarHalfW, cy+collarHalfH, c, clip)
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA, clip func(int, int) bool) {
	for y := int(math.Floor(y0)); y <= int(math.Ceil(y1)); y++ {
		for x := int(math.Floor(x0)); x <= int(math.Ceil(x1)); x++ {
			if x >= 0 && x < size && y >= 0 && y < size && clip(x, y) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawFilledCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA, clip func(int, int) bool) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			if x < 0 || x >= size || y < 0 || y >= size {
				continue
			}
			dx := float64(x) - cx
			dy := float64(y) - cy
			if dx*dx+dy*dy <= r*r && clip(x, y) {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func inRoundedRect(px, py, x0, y0 float64, w, h int, r float64) bool {
	x1 := x0 + float64(w)
	y1 := y0 + float64(h)
	if px < x0 || px > x1 || py < y0 || py > y1 {
		return false
	}
	// Check corners
	corners := [][2]float64{
		{x0 + r, y0 + r},
		{x1 - r, y0 + r},
		{x0 + r, y1 - r},
		{x1 - r, y1 - r},
	}
	for _, c := range corners {
		if px < c[0]-r || px > c[0]+r || py < c[1]-r || py > c[1]+r {
			// In a corner zone?
			if (px < x0+r || px > x1-r) && (py < y0+r || py > y1-r) {
				dx := px - c[0]
				dy := py - c[1]
				if dx*dx+dy*dy > r*r {
					return false
				}
			}
		}
	}
	return true
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(a, b uint8, t float64) uint8 {
		return uint8(float64(a) + t*(float64(b)-float64(a)))
	}
	return color.RGBA{
		R: lerp(a.R, b.R, t),
		G: lerp(a.G, b.G, t),
		B: lerp(a.B, b.B, t),
		A: lerp(a.A, b.A, t),
	}
}
