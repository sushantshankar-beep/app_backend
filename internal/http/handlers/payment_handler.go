package handlers

import (
	"github.com/gin-gonic/gin"
	"app_backend/internal/service"
)

type PaymentHandler struct {
	paymentSvc *service.PaymentService
}

func NewPaymentHandler(paymentSvc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	var req struct {
		ServiceID string  `json:"serviceId"`
		UserID    string  `json:"userId"`
		Name      string  `json:"name"`
		Email     string  `json:"email"`
		Phone     string  `json:"phone"`
		Price     float64 `json:"price"`
	}

	c.ShouldBindJSON(&req)

	resp, err := h.paymentSvc.InitiatePayment(
		c.Request.Context(),
		req.ServiceID,
		req.UserID,
		req.Name,
		req.Email,
		req.Phone,
		req.Price,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, resp)
}
