package handlers

import (
	"net/http"
	"app_backend/internal/service"
    "strconv"
	"math"
	"github.com/gin-gonic/gin"
	"app_backend/internal/http/middleware"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CouponHandler struct {
	couponSvc *service.CouponService
}

func NewCouponHandler(couponSvc *service.CouponService) *CouponHandler {
	return &CouponHandler{couponSvc: couponSvc}
}

func (h *CouponHandler) GetAvailableCoupons(c *gin.Context) {

	serviceID := c.Query("serviceId")
	if serviceID == "" {
		c.JSON(400, gin.H{"error": "serviceId required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	data, total, err := h.couponSvc.GetAvailableCoupons(
		c.Request.Context(),
		serviceID,
		page,
		limit,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(200, gin.H{
		"data":        data,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	})
}

func (h *CouponHandler) ValidateCoupon(c *gin.Context) {

    userObjIDAny, exists := c.Get(middleware.ContextKeyUserObjectID)
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    userObjID, ok := userObjIDAny.(primitive.ObjectID)
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
        return
    }

    var req struct {
        Code          string `json:"code"`
        ServiceID     string `json:"serviceId"`
    }

    if err := c.ShouldBindJSON(&req); err != nil || req.ServiceID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    isNewUser := c.GetBool("isNew")

    result, err := h.couponSvc.ValidateAndApply(
        c.Request.Context(),
        userObjID.Hex(),
        req.Code,
        req.ServiceID,
        isNewUser,
    )
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "originalAmount":   result.OriginalAmount,
        "discountedAmount": result.DiscountedAmount,
        "totalDiscount":    result.TotalDiscount,
        "appliedPromo":     result.AppliedPromo,
        "appliedDiscount":  result.AppliedDiscount,
    })
}
