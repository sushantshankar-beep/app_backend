package s3

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Uploader struct {
	client *s3.Client
	bucket string
}

func NewUploader() (*Uploader, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, err
	}

	return &Uploader{
		client: s3.NewFromConfig(cfg),
		bucket: os.Getenv("AWS_S3_BUCKET"),
	}, nil
}

func (u *Uploader) Upload(ctx context.Context, data []byte, key string) (string, error) {
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
		ACL:    "public-read",
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", u.bucket, key), nil
}

func GenerateKey(prefix string) string {
	return fmt.Sprintf("%s/%d.jpg", prefix, time.Now().UnixNano())
}
