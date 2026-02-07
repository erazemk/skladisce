package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"

	"golang.org/x/image/draw"
)

// MaxDimension is the maximum width or height for stored images.
const MaxDimension = 1024

// JPEGQuality is the compression quality for JPEG output.
const JPEGQuality = 85

// AllowedMIME lists the accepted input MIME types.
var AllowedMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
}

// ProcessResult contains the processed image data.
type ProcessResult struct {
	Data []byte
	MIME string
}

// Process reads image data, validates the format by sniffing bytes,
// downscales if larger than MaxDimension, and re-encodes with compression.
// Always outputs JPEG for consistency and smaller file sizes.
func Process(r io.Reader) (*ProcessResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading image data: %w", err)
	}

	// Sniff actual MIME type from bytes (not trusting client headers).
	detected := http.DetectContentType(data)
	if !AllowedMIME[detected] {
		return nil, fmt.Errorf("unsupported image format: %s (only JPEG and PNG accepted)", detected)
	}

	// Decode the image.
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	// Downscale if needed.
	img = downscale(img, MaxDimension)

	// Re-encode as JPEG.
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: JPEGQuality}); err != nil {
		return nil, fmt.Errorf("encoding JPEG: %w", err)
	}

	return &ProcessResult{
		Data: buf.Bytes(),
		MIME: "image/jpeg",
	}, nil
}

// downscale resizes the image so neither dimension exceeds maxDim.
// Uses high-quality Catmull-Rom interpolation.
// Returns the original image if already within bounds.
func downscale(img image.Image, maxDim int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxDim && h <= maxDim {
		return img
	}

	// Calculate new dimensions preserving aspect ratio.
	newW, newH := w, h
	if w > h {
		newW = maxDim
		newH = int(float64(h) * float64(maxDim) / float64(w))
	} else {
		newH = maxDim
		newW = int(float64(w) * float64(maxDim) / float64(h))
	}

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

func init() {
	// Register decoders (jpeg is registered by default, but be explicit).
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
}
