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
	progress float64 // 0.0 to 1.0
	label    string
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
	img := renderCircularProgressImage(200, 200, cp.progress)
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

	img := renderCircularProgressImage(w, h, r.progress.progress)
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

// renderCircularProgressImage creates an image of the circular progress ring
func renderCircularProgressImage(w, h int, progress float64) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	cx := float64(w) / 2
	cy := float64(h) / 2

	outerRadius := math.Min(cx, cy) * 0.9
	innerRadius := outerRadius * 0.7

	bgRingColor := color.RGBA{R: 220, G: 220, B: 220, A: 255}
	progressRingColor := color.RGBA{R: 52, G: 152, B: 255, A: 255}
	innerCircleColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	transparent := color.RGBA{R: 0, G: 0, B: 0, A: 0}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			distance := math.Sqrt(dx*dx + dy*dy)

			var pixelColor color.RGBA

			if distance >= innerRadius && distance <= outerRadius {
				angle := math.Atan2(dy, dx) + math.Pi/2
				if angle < 0 {
					angle += 2 * math.Pi
				}
				normalizedAngle := angle / (2 * math.Pi)

				if normalizedAngle <= progress {
					pixelColor = progressRingColor
				} else {
					pixelColor = bgRingColor
				}
			} else if distance < innerRadius {
				pixelColor = innerCircleColor
			} else {
				pixelColor = transparent
			}

			img.Set(x, y, pixelColor)
		}
	}

	return img
}
