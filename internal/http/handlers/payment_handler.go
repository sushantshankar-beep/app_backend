package handlers
import (
	"context"
	"fmt"
	"os"
	"time"
	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"app_backend/internal/repository"
	"crypto/sha512"
	"encoding/hex"
)


type PaymentHandler struct {
	paymentSvc *service.PaymentService

	refundRepo        *repository.RefundRepo
	refundWebhookRepo *repository.PaymentRepository // reuse webhook collection
	paymentRepo *repository.PaymentRepository
}


func NewPaymentHandler(
	paymentSvc *service.PaymentService,
	refundRepo *repository.RefundRepo,
	paymentRepo *repository.PaymentRepository,
) *PaymentHandler {
	return &PaymentHandler{
		paymentSvc: paymentSvc,
		refundRepo: refundRepo,
		refundWebhookRepo: paymentRepo,
	}
}



func (h *PaymentHandler) InitiatePayment(c *gin.Context) {
	var req struct {
		ServiceID string  `json:"serviceId"`
		UserID    string  `json:"userId"`
		Name      string  `json:"name"`
		Phone     string  `json:"phone"`
		Price     float64 `json:"price"`
	}

	c.ShouldBindJSON(&req)

	resp, err := h.paymentSvc.InitiatePayment(
		c.Request.Context(),
		req.ServiceID,
		req.UserID,
		req.Name,
		req.Phone,
		req.Price,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, resp)
}
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {

	txnID := c.Param("txnId")
	fmt.Println("this is txn id ",txnID)

	resp, err := h.paymentSvc.VerifyPayment(
		c.Request.Context(),
		txnID,
	)

	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, resp)
}
type PayURefundWebhook struct {
	MihPayID string `json:"mihpayid"`
	Status   string `json:"status"`
	Hash     string `json:"hash"`
	UserID   string `json:"userId"`
}
func sha512Hash(s string) string {
	h := sha512.Sum512([]byte(s))
	return hex.EncodeToString(h[:])
}

func verifyRefundHash(p PayURefundWebhook) bool {

	str := fmt.Sprintf(
		"%s|%s|%s",
		p.MihPayID,
		p.Status,
		os.Getenv("PAYU_SALT"),
	)

	return sha512Hash(str) == p.Hash
}

func (h *PaymentHandler) RefundWebhook(c *gin.Context) {

	var payload map[string]any

	if err := c.BindJSON(&payload); err != nil {
		c.Status(400)
		return
	}

	mih, _ := payload["mihpayid"].(string)
	txn, _ := payload["txnid"].(string)
	status, _ := payload["status"].(string)

	ctx := context.Background()

	// 📦 Save raw webhook payload
	h.paymentRepo.SaveWebhook(ctx, txn, payload)

	// 🔄 Update refund transaction
	_ = h.refundRepo.UpdateByMihPayID(
		ctx,
		mih,
		bson.M{
			"status":    status,
			"updatedAt": time.Now(),
		},
	)

	c.Status(200)
}


