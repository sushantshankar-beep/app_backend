package payment

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{svc: s}
}

func (h *Handler) Initiate(c *gin.Context) {
	var service map[string]any
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	resp, err := h.svc.InitiatePayment(c.Request.Context(), service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Verify(c *gin.Context) {
	var req struct {
		TxnID string `json:"txnid" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "txnid required"})
		return
	}
	resp, err := h.svc.VerifyPayment(c.Request.Context(), req.TxnID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// PayU will POST form (or JSON). We accept both.
func (h *Handler) Webhook(c *gin.Context) {
	// read body into map
	var payload map[string]any
	if err := c.ShouldBind(&payload); err != nil {
		// fallback: try to decode JSON
		if err := json.NewDecoder(c.Request.Body).Decode(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
	}

	// process quickly
	if err := h.svc.ProcessWebhook(c.Request.Context(), payload); err != nil {
		// still return 200 to PayU if you prefer, but I return 400 to indicate signature mismatch
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Return fast success response
	c.JSON(http.StatusOK, gin.H{"success": true})
}
