package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessUploadedImage_ResizesPNG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			src.Set(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: 100, A: 255})
		}
	}

	buf := bytes.NewBuffer(nil)
	require.NoError(t, png.Encode(buf, src))

	result, err := ProcessUploadedImage("test.png", buf.Bytes(), ProcessOptions{
		AllowedMIMEs:   []string{"image/png"},
		MaxInputBytes:  int64(len(buf.Bytes())) + 1024,
		MaxOutputBytes: int64(len(buf.Bytes())) + 1024,
		MaxWidth:       100,
		MaxHeight:      100,
	})

	require.NoError(t, err)
	assert.Equal(t, "image/png", result.ContentType)
	assert.Equal(t, ".png", result.Extension)
	assert.Equal(t, 100, result.Width)
	assert.Equal(t, 50, result.Height)
}

func TestProcessUploadedImage_PreservesGIFData(t *testing.T) {
	palette := color.Palette{color.Black, color.White}
	frame := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	frame.Pix = []uint8{0, 1, 1, 0}
	src := &gif.GIF{
		Image: []*image.Paletted{frame},
		Delay: []int{10},
	}

	buf := bytes.NewBuffer(nil)
	require.NoError(t, gif.EncodeAll(buf, src))

	result, err := ProcessUploadedImage("anim.gif", buf.Bytes(), ProcessOptions{
		AllowedMIMEs:   []string{"image/gif"},
		MaxInputBytes:  int64(len(buf.Bytes())) + 1024,
		MaxOutputBytes: int64(len(buf.Bytes())) + 1024,
	})

	require.NoError(t, err)
	assert.Equal(t, "image/gif", result.ContentType)
	assert.Equal(t, ".gif", result.Extension)
	assert.Equal(t, buf.Bytes(), result.Data)
	assert.Equal(t, 2, result.Width)
	assert.Equal(t, 2, result.Height)
}

func TestProcessUploadedImage_RejectsUnsupportedType(t *testing.T) {
	result, err := ProcessUploadedImage("test.txt", []byte("plain-text"), ProcessOptions{
		AllowedMIMEs: []string{"image/png"},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported image type")
}

func TestProcessUploadedImage_RejectsAllowedButUndecodablePNG(t *testing.T) {
	result, err := ProcessUploadedImage("broken.png", brokenPNGBytes(), ProcessOptions{
		AllowedMIMEs: []string{"image/png"},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func TestProcessUploadedImage_RejectsUndecodableWebP(t *testing.T) {
	result, err := ProcessUploadedImage("broken.webp", fakeWebPBytes(), ProcessOptions{
		AllowedMIMEs: []string{"image/webp"},
	})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode image")
}

func brokenPNGBytes() []byte {
	return []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00,
	}
}

func fakeWebPBytes() []byte {
	return []byte{
		'R', 'I', 'F', 'F',
		0x1a, 0x00, 0x00, 0x00,
		'W', 'E', 'B', 'P',
		'V', 'P', '8', ' ',
		0x0e, 0x00, 0x00, 0x00,
		0x2f, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
}
