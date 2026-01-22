package s3

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Uploader struct {
	s3Client   *s3.S3
	bucketName string
	folder     string
}

type FieldConfig struct {
	FormFieldName string
	ContextKey    string
}

func NewUploader(sess *session.Session, bucketName string, folder string) *Uploader {
	return &Uploader{
		s3Client:   s3.New(sess),
		bucketName: bucketName,
		folder:     strings.Trim(folder, "/"),
	}
}

func (u *Uploader) compressImage(fileBytes []byte, contentType string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(fileBytes))
	if err != nil {
		return fileBytes, err
	}

	resized := resize.Resize(1920, 0, img, resize.Lanczos3)

	var compressed bytes.Buffer
	switch contentType {
	case "image/png":
		err = png.Encode(&compressed, resized)
	default:
		err = jpeg.Encode(&compressed, resized, &jpeg.Options{Quality: 80})
	}

	if err != nil {
		return fileBytes, err
	}

	if compressed.Len() < len(fileBytes) {
		return compressed.Bytes(), nil
	}
	return fileBytes, nil
}

func (u *Uploader) Upload(fieldConfigs []FieldConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		form, err := c.MultipartForm()
		if err != nil {
			c.Next()
			return
		}

		uploadedURLs := make(map[string][]string)

		for _, config := range fieldConfigs {
			files := form.File[config.FormFieldName]
			if len(files) == 0 {
				continue
			}

			var urls []string
			for _, fileHeader := range files {
				file, err := fileHeader.Open()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": fmt.Sprintf("Failed to open file: %v", err),
					})
					c.Abort()
					return
				}
				defer file.Close()

				buf := new(bytes.Buffer)
				if _, err := buf.ReadFrom(file); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": fmt.Sprintf("Failed to read file: %v", err),
					})
					c.Abort()
					return
				}

				fileBytes := buf.Bytes()
				contentType := fileHeader.Header.Get("Content-Type")
				ext := strings.ToLower(filepath.Ext(fileHeader.Filename))

				if contentType == "image/jpeg" || contentType == "image/jpg" || contentType == "image/png" ||
					ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
					compressedBytes, err := u.compressImage(fileBytes, contentType)
					if err == nil {
						fileBytes = compressedBytes
					} else {
						log.Println("COMPRESSION FAILED", err)
					}
				}

				safeName := strings.ReplaceAll(fileHeader.Filename, " ", "_")

                key := fmt.Sprintf("%s/%d_%s",u.folder,time.Now().UnixMilli(),safeName)

				uploadInput := &s3.PutObjectInput{
					Bucket:      aws.String(u.bucketName),
					Key:         aws.String(key),
					Body:        bytes.NewReader(fileBytes),
					ContentType: aws.String(contentType),
					ACL:         aws.String("public-read"),
				}

				result, err := u.s3Client.PutObject(uploadInput)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"message": fmt.Sprintf("Failed to upload to S3: %v", err),
					})
					c.Abort()
					return
				}

				url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", u.bucketName, key)
				urls = append(urls, url)

				fmt.Printf("Uploaded file: %s, ETag: %s\n", url, *result.ETag)
			}

			if len(urls) > 0 {
				uploadedURLs[config.ContextKey] = urls
			}
		}

		for key, urls := range uploadedURLs {
			c.Set(key, urls)
		}

		c.Next()
	}
}

func GetUploadedURL(c *gin.Context, key string) (string, bool) {
	value, exists := c.Get(key)
	if !exists {
		return "", false
	}

	url, ok := value.(string)
	return url, ok
}

func GetUploadedURLs(c *gin.Context, key string) ([]string, bool) {
	value, exists := c.Get(key)
	if !exists {
		return nil, false
	}

	urls, ok := value.([]string)
	return urls, ok
}
