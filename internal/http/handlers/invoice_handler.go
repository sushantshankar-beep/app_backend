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
	invoiceID := c.Param("invoiceId")

	invoice, err := h.svc.GetInvoice(c.Request.Context(), invoiceID)
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
	invoiceID := c.Param("invoiceId")

	pdfPath, filename, err := h.svc.GetInvoicePDF(c.Request.Context(), invoiceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/pdf")
	c.File(pdfPath)
}