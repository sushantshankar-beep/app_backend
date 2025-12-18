package handlers

import "github.com/gin-gonic/gin"

func (h *PaymentHandler) PayUWebhook(c *gin.Context) {
	payload := map[string]string{}
	c.Bind(&payload)

	if err := h.paymentSvc.ProcessWebhook(c.Request.Context(), payload); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}
