package s3

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type PDFUploader struct {
	s3Client   *s3.S3
	bucketName string
	folder     string
}

func NewPDFUploader(sess *session.Session, bucketName string, folder string) *PDFUploader {
	return &PDFUploader{
		s3Client:   s3.New(sess),
		bucketName: bucketName,
		folder:     strings.Trim(folder, "/"),
	}
}

func (u *PDFUploader) UploadPDF(pdfBytes []byte, providerID string) (string, error) {
	fileName := fmt.Sprintf("agreement_%s_%d.pdf", providerID, time.Now().UnixMilli())
	key := fmt.Sprintf("%s/%s", u.folder, fileName)

	uploadInput := &s3.PutObjectInput{
		Bucket:      aws.String(u.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(pdfBytes),
		ContentType: aws.String("application/pdf"),
		ACL:         aws.String("public-read"),
	}

	result, err := u.s3Client.PutObject(uploadInput)
	if err != nil {
		return "", fmt.Errorf("failed to upload PDF to S3: %w", err)
	}

	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", u.bucketName, key)
	log.Printf("✅ PDF uploaded successfully: %s, ETag: %s", url, *result.ETag)
	return url, nil
}

func (u *PDFUploader) DeletePDF(pdfURL string) error {
	if pdfURL == "" {
		return nil
	}

	key := extractKeyFromURL(pdfURL, u.bucketName)
	if key == "" {
		return fmt.Errorf("invalid PDF URL format")
	}

	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(u.bucketName),
		Key:    aws.String(key),
	}

	_, err := u.s3Client.DeleteObject(deleteInput)
	if err != nil {
		log.Printf("⚠️ Failed to delete old PDF: %v", err)
	} else {
		log.Printf("🗑️ Old PDF deleted: %s", pdfURL)
	}
	return err
}

func extractKeyFromURL(url, bucketName string) string {
	prefix := fmt.Sprintf("https://%s.s3.amazonaws.com/", bucketName)
	if strings.HasPrefix(url, prefix) {
		return url[len(prefix):]
	}
	return ""
}