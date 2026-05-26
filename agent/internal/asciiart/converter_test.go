package asciiart

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"socket-console-agent/internal/config"
)

func TestSupportedImage(t *testing.T) {
	tests := map[string]bool{
		"image.PNG":  true,
		"photo.jpg":  true,
		"photo.jpeg": true,
		"anim.gif":   true,
		"bitmap.bmp": true,
		"notes.txt":  false,
		"noext":      false,
	}

	for path, want := range tests {
		if got := SupportedImage(path); got != want {
			t.Fatalf("SupportedImage(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestConvertBuildsFrameFromPNG(t *testing.T) {
	path := writeTestPNG(t, image.NewRGBA(image.Rect(0, 0, 2, 1)), func(img *image.RGBA) {
		img.SetRGBA(0, 0, color.RGBA{R: 0, G: 0, B: 0, A: 255})
		img.SetRGBA(1, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	})

	frame, err := Convert(path, config.ImagesConfig{
		ASCIIWidth:  2,
		ASCIIHeight: 1,
		Charset:     " .#",
		PaletteSize: 8,
	})
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if frame.Type != "ascii_frame" {
		t.Fatalf("frame.Type = %q, want ascii_frame", frame.Type)
	}
	if frame.Width != 2 || frame.Height != 1 {
		t.Fatalf("frame size = %dx%d, want 2x1", frame.Width, frame.Height)
	}
	if frame.Source != filepath.Base(path) {
		t.Fatalf("frame.Source = %q, want %q", frame.Source, filepath.Base(path))
	}
	if len(frame.Rows) != 1 {
		t.Fatalf("len(frame.Rows) = %d, want 1", len(frame.Rows))
	}
	if frame.Rows[0].Text != " #" {
		t.Fatalf("row text = %q, want %q", frame.Rows[0].Text, " #")
	}
	if len(frame.Rows[0].FG) != 2 {
		t.Fatalf("len(row.FG) = %d, want 2", len(frame.Rows[0].FG))
	}
	if len(frame.Palette) != 2 {
		t.Fatalf("len(frame.Palette) = %d, want 2", len(frame.Palette))
	}
}

func TestImagePathsFiltersAndSortsSupportedFiles(t *testing.T) {
	dir := t.TempDir()
	writeEmptyFile(t, filepath.Join(dir, "z.txt"))
	writePNGAt(t, filepath.Join(dir, "b.png"))
	writePNGAt(t, filepath.Join(dir, "a.JPG"))

	paths, err := imagePaths(dir)
	if err != nil {
		t.Fatalf("imagePaths() error = %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("len(paths) = %d, want 2: %v", len(paths), paths)
	}
	if filepath.Base(paths[0]) != "a.JPG" || filepath.Base(paths[1]) != "b.png" {
		t.Fatalf("paths are not sorted/supported-only: %v", paths)
	}
}

func TestCacheKeyIncludesConversionSettings(t *testing.T) {
	cfg := config.ImagesConfig{ASCIIWidth: 10, ASCIIHeight: 5, Charset: "01", PaletteSize: 8}
	changed := cfg
	changed.ASCIIWidth = 11
	modTime := mustParseTime(t, "2026-05-26T00:00:00Z")

	first := cacheKey("image.png", modTime, cfg)
	second := cacheKey("image.png", modTime, changed)
	if first == second {
		t.Fatal("cacheKey did not change after ASCIIWidth changed")
	}
}

func writeTestPNG(t *testing.T, img *image.RGBA, mutate func(*image.RGBA)) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.png")
	if mutate != nil {
		mutate(img)
	}
	writePNG(t, path, img)
	return path
}

func writePNGAt(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.SetRGBA(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	writePNG(t, path, img)
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", path, err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
}

func writeEmptyFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time.Parse(%q) error = %v", value, err)
	}
	return parsed
}
