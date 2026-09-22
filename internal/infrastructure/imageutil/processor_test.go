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

func TestCanonicalExtension_KnownContentTypesIgnoreFilename(t *testing.T) {
	// A known content type decides the extension on its own; the uploaded name
	// is not consulted, whatever it contains.
	name := "../../../../etc/cron.d/script.sh"
	assert.Equal(t, ".jpg", canonicalExtension("image/jpeg", name))
	assert.Equal(t, ".png", canonicalExtension("image/png", name))
	assert.Equal(t, ".gif", canonicalExtension("image/gif", name))
	assert.Equal(t, ".webp", canonicalExtension("image/webp", name))
}

func TestCanonicalExtension_UnknownContentTypeRejectsUnusableFilenames(t *testing.T) {
	unusable := []string{
		"",
		"noext",
		"trailing.",
		"nul.pn\x00g",
		"newline.pn\ng",
		"space.p ng",
		"sep.a/b",
		`backslash.a\b`,
		"dotdot..",
		"upper.PNG/../x",
		"overlong.abcdefghij",
		"unicode.pñg",
	}
	for _, filename := range unusable {
		assert.Equal(t, ".bin", canonicalExtension("application/octet-stream", filename), "filename %q", filename)
	}
}

func TestCanonicalExtension_UnknownContentTypeKeepsPlainExtension(t *testing.T) {
	assert.Equal(t, ".svg", canonicalExtension("image/svg+xml", "logo.svg"))
	assert.Equal(t, ".svg", canonicalExtension("image/svg+xml", "logo.SVG"))
	assert.Equal(t, ".mp4", canonicalExtension("video/mp4", "clip.mp4"))
	assert.Equal(t, ".jpeg", canonicalExtension("application/octet-stream", "photo.jpeg"))
}

func TestDetectImageExtension_NamesTheBytes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	buf := bytes.NewBuffer(nil)
	require.NoError(t, png.Encode(buf, src))

	contentType, ext, ok := DetectImageExtension(buf.Bytes())

	assert.True(t, ok)
	assert.Equal(t, "image/png", contentType)
	assert.Equal(t, ".png", ext)
}

func TestDetectImageExtension_RejectsNonImages(t *testing.T) {
	// A name ending in .png does not make the bytes a PNG, and these bytes are
	// what a stored file would be named after.
	for name, data := range map[string][]byte{
		"html":  []byte("<html><script>alert(1)</script></html>"),
		"svg":   []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`),
		"text":  []byte("just some text"),
		"empty": {},
	} {
		contentType, ext, ok := DetectImageExtension(data)

		assert.False(t, ok, "%s detected as %q", name, contentType)
		assert.Empty(t, ext, "%s", name)
	}
}

func TestDetectImageExtension_AcceptsWebP(t *testing.T) {
	contentType, ext, ok := DetectImageExtension(fakeWebPBytes())

	assert.True(t, ok)
	assert.Equal(t, "image/webp", contentType)
	assert.Equal(t, ".webp", ext)
}
