package handlers

import "github.com/gin-gonic/gin"

func (h *PaymentHandler) Refund(c *gin.Context) {
	var req struct {
		MihPayID string  `json:"mihpayid"`
		Amount   float64 `json:"amount"`
	}
	c.BindJSON(&req)

	if err := h.paymentSvc.Refund(c.Request.Context(), req.MihPayID, req.Amount); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "refund initiated"})
}
