package imagemagick

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vincent-petithory/dataurl"
	"go.uber.org/zap/zaptest"
)

func TestResizeImageWithValidationErrors(t *testing.T) {
	var tests = []struct {
		name      string
		imageData *dataurl.DataURL
		err       error
	}{
		{
			name:      "decode image failure",
			imageData: getImage(t, "_fixtures/small-svg.svg", "image/svg+xml"),
			err:       errors.New("failed to decode image"),
		},
		{
			name:      "not supported image type",
			imageData: getImage(t, "_fixtures/some-gif.gif", "image/gif"),
			err:       errors.New("image type not supported"),
		},
	}
	m := prepareTestingEnvironment(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.ResizeImageProportionally(tc.imageData, 1024, 1024)
			assert.NotNil(t, err)
			assert.Equal(t, tc.err, err)
		})
	}
}

func TestResizeImageWillResizeImage(t *testing.T) {
	var tests = []struct {
		name      string
		imageData *dataurl.DataURL
		expectedW int
		expectedH int
	}{
		{
			name:      "resize png 16:9",
			imageData: getImage(t, "_fixtures/green_1920_1080.png", "image/png"),
			expectedW: 1024,
			expectedH: 576,
		},
		{
			name:      "resize jpg 4:3",
			imageData: getImage(t, "_fixtures/pink_1280_960.jpg", "image/jpeg"),
			expectedW: 1024,
			expectedH: 768,
		},
		{
			name:      "horizontally oriented jpg resize non-standard ratio",
			imageData: getImage(t, "_fixtures/red_1124_500.jpg", "image/jpeg"),
			expectedW: 1024,
			expectedH: 455,
		},
		{
			name:      "vertically oriented jpg resize non-standard ratio",
			imageData: getImage(t, "_fixtures/red_500_1124.jpg", "image/jpeg"),
			expectedW: 455,
			expectedH: 1024,
		},
		{
			name:      "small png shouldn't resize",
			imageData: getImage(t, "_fixtures/red_1024_1024.png", "image/png"),
			expectedW: 1024,
			expectedH: 1024,
		},
	}
	m := prepareTestingEnvironment(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newImage, _ := m.ResizeImageProportionally(tc.imageData, 1024, 1024)
			w, h := getImageSize(t, newImage)
			assert.Equal(t, tc.expectedW, w)
			assert.Equal(t, tc.expectedH, h)
		})
	}
}

func TestConvertStringToImageDataURLWithIncorrectStringWillReturnError(t *testing.T) {
	var tests = []struct {
		name             string
		image            string
		expectedResponse string
	}{
		{
			name:             "incorrect MediaType",
			image:            fmt.Sprintf("data:%s;base64,%s", "text/plain", base64.StdEncoding.EncodeToString([]byte(""))),
			expectedResponse: "data type not supported",
		},
		{
			name:             "incorrect base64 format",
			image:            "some string",
			expectedResponse: "failed to parse data url",
		},
	}
	m := prepareTestingEnvironment(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := m.ConvertStringToImageDataURL(tc.image)
			assert.Equal(t, tc.expectedResponse, err.Error())
		})
	}
}

func TestConvertStringToImageDataURL(t *testing.T) {
	m := prepareTestingEnvironment(t)
	response, err := m.ConvertStringToImageDataURL(getImageBase64String(t, "_fixtures/red_1024_1024.png", "image/png"))
	assert.Nil(t, err)
	assert.NotNil(t, response)
}

func prepareTestingEnvironment(t *testing.T) *Magician {
	t.Helper()
	loggerMock := zaptest.NewLogger(t)
	m := &Magician{
		Logger: loggerMock.Sugar(),
	}
	return m
}

func getImage(t *testing.T, filePath string, mimeType string) *dataurl.DataURL {
	t.Helper()
	base64DataURL := getImageBase64String(t, filePath, mimeType)
	data, err := dataurl.DecodeString(base64DataURL)
	assert.Nil(t, err)
	return data
}

func getImageBase64String(t *testing.T, filePath string, mimeType string) string {
	t.Helper()
	imageData, err := os.ReadFile(filePath)
	assert.Nil(t, err)
	base64Data := base64.StdEncoding.EncodeToString(imageData)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
}

func getImageSize(t *testing.T, data *dataurl.DataURL) (width int, height int) {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(data.Data))
	assert.Nil(t, err)
	width = img.Bounds().Dx()
	height = img.Bounds().Dy()
	return width, height
}
