// Package annotate provides EXIF orientation handling for image labelling.
package annotate

import (
	"bytes"
	"encoding/binary"
	"image"
)

// readOrientation reads the EXIF orientation tag (0x0112 in IFD0) from a
// JPEG APP1 segment or standalone TIFF blob. Returns 1 when absent. Only
// JPEG and TIFF carry orientation, so non-JPEG inputs return 1.
func readOrientation(raw []byte) int {
	orient := 1
	walkTIFF(raw, func(b []byte) {
		if v := orientationFromTIFF(b); v > 0 {
			orient = v
		}
	})
	return orient
}

// walkTIFF invokes fn on each JPEG APP1 EXIF block, or the whole input if
// it is already a TIFF structure.
func walkTIFF(raw []byte, fn func([]byte)) {
	if len(raw) < 4 {
		return
	}
	if raw[0] == 'I' || raw[0] == 'M' {
		fn(raw)
		return
	}
	for i := 2; i < len(raw); {
		if raw[i] != 0xFF {
			return
		}
		marker := raw[i+1]
		i += 2
		if marker == 0xDA { // start of scan
			return
		}
		if i+2 > len(raw) {
			return
		}
		size := int(binary.BigEndian.Uint16(raw[i : i+2]))
		seg := raw[i : i+size]
		if marker == 0xE1 && len(seg) >= 6 && bytes.Equal(seg[0:6], []byte("Exif\x00\x00")) {
			fn(seg[6:])
		}
		i += size
	}
}

func orientationFromTIFF(b []byte) int {
	if len(b) < 8 {
		return 0
	}
	var order binary.ByteOrder
	switch {
	case b[0] == 'I' && b[1] == 'I':
		order = binary.LittleEndian
	case b[0] == 'M' && b[1] == 'M':
		order = binary.BigEndian
	default:
		return 0
	}
	ifdOffset := int(order.Uint32(b[4:8]))
	if ifdOffset+2 > len(b) {
		return 0
	}
	numEntries := int(order.Uint16(b[ifdOffset : ifdOffset+2]))
	p := ifdOffset + 2
	for i := 0; i < numEntries && p+12 <= len(b); i, p = i+1, p+12 {
		tag := order.Uint16(b[p : p+2])
		typ := order.Uint16(b[p+2 : p+4])
		count := order.Uint32(b[p+4 : p+8])
		valOff := order.Uint32(b[p+8 : p+12])
		if tag != 0x0112 { // orientation
			continue
		}
		if typ == 3 && count == 1 { // SHORT
			return int(valOff & 0xFFFF)
		}
		if typ == 1 && count == 1 && valOff < 256 { // BYTE
			return int(valOff)
		}
		return 0
	}
	return 0
}

// transform maps source coords to destination coords for an orientation.
type transform func(x, y, w, h int) (int, int)

var (
	flipH       = func(x, y, w, h int) (int, int) { return w - 1 - x, y }
	flipV       = func(x, y, w, h int) (int, int) { return x, h - 1 - y }
	rotate180   = func(x, y, w, h int) (int, int) { return w - 1 - x, h - 1 - y }
	rotate90CW  = func(x, y, w, h int) (int, int) { return h - 1 - y, x }
	rotate90CCW = func(x, y, w, h int) (int, int) { return y, w - 1 - x }
)

func compose(f, g transform) transform {
	return func(x, y, w, h int) (int, int) {
		nx, ny := g(x, y, w, h)
		return f(nx, ny, w, h)
	}
}

var transforms = map[int]transform{
	1: func(x, y, w, h int) (int, int) { return x, y },
	2: flipH,
	3: rotate180,
	4: flipV,
	5: compose(rotate90CW, flipH),
	6: rotate90CW,
	7: compose(rotate90CCW, flipH),
	8: rotate90CCW,
}

// ReadOrientation returns the EXIF orientation value from raw image bytes.
func ReadOrientation(raw []byte) int { return readOrientation(raw) }

// ApplyOrientation rotates src to upright per the EXIF orientation value.
func ApplyOrientation(src image.Image, orient int) image.Image {
	return applyOrientation(src, orient)
}

// applyOrientation returns src transformed to upright per the EXIF
// orientation value.
func applyOrientation(src image.Image, orient int) image.Image {
	t, ok := transforms[orient]
	if !ok || orient == 1 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dstW, dstH := w, h
	if orient >= 5 && orient <= 8 {
		dstW, dstH = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			nx, ny := t(x, y, w, h)
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
