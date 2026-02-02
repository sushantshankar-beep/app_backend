package handlers

import (
	"net/http"
	"strconv"

	"app_backend/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	svc *service.NotificationQueryService
}

func NewNotificationHandler(
	svc *service.NotificationQueryService,
) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

/* ============================================================
   USER – LATEST SERVICE
============================================================ */

func (h *NotificationHandler) ListUserLatestService(c *gin.Context) {

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, skip := parsePagination(c)

	serviceID, res, err := h.svc.ListLatestServiceNotifications(
		c.Request.Context(),
		userID,
		"user",
		limit,
		skip,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"serviceId":     serviceID.Hex(),
		"notifications": res,
	})
}

/* ============================================================
   PROVIDER – LATEST SERVICE
============================================================ */

func (h *NotificationHandler) ListProviderLatestService(c *gin.Context) {

	providerID := c.GetString("providerID")
	if providerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, skip := parsePagination(c)

	serviceID, res, err := h.svc.ListLatestServiceNotifications(
		c.Request.Context(),
		providerID,
		"provider",
		limit,
		skip,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"serviceId":     serviceID.Hex(),
		"notifications": res,
	})
}

/* ============================================================
   HELPERS
============================================================ */

func parsePagination(c *gin.Context) (int64, int64) {

	limit, _ := strconv.ParseInt(
		c.DefaultQuery("limit", "20"),
		10,
		64,
	)

	skip, _ := strconv.ParseInt(
		c.DefaultQuery("skip", "0"),
		10,
		64,
	)

	return limit, skip
}
