package service

type ImageUploadS3Service interface {
	BuildUploadResponse(imageURL string) map[string]interface{}
}

type imageUploadS3Service struct{}

func NewImageUploadS3Service() ImageUploadS3Service {
	return &imageUploadS3Service{}
}

func (s *imageUploadS3Service) BuildUploadResponse(imageURL string) map[string]interface{} {
	return map[string]interface{}{
		"message": "image uploaded successfully",
		"url":     imageURL,
	}
}
