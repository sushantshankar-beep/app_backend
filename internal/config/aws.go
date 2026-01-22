package config

import (
	"log"
	"os"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

func InitAWSSession() (*session.Session, error) {
	region := os.Getenv("AWS_REGION")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if region == "" {
		region = "us-east-1" 
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			accessKey,
			secretKey,
			"",
		),
	})

	if err != nil {
		log.Printf("Failed to create AWS session: %v", err)
		return nil, err
	}

	return sess, nil
}