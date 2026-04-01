package imageutil

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"path/filepath"
	"strings"
)

type ProcessOptions struct {
	AllowedMIMEs   []string
	MaxInputBytes  int64
	MaxOutputBytes int64
	MaxWidth       int
	MaxHeight      int
	JPEGQuality    int
	JPEGMinQuality int
}

type ProcessedImage struct {
	Data        []byte
	ContentType string
	Extension   string
	Width       int
	Height      int
}

func ProcessUploadedImage(filename string, data []byte, opts ProcessOptions) (*ProcessedImage, error) {
	if len(data) == 0 {
		return nil, errors.New("image file is empty")
	}
	if opts.MaxInputBytes > 0 && int64(len(data)) > opts.MaxInputBytes {
		return nil, fmt.Errorf("image file is too large (max %d bytes)", opts.MaxInputBytes)
	}

	contentType := http.DetectContentType(data)
	if !isAllowedMime(contentType, opts.AllowedMIMEs) {
		return nil, fmt.Errorf("unsupported image type %q", contentType)
	}

	extension := canonicalExtension(contentType, filename)
	if contentType == "image/gif" || contentType == "image/webp" {
		if opts.MaxOutputBytes > 0 && int64(len(data)) > opts.MaxOutputBytes {
			return nil, fmt.Errorf("%s image file is too large (max %d bytes)", strings.TrimPrefix(contentType, "image/"), opts.MaxOutputBytes)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil && contentType != "image/webp" {
			return nil, fmt.Errorf("failed to decode image config: %w", err)
		}
		return &ProcessedImage{
			Data:        data,
			ContentType: contentType,
			Extension:   extension,
			Width:       cfg.Width,
			Height:      cfg.Height,
		}, nil
	}

	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return nil, errors.New("image has invalid dimensions")
	}

	maxWidth := clampDimension(opts.MaxWidth, width)
	maxHeight := clampDimension(opts.MaxHeight, height)
	targetWidth, targetHeight := fitDimensions(width, height, maxWidth, maxHeight)

	quality := opts.JPEGQuality
	if quality <= 0 {
		quality = 82
	}
	minQuality := opts.JPEGMinQuality
	if minQuality <= 0 {
		minQuality = 55
	}

	currentWidth := targetWidth
	currentHeight := targetHeight
	for {
		candidate := src
		if currentWidth != width || currentHeight != height {
			candidate = resizeImage(src, currentWidth, currentHeight)
		}

		encoded, err := encodeImage(candidate, contentType, quality)
		if err != nil {
			return nil, err
		}
		if opts.MaxOutputBytes <= 0 || int64(len(encoded)) <= opts.MaxOutputBytes {
			return &ProcessedImage{
				Data:        encoded,
				ContentType: contentType,
				Extension:   extension,
				Width:       currentWidth,
				Height:      currentHeight,
			}, nil
		}

		if contentType == "image/jpeg" && quality > minQuality {
			quality -= 7
			if quality < minQuality {
				quality = minQuality
			}
			continue
		}

		nextWidth, nextHeight := shrinkDimensions(currentWidth, currentHeight)
		if nextWidth == currentWidth && nextHeight == currentHeight {
			break
		}
		currentWidth = nextWidth
		currentHeight = nextHeight
	}

	return nil, fmt.Errorf("image file could not be optimized below %d bytes", opts.MaxOutputBytes)
}

func isAllowedMime(contentType string, allowed []string) bool {
	for _, item := range allowed {
		if item == contentType {
			return true
		}
	}
	return false
}

func canonicalExtension(contentType, filename string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return ".bin"
	}
	return ext
}

func clampDimension(limit, original int) int {
	if limit <= 0 || limit > original {
		return original
	}
	return limit
}

func fitDimensions(width, height, maxWidth, maxHeight int) (int, int) {
	if width <= maxWidth && height <= maxHeight {
		return width, height
	}

	widthRatio := float64(maxWidth) / float64(width)
	heightRatio := float64(maxHeight) / float64(height)
	scale := widthRatio
	if heightRatio < scale {
		scale = heightRatio
	}
	if scale <= 0 {
		return width, height
	}

	targetWidth := int(float64(width) * scale)
	targetHeight := int(float64(height) * scale)
	if targetWidth < 1 {
		targetWidth = 1
	}
	if targetHeight < 1 {
		targetHeight = 1
	}
	return targetWidth, targetHeight
}

func shrinkDimensions(width, height int) (int, int) {
	if width <= 1 && height <= 1 {
		return width, height
	}
	nextWidth := int(float64(width) * 0.85)
	nextHeight := int(float64(height) * 0.85)
	if nextWidth < 1 {
		nextWidth = 1
	}
	if nextHeight < 1 {
		nextHeight = 1
	}
	return nextWidth, nextHeight
}

func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + (y*srcHeight)/height
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + (x*srcWidth)/width
			dst.Set(x, y, color.RGBAModel.Convert(src.At(srcX, srcY)))
		}
	}

	return dst
}

func encodeImage(src image.Image, contentType string, quality int) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	switch contentType {
	case "image/jpeg":
		if err := jpeg.Encode(buf, src, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("failed to encode jpeg: %w", err)
		}
	case "image/png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(buf, src); err != nil {
			return nil, fmt.Errorf("failed to encode png: %w", err)
		}
	case "image/gif":
		if err := gif.Encode(buf, src, nil); err != nil {
			return nil, fmt.Errorf("failed to encode gif: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported image type %q", contentType)
	}
	return buf.Bytes(), nil
}
