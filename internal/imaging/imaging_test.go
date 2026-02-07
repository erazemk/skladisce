package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func createTestJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func createTestPNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{0, 0, 255, 255})
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func TestProcessJPEG(t *testing.T) {
	data := createTestJPEG(100, 100)
	result, err := Process(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Process JPEG: %v", err)
	}
	if result.MIME != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", result.MIME)
	}
	if len(result.Data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestProcessPNG(t *testing.T) {
	data := createTestPNG(100, 100)
	result, err := Process(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Process PNG: %v", err)
	}
	if result.MIME != "image/jpeg" {
		t.Errorf("expected image/jpeg (always outputs JPEG), got %s", result.MIME)
	}
}

func TestProcessDownscale(t *testing.T) {
	// Create a 2048x2048 image.
	data := createTestJPEG(2048, 2048)
	result, err := Process(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Process large image: %v", err)
	}

	// Decode the result and check dimensions.
	img, _, err := image.Decode(bytes.NewReader(result.Data))
	if err != nil {
		t.Fatalf("decoding result: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() > MaxDimension || bounds.Dy() > MaxDimension {
		t.Errorf("expected max %dx%d, got %dx%d", MaxDimension, MaxDimension, bounds.Dx(), bounds.Dy())
	}
}

func TestProcessSmallImageNotUpscaled(t *testing.T) {
	data := createTestJPEG(50, 50)
	result, err := Process(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Process small image: %v", err)
	}

	img, _, err := image.Decode(bytes.NewReader(result.Data))
	if err != nil {
		t.Fatalf("decoding result: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 50 || bounds.Dy() != 50 {
		t.Errorf("small image should not be resized: got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestProcessInvalidFormat(t *testing.T) {
	_, err := Process(bytes.NewReader([]byte("not an image")))
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestProcessGIFRejected(t *testing.T) {
	// GIF magic bytes.
	_, err := Process(bytes.NewReader([]byte("GIF89a...")))
	if err == nil {
		t.Error("expected error for GIF")
	}
}
