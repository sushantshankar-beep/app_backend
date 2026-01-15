package utils

import (
	"bytes"
	"image"
	"image/jpeg"
	"mime/multipart"

	"github.com/nfnt/resize"
)

func CompressImage(file multipart.File) ([]byte, error) {
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	resized := resize.Resize(1024, 0, img, resize.Lanczos3)

	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, resized, &jpeg.Options{Quality: 70})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
