package main

import (
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// CircularProgress is a custom widget that displays a circular progress ring
// with a large label rendered in the center.
type CircularProgress struct {
	widget.BaseWidget
	progress  float64 // 0.0 to 1.0
	label     string
	splitMode bool // when true, draws a divider at the 50% mark
}

// NewCircularProgress creates a new circular progress widget
func NewCircularProgress() *CircularProgress {
	cp := &CircularProgress{}
	cp.ExtendBaseWidget(cp)
	return cp
}

// SetProgress updates the progress value (0.0 to 1.0)
func (cp *CircularProgress) SetProgress(progress float64) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	cp.progress = progress
	cp.Refresh()
}

// SetLabel updates the center label text
func (cp *CircularProgress) SetLabel(label string) {
	cp.label = label
	cp.Refresh()
}

// SetSplitMode enables or disables the midpoint divider line.
func (cp *CircularProgress) SetSplitMode(split bool) {
	cp.splitMode = split
	cp.Refresh()
}

// SetProgressAndLabel updates both progress and label atomically
func (cp *CircularProgress) SetProgressAndLabel(progress float64, label string) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	cp.progress = progress
	cp.label = label
	cp.Refresh()
}

// CreateRenderer implements the fyne.Widget interface
func (cp *CircularProgress) CreateRenderer() fyne.WidgetRenderer {
	img := renderCircularProgressImage(200, 200, cp.progress, cp.splitMode)
	raster := canvas.NewImageFromImage(img)
	raster.FillMode = canvas.ImageFillContain

	text := canvas.NewText(cp.label, color.Black)
	text.Alignment = fyne.TextAlignCenter
	text.TextStyle = fyne.TextStyle{Bold: true}

	return &circularProgressRenderer{
		progress: cp,
		raster:   raster,
		text:     text,
	}
}

// MinSize returns the minimum size of the widget
func (cp *CircularProgress) MinSize() fyne.Size {
	return fyne.NewSize(200, 200)
}

type circularProgressRenderer struct {
	progress *CircularProgress
	raster   *canvas.Image
	text     *canvas.Text
}

func (r *circularProgressRenderer) Layout(size fyne.Size) {
	r.raster.Resize(size)
	r.raster.Move(fyne.NewPos(0, 0))

	r.text.TextSize = size.Width * 0.32
	textH := fyne.MeasureText(r.text.Text, r.text.TextSize, r.text.TextStyle).Height
	r.text.Resize(fyne.NewSize(size.Width, textH))
	r.text.Move(fyne.NewPos(0, (size.Height-textH)/2))
}

func (r *circularProgressRenderer) MinSize() fyne.Size {
	return r.progress.MinSize()
}

func (r *circularProgressRenderer) Refresh() {
	size := r.raster.Size()
	w := int(size.Width)
	h := int(size.Height)
	if w == 0 || h == 0 {
		w, h = 200, 200
	}

	img := renderCircularProgressImage(w, h, r.progress.progress, r.progress.splitMode)
	r.raster.Image = img
	r.raster.Refresh()

	r.text.Text = r.progress.label
	r.text.TextSize = size.Width * 0.32
	textH := fyne.MeasureText(r.text.Text, r.text.TextSize, r.text.TextStyle).Height
	r.text.Resize(fyne.NewSize(size.Width, textH))
	r.text.Move(fyne.NewPos(0, (size.Height-textH)/2))
	r.text.Refresh()
}

func (r *circularProgressRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.raster, r.text}
}

func (r *circularProgressRenderer) Destroy() {
}

func (r *circularProgressRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

// renderCircularProgressImage creates an anti-aliased image of the circular progress ring
// by rendering at 3x resolution and downsampling (supersampling).
func renderCircularProgressImage(w, h int, progress float64, splitMode bool) image.Image {
	const scale = 3
	big := renderRawImage(w*scale, h*scale, progress, splitMode)

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			var r, g, b, a int
			for dy := range scale {
				for dx := range scale {
					c := big.RGBAAt(x*scale+dx, y*scale+dy)
					r += int(c.R)
					g += int(c.G)
					b += int(c.B)
					a += int(c.A)
				}
			}
			n := scale * scale
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(r / n),
				G: uint8(g / n),
				B: uint8(b / n),
				A: uint8(a / n),
			})
		}
	}
	return img
}

// renderRawImage draws the ring at the given resolution without anti-aliasing.
// Called at 3x size by renderCircularProgressImage for supersampling.
func renderRawImage(w, h int, progress float64, splitMode bool) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	cx := float64(w) / 2
	cy := float64(h) / 2

	outerRadius := math.Min(cx, cy) * 0.9
	innerRadius := outerRadius * 0.7

	bgRingColor := color.RGBA{R: 220, G: 220, B: 220, A: 255}
	progressRingColor := color.RGBA{R: 52, G: 152, B: 255, A: 255}
	innerCircleColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	borderColor := color.RGBA{R: 160, G: 160, B: 160, A: 255}
	dividerColor := color.RGBA{R: 160, G: 160, B: 160, A: 255}

	borderWidth := math.Max(1.0, math.Min(float64(w), float64(h))*0.007)
	outerBorderInner := outerRadius - borderWidth
	innerBorderOuter := innerRadius + borderWidth

	// Divider bar: a radial strip at the 50% angle (straight down, angle=π in atan2 coords).
	// We match points whose angle is within half a dividerHalfAngle of the bottom.
	dividerHalfWidth := math.Max(1.0, math.Min(float64(w), float64(h))*0.007)

	for y := range h {
		for x := range w {
			dx := float64(x) - cx
			dy := float64(y) - cy
			distance := math.Sqrt(dx*dx + dy*dy)

			var c color.RGBA
			switch {
			case distance > outerBorderInner && distance <= outerRadius:
				c = borderColor
			case distance >= innerRadius && distance <= innerBorderOuter:
				c = borderColor
			case distance > innerBorderOuter && distance <= outerBorderInner:
				angle := math.Atan2(dy, dx) + math.Pi/2
				if angle < 0 {
					angle += 2 * math.Pi
				}

				// Check if this pixel falls on the split divider (bottom of ring, 50% mark).
				if splitMode && math.Abs(dx) <= dividerHalfWidth && dy > 0 {
					c = dividerColor
				} else if angle/(2*math.Pi) <= progress {
					c = progressRingColor
				} else {
					c = bgRingColor
				}
			case distance < innerRadius:
				c = innerCircleColor
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}
