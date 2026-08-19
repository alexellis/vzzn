package annotate

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// glyphData holds a 5x7 bitmap font: seven rows of five '0'/'1' characters,
// top to bottom, left to right. Lowercase aliases are resolved to uppercase.
var glyphData = map[rune]string{
	'A': "01110\n10001\n10001\n11111\n10001\n10001\n10001",
	'B': "11110\n10001\n10001\n11110\n10001\n10001\n11110",
	'C': "01110\n10001\n10000\n10000\n10000\n10001\n01110",
	'D': "11110\n10001\n10001\n10001\n10001\n10001\n11110",
	'E': "11111\n10000\n10000\n11110\n10000\n10000\n11111",
	'F': "11111\n10000\n10000\n11110\n10000\n10000\n10000",
	'G': "01110\n10000\n10000\n10111\n10001\n10001\n01110",
	'H': "10001\n10001\n10001\n11111\n10001\n10001\n10001",
	'I': "11111\n00100\n00100\n00100\n00100\n00100\n11111",
	'J': "00111\n00010\n00010\n00010\n00010\n10010\n01100",
	'K': "10001\n10010\n10100\n11000\n10100\n10010\n10001",
	'L': "10000\n10000\n10000\n10000\n10000\n10000\n11111",
	'M': "10001\n11011\n10101\n10101\n10001\n10001\n10001",
	'N': "10001\n11001\n10101\n10011\n10001\n10001\n10001",
	'O': "01110\n10001\n10001\n10001\n10001\n10001\n01110",
	'P': "11110\n10001\n10001\n11110\n10000\n10000\n10000",
	'Q': "01110\n10001\n10001\n10001\n10101\n10010\n01101",
	'R': "11110\n10001\n10001\n11110\n10100\n10010\n10001",
	'S': "01111\n10000\n10000\n01110\n00001\n00001\n11110",
	'T': "11111\n00100\n00100\n00100\n00100\n00100\n00100",
	'U': "10001\n10001\n10001\n10001\n10001\n10001\n01110",
	'V': "10001\n10001\n10001\n10001\n10001\n01010\n00100",
	'W': "10001\n10001\n10001\n10101\n10101\n10101\n01010",
	'X': "10001\n10001\n01010\n00100\n01010\n10001\n10001",
	'Y': "10001\n10001\n01010\n00100\n00100\n00100\n00100",
	'Z': "11111\n00001\n00010\n00100\n01000\n10000\n11111",
	'0': "01110\n10001\n10011\n10101\n11001\n10001\n01110",
	'1': "00100\n01100\n00100\n00100\n00100\n00100\n01110",
	'2': "01110\n10001\n00001\n00010\n00100\n01000\n11111",
	'3': "01110\n10001\n00001\n00110\n00001\n10001\n01110",
	'4': "00010\n00110\n01010\n10010\n11111\n00010\n00010",
	'5': "11111\n10000\n10000\n11110\n00001\n00001\n01110",
	'6': "00110\n01000\n10000\n11110\n10001\n10001\n01110",
	'7': "11111\n00001\n00010\n00100\n01000\n01000\n01000",
	'8': "01110\n10001\n10001\n01110\n10001\n10001\n01110",
	'9': "01110\n10001\n10001\n01111\n00001\n00010\n01100",
	' ': "\n\n\n\n\n\n",
	'-': "\n\n\n01110\n\n\n",
	'.': "\n\n\n\n\n01100\n01100",
	':': "\n00100\n\n\n00100\n\n",
	'/': "\n00001\n00010\n00100\n01000\n\n",
	'(': "00010\n00100\n01000\n01000\n01000\n00100\n00010",
	')': "00100\n00010\n00001\n00001\n00001\n00010\n00100",
	',': "\n\n\n00100\n00100\n00100\n01100",
	'!': "00100\n00100\n00100\n00100\n00100\n\n00100",
	'?': "01110\n10001\n00001\n00110\n00100\n\n00100",
}

type glyph [7][5]bool

var font = buildFont()

func buildFont() map[rune]glyph {
	out := make(map[rune]glyph, len(glyphData))
	for r, s := range glyphData {
		rows := strings.Split(s, "\n")
		if len(rows) != 7 {
			continue
		}
		var g glyph
		ok := true
		for i, row := range rows {
			if len(row) != 5 {
				ok = false
				break
			}
			for j := 0; j < 5; j++ {
				g[i][j] = row[j] == '1'
			}
		}
		if ok {
			out[r] = g
		}
	}
	for c := 'a'; c <= 'z'; c++ {
		if g, ok := out[c-'a'+'A']; ok {
			out[c] = g
		}
	}
	return out
}

// rasterize renders text at the given scale with a white background and
// black ink, returning the image and its dimensions.
func rasterize(text string, scale int) (*image.RGBA, int, int) {
	const cellW, cellH = 5, 7
	w := len(text)*cellW*scale + 2*labelPad
	h := cellH*scale + 2*labelPad
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	bg := image.NewUniform(color.RGBA{labelBg[0], labelBg[1], labelBg[2], 255})
	draw.Draw(img, img.Bounds(), bg, image.Point{}, draw.Over)
	ink := image.NewUniform(color.RGBA{labelInk[0], labelInk[1], labelInk[2], 255})
	for i, r := range text {
		g, ok := font[r]
		if !ok {
			g, _ = font[' ']
		}
		for row := 0; row < cellH; row++ {
			for col := 0; col < cellW; col++ {
				if !g[row][col] {
					continue
				}
				x := labelPad + (i*cellW+col)*scale
				y := labelPad + row*scale
				draw.Draw(img, image.Rect(x, y, x+scale, y+scale), ink, image.Point{}, draw.Over)
			}
		}
	}
	return img, w, h
}
