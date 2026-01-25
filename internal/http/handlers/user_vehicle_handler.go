package handlers

import (
	"app_backend/internal/http/middleware"
	"app_backend/internal/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	number := c.Param("vehicleNumber")

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

/* Get /user/vehicleData */
func (h *UserVehicleHandler) GetVehicleData(c *gin.Context) {
	vehicleType := c.Query("vehicleType")
	make := c.Query("make")
	model := c.Query("model")

	data, err := h.svc.GetVehicleData(c.Request.Context(), vehicleType, make, model)

	if err != nil {
		c.JSON(500, gin.H{"error": "failed to fetch vehicles"})
		return
	}

	c.JSON(200, gin.H{
		"data":  data,
		"count": len(data),
	})
}
