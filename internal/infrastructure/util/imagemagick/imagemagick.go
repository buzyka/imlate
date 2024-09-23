package imagemagick

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
	"github.com/vincent-petithory/dataurl"
	"gitlab.shopware.com/shopware/6/services/ai-proxy/util"
	"go.uber.org/zap"
)

var allowedImageTypes = []string{"jpeg", "jpg", "png"}

type Magician struct {
	Logger *zap.SugaredLogger
}

func (m *Magician) ConvertStringToImageDataURL(str string) (*dataurl.DataURL, error) {
	data, err := dataurl.DecodeString(str)
	if err != nil {
		m.Logger.Error(fmt.Sprintf("Error while parsing data url: %s", err.Error()))
		return nil, errors.New("failed to parse data url")
	}

	if data.MediaType.Type != "image" {
		m.Logger.Warnf(fmt.Sprintf("Data type not supported: %s", data.MediaType.Type))
		return nil, errors.New("data type not supported")
	}

	return data, nil
}

func (m *Magician) ResizeImageProportionally(data *dataurl.DataURL, maxImageWidth int, maxImageHeight int) (*dataurl.DataURL, error) {
	img, format, err := image.Decode(bytes.NewReader(data.Data))
	if err != nil {
		m.Logger.Error(fmt.Sprintf("Error while decoding image: %s", err.Error()))
		return nil, errors.New("failed to decode image")
	}

	if !util.InArray(format, allowedImageTypes) {
		m.Logger.Warnf(fmt.Sprintf("Image type not supported: %s", format))
		return nil, errors.New("image type not supported")
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	if width <= maxImageWidth && height <= maxImageHeight {
		return data, nil
	}

	aspectRatio := float64(width) / float64(height)
	var newHeight int
	var newWidth int

	if aspectRatio > 1 {
		newWidth = maxImageWidth
		newHeight = int(float64(newWidth) / aspectRatio)
	} else {
		newHeight = maxImageHeight
		newWidth = int(float64(newHeight) * aspectRatio)
	}

	resizedImage := imaging.Resize(img, newWidth, newHeight, imaging.NearestNeighbor)

	buff := new(bytes.Buffer)

	switch format {
	case "jpeg", "jpg":
		err = imaging.Encode(buff, resizedImage, imaging.JPEG)
	case "png":
		err = imaging.Encode(buff, resizedImage, imaging.PNG)
	}

	if err != nil {
		m.Logger.Error(fmt.Sprintf("Error while encoding png image: %s", err.Error()))
		return nil, errors.New("failed to encode image")
	}

	return dataurl.New(buff.Bytes(), data.MediaType.String()), nil
}
