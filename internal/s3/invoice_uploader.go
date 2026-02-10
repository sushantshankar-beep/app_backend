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

type InvoiceUploader struct {
	s3Client   *s3.S3
	bucketName string
	folder     string
}

func NewInvoiceUploader(
	sess *session.Session,
	bucketName string,
	folder string,
) *InvoiceUploader {

	return &InvoiceUploader{
		s3Client:   s3.New(sess),
		bucketName: bucketName,
		folder:     strings.Trim(folder, "/"),
	}
}

// =============================
// Upload Invoice PDF
// =============================
func (u *InvoiceUploader) UploadInvoicePDF(
	pdfBytes []byte,
	serviceID string,
) (string, error) {

	fileName := fmt.Sprintf(
		"invoice_%s_%d.pdf",
		serviceID,
		time.Now().UnixMilli(),
	)

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
		return "", fmt.Errorf("failed to upload invoice PDF: %w", err)
	}

	url := fmt.Sprintf(
		"https://%s.s3.amazonaws.com/%s",
		u.bucketName,
		key,
	)

	log.Printf("✅ Invoice uploaded: %s (etag=%s)", url, *result.ETag)

	return url, nil
}

// =============================
// Delete Invoice PDF
// =============================
func (u *InvoiceUploader) DeleteInvoicePDF(pdfURL string) error {

	if pdfURL == "" {
		return nil
	}

	key := extractKeyFromURL(pdfURL, u.bucketName)
	if key == "" {
		return fmt.Errorf("invalid invoice url")
	}

	_, err := u.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(u.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		log.Printf("⚠️ failed to delete invoice: %v", err)
		return err
	}

	log.Printf("🗑️ invoice deleted: %s", pdfURL)
	return nil
}
