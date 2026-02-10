package handlers

import (
 "net/http"
	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	svc *service.InvoiceService
}

func NewInvoiceHandler(s *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{svc: s}
}

func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	bookingID := c.Query("bookingId")

	invoice, err := h.svc.GetInvoice(c.Request.Context(), bookingID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "invoice not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"invoice": invoice,
	})
}
func (h *InvoiceHandler) DownloadInvoice(c *gin.Context) {

	bookingId := c.Query("bookingId")

	pdfURL, err := h.svc.GetInvoicePDF(
		c.Request.Context(),
		bookingId,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, pdfURL)
}


func (h *InvoiceHandler) GenerateInvoice(c *gin.Context) {
	var req struct {
		UserID    string `json:"userId" binding:"required"`
		ServiceID string `json:"serviceId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invoice, err := h.svc.GenerateInvoice(c.Request.Context(), req.UserID, req.ServiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invoice generated successfully",
		"invoice": invoice,
	})
}
