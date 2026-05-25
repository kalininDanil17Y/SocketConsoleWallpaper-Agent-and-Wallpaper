package asciiart

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"

	"socket-console-agent/internal/config"
)

func Convert(path string, cfg config.ImagesConfig) (*Frame, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	src, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	w := cfg.ASCIIWidth
	h := cfg.ASCIIHeight
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	charset := []rune(cfg.Charset)
	if len(charset) == 0 {
		charset = []rune(" .:-=+*#%@")
	}

	palette := make([]string, 0, cfg.PaletteSize)
	paletteIndex := make(map[string]int, cfg.PaletteSize)
	rows := make([]Row, h)

	for y := 0; y < h; y++ {
		text := make([]rune, w)
		fg := make([]int, w)
		for x := 0; x < w; x++ {
			r, g, b, _ := dst.At(x, y).RGBA()
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)
			text[x] = brightnessRune(r8, g8, b8, charset)
			colorHex := quantizeHex(r8, g8, b8, cfg.PaletteSize)
			idx, ok := paletteIndex[colorHex]
			if !ok {
				idx = len(palette)
				paletteIndex[colorHex] = idx
				palette = append(palette, colorHex)
			}
			fg[x] = idx
		}
		rows[y] = Row{Text: string(text), FG: fg}
	}

	return &Frame{
		Type:    "ascii_frame",
		Width:   w,
		Height:  h,
		Palette: palette,
		Rows:    rows,
		Source:  filepath.Base(path),
	}, nil
}

func brightnessRune(r, g, b uint8, charset []rune) rune {
	brightness := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
	idx := int(math.Round((brightness / 255) * float64(len(charset)-1)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(charset) {
		idx = len(charset) - 1
	}
	return charset[idx]
}

func quantizeHex(r, g, b uint8, paletteSize int) string {
	if paletteSize < 2 {
		paletteSize = 2
	}
	levels := int(math.Floor(math.Cbrt(float64(paletteSize))))
	if levels < 2 {
		levels = 2
	}
	q := func(v uint8) uint8 {
		if levels <= 1 {
			return v
		}
		step := 255.0 / float64(levels-1)
		return uint8(math.Round(float64(v)/step) * step)
	}
	return fmt.Sprintf("#%02x%02x%02x", q(r), q(g), q(b))
}

func SupportedImage(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp":
		return true
	default:
		return false
	}
}

func averageColor(img image.Image, rect image.Rectangle) color.RGBA {
	var rSum, gSum, bSum, aSum uint64
	var count uint64
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			rSum += uint64(r >> 8)
			gSum += uint64(g >> 8)
			bSum += uint64(b >> 8)
			aSum += uint64(a >> 8)
			count++
		}
	}
	if count == 0 {
		return color.RGBA{}
	}
	return color.RGBA{R: uint8(rSum / count), G: uint8(gSum / count), B: uint8(bSum / count), A: uint8(aSum / count)}
}
