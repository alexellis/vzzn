// Package annotate parses object detections from the model and renders
// them onto an image: box outlines plus text labels from an embedded
// 5x7 bitmap font, using only the standard library.
package annotate

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// Box is one detection with corners normalized to 0-1000 against the
// image's width and height.
type Box struct {
	Label          string
	X0, Y0, X1, Y1 int
}

// Parse extracts detections from a model response, tolerating markdown
// fences and surrounding prose by slicing on the outermost braces.
func Parse(raw string) ([]Box, error) {
	s := strings.TrimSpace(raw)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			s = s[i : j+1]
		}
	}
	var doc struct {
		Objects []struct {
			Label string `json:"label"`
			Box   [4]int `json:"box"`
		} `json:"objects"`
	}
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return nil, fmt.Errorf("parsing model response: %w", err)
	}
	var out []Box
	for _, o := range doc.Objects {
		n := normalize(o.Box)
		if n[2]-n[0] < 2 || n[3]-n[1] < 2 {
			continue
		}
		label := strings.TrimSpace(o.Label)
		if label == "" {
			label = "object"
		}
		out = append(out, Box{Label: label, X0: n[0], Y0: n[1], X1: n[2], Y1: n[3]})
	}
	return out, nil
}

func normalize(c [4]int) (b [4]int) {
	for i := range c {
		b[i] = c[i]
		if b[i] < 0 {
			b[i] = 0
		}
		if b[i] > 1000 {
			b[i] = 1000
		}
	}
	if b[2] < b[0] {
		b[0], b[2] = b[2], b[0]
	}
	if b[3] < b[1] {
		b[1], b[3] = b[3], b[1]
	}
	return b
}

var (
	boxColor   = [3]uint8{0, 255, 0}
	labelBg    = [3]uint8{255, 255, 255}
	labelInk   = [3]uint8{0, 0, 0}
	labelScale = 3
	labelPad   = 4
)

// Render draws every box and its label onto a copy of src.
func Render(src image.Image, boxes []Box) image.Image {
	sb := src.Bounds()
	dst := image.NewRGBA(sb)
	draw.Draw(dst, sb, src, sb.Min, draw.Over)

	t := 2
	if sb.Dx() > 1600 {
		t = 3
	}

	for _, bx := range boxes {
		x0 := sb.Min.X + bx.X0*sb.Dx()/1000
		y0 := sb.Min.Y + bx.Y0*sb.Dy()/1000
		x1 := sb.Min.X + bx.X1*sb.Dx()/1000
		y1 := sb.Min.Y + bx.Y1*sb.Dy()/1000
		if x1-x0 < t*2 || y1-y0 < t*2 {
			continue
		}
		edge(dst, x0, y0, x1, y1, t, boxColor)
		drawLabel(dst, sb, bx.Label, x0, y0, labelScale)
	}
	return dst
}

func edge(dst *image.RGBA, x0, y0, x1, y1, t int, c [3]uint8) {
	uniform := image.NewUniform(color.RGBA{c[0], c[1], c[2], 255})
	draw.Draw(dst, image.Rect(x0, y0, x1, y0+t), uniform, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x0, y1-t, x1, y1), uniform, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x0, y0, x0+t, y1), uniform, image.Point{}, draw.Over)
	draw.Draw(dst, image.Rect(x1-t, y0, x1, y1), uniform, image.Point{}, draw.Over)
}

func drawLabel(dst *image.RGBA, sb image.Rectangle, text string, x, y int, scale int) {
	img, w, h := rasterize(strings.ToUpper(text), scale)
	lx := x
	if lx < sb.Min.X {
		lx = sb.Min.X
	}
	if lx+w > sb.Max.X {
		lx = sb.Max.X - w
	}
	ly := y - h
	if ly < sb.Min.Y {
		ly = y
	}
	if ly+h > sb.Max.Y {
		ly = sb.Max.Y - h
	}
	draw.Draw(dst, image.Rect(lx, ly, lx+w, ly+h), img, image.Point{}, draw.Over)
}
