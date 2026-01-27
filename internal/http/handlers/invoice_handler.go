package handlers

import (
 "net/http"
"go.mongodb.org/mongo-driver/bson/primitive"
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
	serviceID := c.Param("serviceId")

	inv, err := h.svc.GetInvoice(c, serviceID)
	if err != nil {
		c.JSON(404, gin.H{"error": "invoice not found"})
		return
	}

	c.JSON(200, gin.H{"data": inv})
}

func (h *InvoiceHandler) DownloadInvoice(c *gin.Context) {
	serviceID := c.Param("serviceId")

	inv, err := h.svc.GetInvoice(c, serviceID)
	if err != nil {
		c.JSON(404, gin.H{"error": "invoice not found"})
		return
	}

	c.Redirect(302, inv.PDFUrl)
}

func (h *InvoiceHandler) GetInvoiceByService(c *gin.Context) {
	serviceID := c.Param("serviceID")
	
	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid serviceID"})
		return
	}

	invoice, err := h.svc.GetInvoiceByService(c.Request.Context(), objID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice)
}

