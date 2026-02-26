package handlers

import (
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CouponHandler struct {
	couponSvc *service.CouponService
}

func NewCouponHandler(couponSvc *service.CouponService) *CouponHandler {
	return &CouponHandler{couponSvc: couponSvc}
}

func (h *CouponHandler) GetAvailableCoupons(c *gin.Context) {
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

	search := c.Query("search")

	data, total, err := h.couponSvc.GetAvailableCoupons(
		c.Request.Context(),
		userObjID.Hex(),
		serviceID,
		search,
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

func (h *CouponHandler) GetAutoDiscount(c *gin.Context) {
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

	serviceID := c.Query("serviceId")
	if serviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serviceId required"})
		return
	}

	isNewUser := c.GetBool("isNew")

	result, err := h.couponSvc.ApplyAutoDiscount(
		c.Request.Context(),
		serviceID,
		isNewUser,
		true,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = userObjID

	c.JSON(http.StatusOK, gin.H{
		"originalAmount":      result.ServiceAmount,
		"discountedAmount":    result.DiscountedAmount,
		"totalDiscount":       result.TotalDiscount,
		"appliedDiscount":     result.AppliedDiscount,
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
		ServiceID string `json:"serviceId"`
		Code      string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	isNewUser := c.GetBool("isNew")

    autoResult, _ := h.couponSvc.ApplyAutoDiscount(c, req.ServiceID, isNewUser,false)

	result, err := h.couponSvc.ApplyPromoCode(
		c, userObjID.Hex(), req.Code, req.ServiceID, isNewUser, autoResult,
	)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	gst := result.ServiceAmount * 18 / 100
	c.JSON(200, gin.H{
		"valid":         true,
		"serviceAmount": result.ServiceAmount,
		"discount":      result.TotalDiscount,
		"gst":           gst,
		"totalAmount":   result.ServiceAmount + gst - result.TotalDiscount,
		"appliedPromo": gin.H{
			"code":                result.AppliedPromo.Code,
			"discountAmt":         result.AppliedPromo.DiscountAmt,
			"amountAfterDiscount": result.DiscountedAmount,
		},
	})
}

func (h *CouponHandler) RemoveCoupon(c *gin.Context) {
	userObjIDAny, exists := c.Get(middleware.ContextKeyUserObjectID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	_, ok := userObjIDAny.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id"})
		return
	}

	var req struct {
		ServiceID string `json:"serviceId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	isNewUser := c.GetBool("isNew")

	result, err := h.couponSvc.RemoveCoupon(
		c.Request.Context(),
		req.ServiceID,
		isNewUser,
	)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	gst := result.ServiceAmount * 18 / 100

	c.JSON(200, gin.H{
		"couponRemoved": true,
		"serviceAmount": result.ServiceAmount,
		"discount":      result.TotalDiscount,
		"gst":           gst,
		"totalAmount":   result.ServiceAmount + gst - result.TotalDiscount,
	})
}
