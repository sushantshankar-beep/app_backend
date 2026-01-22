package handlers

import (
	"net/http"

	"app_backend/internal/s3"
	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
)

type ImageUploadS3Handler struct {
	service service.ImageUploadS3Service
}

func NewUploadHandler(service service.ImageUploadS3Service) *ImageUploadS3Handler {
	return &ImageUploadS3Handler{service: service}
}

func (h *ImageUploadS3Handler) UploadSingle(c *gin.Context) {
	urls, ok := s3.GetUploadedURLs(c, "image")
	if !ok || len(urls) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "image not uploaded",
		})
		return
	}

	response := h.service.BuildUploadResponse(urls[0])
	c.JSON(http.StatusOK, response)
}
