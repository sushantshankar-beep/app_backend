package handlers

import (
	// "net/http"

	// "app_backend/internal/domain"
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"

	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	// "context"
)

type UserVehicleHandler struct {
	svc *service.UserVehicleService
}

func NewUserVehicleHandler(
	svc *service.UserVehicleService,
) *UserVehicleHandler {
	return &UserVehicleHandler{svc: svc}
}

/*
GET /user/vehicle
*/
func (h *UserVehicleHandler) GetVehicleByNumber(c *gin.Context) {
	number := c.Query("vehicleNumber")
	fmt.Println("this is vehicle number :", number)

	v, exists, err := h.svc.GetVehicleByNumber(
		c.Request.Context(),
		number,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if !exists {
		c.JSON(200, gin.H{"exists": false})
		return
	}

	c.JSON(200, gin.H{
		"exists":  true,
		"vehicle": v,
	})
}

/*
POST /user/vehicle
*/
func (h *UserVehicleHandler) SaveVehicle(c *gin.Context) {
	userObjID := c.MustGet(
		middleware.ContextKeyUserObjectID,
	).(primitive.ObjectID)

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	v, err := h.svc.SaveVehicleForUser(
		c.Request.Context(),
		userObjID,
		req,
	)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"saved":   true,
		"vehicle": v,
	})
}
