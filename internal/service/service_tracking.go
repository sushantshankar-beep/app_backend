package service

import (
	"context"
	"errors"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/socket"
	"app_backend/internal/utils"

	"app_backend/internal/ports"
	"fmt"
	"math"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ServiceTrackingService struct {
	acceptedRepo *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	providerRepo *repository.ProviderRepo
	socket       *socket.Emitter
	notify        ports.NotificationService
	rdb          *redis.Client
	complaintRepo *repository.ComplaintRepo
	invoiceRepo  *repository.InvoiceRepo
}

func NewServiceTrackingService(
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo *repository.UserRepo,
	providerRepo *repository.ProviderRepo,
	socket *socket.Emitter,
	notify ports.NotificationService,
	rdb *redis.Client,
	complaintRepo *repository.ComplaintRepo,
	invoiceRepo  *repository.InvoiceRepo,
) *ServiceTrackingService {

	return &ServiceTrackingService{
		acceptedRepo:  acceptedRepo,
		userRepo:      userRepo,
		providerRepo:  providerRepo,
		socket:        socket,
		notify:        notify,
		rdb:           rdb,
		complaintRepo: complaintRepo,
		invoiceRepo: invoiceRepo,
	}
}


/* ============================================================
	                    TRACKING SCREENS
============================================================ */
func distanceKmHaversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371

	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func estimateETASeconds(distanceKm float64) int64 {
	const avgSpeed = 30.0
	return int64((distanceKm / avgSpeed) * 3600)
}

func (s *ServiceTrackingService) UserTrackingScreen(ctx context.Context,serviceID string) (map[string]any, error){

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	user, _ := s.userRepo.GetByID(ctx, svc.User)
	provider, _ := s.providerRepo.FindByID(ctx, domain.ProviderID(svc.Provider.Hex()))

	// ----------------------------------
	// 🧮 Distance & ETA
	// ----------------------------------

	var distanceKm float64
	var etaMinutes int64

	if svc.ProviderLocation.Lat != 0 &&
		svc.ProviderLocation.Long != 0 &&
		svc.UserLocation.Lat != 0 &&
		svc.UserLocation.Long != 0 {

		distanceKm = distanceKmHaversine(
			svc.ProviderLocation.Lat,
			svc.ProviderLocation.Long,
			svc.UserLocation.Lat,
			svc.UserLocation.Long,
		)

		etaMinutes = estimateETASeconds(distanceKm) / 60
	}

	// ----------------------------------
	// 💰 Billing
	// ----------------------------------

	const gstPercent = 18.0
	var serviceCharge, totalDiscount, amountAfterDiscount, gstAmount, totalAmount float64

	serviceCharge = svc.FinalPrice

	if svc.TotalDiscount > 0 || svc.AppliedPromo != nil || svc.AppliedDiscount != nil {
		totalDiscount = svc.TotalDiscount
		gstAmount = (serviceCharge * gstPercent) / 100
		amountAfterDiscount = serviceCharge - totalDiscount
		totalAmount = serviceCharge + gstAmount - totalDiscount
	} else {
		gstAmount = (serviceCharge * gstPercent) / 100
		totalAmount = serviceCharge + gstAmount
	}

	var appliedPromo any
	if svc.AppliedPromo != nil {
		appliedPromo = map[string]any{
			"promoId":     svc.AppliedPromo.PromoID,
			"code":        svc.AppliedPromo.Code,
			"discountAmt": svc.AppliedPromo.DiscountAmt,
		}
	}

	var appliedDiscount any
	if svc.AppliedDiscount != nil {
		appliedDiscount = map[string]any{
			"discountId":  svc.AppliedDiscount.DiscountID,
			"code":        svc.AppliedDiscount.Code,
			"discountAmt": svc.AppliedDiscount.DiscountAmt,
		}
	}

	// ----------------------------------
	// 📄 Complaint (if any)
	// ----------------------------------

	var complaintId any
	if svc.ComplaintUser != nil {
		complaintId = svc.ComplaintUser.Hex()
	}

	// ----------------------------------
	// 📦 Response
	// ----------------------------------

	return map[string]any{
		"screen": "SERVICE_TRACKING",
		"status": svc.Status,
		"otp":    user.ServiceOTP,

		"provider": map[string]any{
			"id":         provider.ID,
			"name":       provider.Name,
			"rating":     provider.Rating,
			"etaMinutes": etaMinutes,
			"distanceKm": distanceKm,
			"phoneNo":    provider.Phone,
			"profileUrl": provider.ProfileURL,
		},

		"booking": map[string]any{
			"bookingId": svc.ServiceNumber,
			"status":    "BID_ACCEPTED",
			"serviceId": serviceID,
		},

		"vehicle": map[string]any{
			"problem":       svc.ServiceType,
			"date": time.Now().Format("2006-01-02 03:04 PM"),
			"vehicleNumber": svc.VehicleNumber,
			"brand":         svc.Brand,
			"fuelType":      svc.FuelType,
			"year":          svc.ModelYear,
			"model":         svc.Model,
		},

		"locations": map[string]any{
			"user": map[string]any{
				"lat":  svc.UserLocation.Lat,
				"long": svc.UserLocation.Long,
			},
			"provider": map[string]any{
				"lat":  svc.ProviderLocation.Lat,
				"long": svc.ProviderLocation.Long,
			},
		},

		"billing": map[string]any{
			"serviceAmount": utils.RoundTo2(serviceCharge),
			"totalDiscount":       utils.RoundTo2(totalDiscount),
			"amountAfterDiscount": utils.RoundTo2(amountAfterDiscount),
			"gst":            utils.RoundTo2(gstAmount),
			"totalAmount":   utils.RoundTo2(totalAmount),
			"currency":      "INR",
			"appliedPromo":        appliedPromo,
			"appliedDiscount":     appliedDiscount,
		},

		"complaintUserId": complaintId,

		"timestamps": svc.Timestamps,
	}, nil
}



func (s *ServiceTrackingService) ProviderTrackingScreen(ctx context.Context,serviceID string) (map[string]any, error){

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	user, _ := s.userRepo.GetByID(ctx, svc.User)

	// ----------------------------------
	// 📄 Complaint (if exists)
	// ----------------------------------

	var complaintId any
	if svc.ComplaintProvider != nil {
		complaintId = svc.ComplaintProvider.Hex()
	}

	return map[string]any{
		"screen": "PROVIDER_TRACKING",

		"user": map[string]any{
			"name":  user.Name,
			"phone": user.Phone,
			"otp":   user.ServiceOTP,
			"profileUrl": user.ImageUrl,
		},

		"service": svc.ServiceType,
		"status":  svc.Status,
		"timestamps": svc.Timestamps,
		"complaintProviderId": complaintId,
	}, nil
}


func (s *ServiceTrackingService) sendStatusNotification(
	svc *domain.AcceptedService,
	newStatus domain.ServiceStatus,
) {

	title := ""
	message := ""

	// ===============================
	// LOAD PROVIDER NAME
	// ===============================

	providerName := "Mechanic"

	if svc.Provider != primitive.NilObjectID {
		provider, err := s.providerRepo.FindByID(
			context.Background(),
			domain.ProviderID(svc.Provider.Hex()),
		)
		if err == nil && provider != nil {
			providerName = provider.Name
		}
	}

	// ===============================
	// STATUS SWITCH
	// ===============================

	switch newStatus {

	case domain.StatusStarted:
		title = fmt.Sprintf("%s Is On The Way", providerName)
		message = fmt.Sprintf("%s has started and is on the way.", providerName)

	case domain.StatusReachedLocation:
		title = "Mechanic Reached"
		message = "Mechanic has reached your location."
		message = fmt.Sprintf("%s has reached your location.", providerName)

	case domain.StatusOTPVerified:
		title = "OTP Verified"
		message = "Service verification completed."

	case domain.StatusInProgress:
		title = "Service Started"
		message = "Your service is currently in progress."

	case domain.StatusCompleted:
		title = "Service Completed"
		message = "Your service has been successfully completed."

	default:
		title = "Service Update"
		message = "Your service status has been updated."
	}

	// ============================================
	// 🔔 ALWAYS SEND TO USER
	// ============================================

	go s.notify.SendToUser(
		context.Background(),
		svc.User.Hex(),
		svc.ID.Hex(),
		title,
		message,
		map[string]string{
			"type":      "SERVICE_STATUS_UPDATE",
			"serviceId": svc.ID.Hex(),
			"status":    string(newStatus),
		},
	)

	// ============================================
	// 🔔 SEND TO PROVIDER ONLY FOR STARTED + COMPLETED
	// ============================================

	if svc.Provider != primitive.NilObjectID &&
		(newStatus == domain.StatusStarted ||
			newStatus == domain.StatusCompleted) {

		go s.notify.SendToProvider(
			context.Background(),
			svc.Provider.Hex(),
			svc.ID.Hex(),
			title,
			message,
			map[string]string{
				"type":      "SERVICE_STATUS_UPDATE",
				"serviceId": svc.ID.Hex(),
				"status":    string(newStatus),
			},
		)
	}
}



func (s *ServiceTrackingService) UpdateStatus(ctx context.Context,serviceID string,newStatus domain.ServiceStatus,lat float64,long float64) (map[string]any, error) {
	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}
	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().FindOne(ctx, bson.M{"_id": objID}).Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}
	prevStatus := svc.Status
	if prevStatus == "cancelled"{
		return nil,errors.New("Service already Cancelled")
	}

	if !domain.CanTransition(prevStatus, newStatus) {
		return nil, errors.New("invalid service state transition")
	}

	now := time.Now()

	update := bson.M{
		"status":    newStatus,
		"updatedAt": now,
	}

	if lat != 0 && long != 0 {
		update["providerLocation"] = bson.M{
			"lat":       lat,
			"long":      long,
			"updatedAt": now,
		}
	}

	switch newStatus {
	case domain.StatusStarted:
		update["timestamps.startedAt"] = now

	case domain.StatusReachedLocation:
		update["timestamps.reachedAt"] = now

	case domain.StatusOTPVerified:
		update["timestamps.OtpVerified"] = now

	case domain.StatusInProgress:
		update["timestamps.inProgressAt"] = now

	case domain.StatusCompleted:
		update["timestamps.completedAt"] = now
		serviceHex := svc.ID.Hex()
		providerHex := svc.Provider.Hex()
		busyKey := "provider:busy:" + providerHex

		val, _ := s.rdb.Get(ctx, busyKey).Result()
		if val == serviceHex {
			s.rdb.Del(ctx, busyKey)
		}

		// unlock service
		s.rdb.Del(ctx, "service:locked:"+serviceHex)
		s.rdb.Del(ctx, "service:stop:"+serviceHex)

		// provider re added
		if svc.ProviderLocation != nil {
			s.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      providerHex,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}
		_, _ = s.acceptedRepo.Col().UpdateByID(
			ctx,
			objID,
			bson.M{
				"$set": bson.M{
					"archived": true,
					"closedAt": now,
				},
			},
		)
		gst := svc.FinalPrice * 18 / 100
		totalAmount := svc.FinalPrice + gst
		user, err := s.userRepo.GetByID(ctx, svc.User)
		if err != nil {
			return nil, err
		}
		userObjID, err := primitive.ObjectIDFromHex(string(user.ID))
		if err != nil {
			return nil, err
		}
		newTotalExpense := user.TotalExpense + totalAmount
		if err := s.userRepo.UpdateTotalExpense(ctx,userObjID,newTotalExpense); err != nil {
			return nil, err
		}
	}
	if _, err := s.acceptedRepo.Col().
		UpdateByID(ctx, objID, bson.M{"$set": update}); err != nil {
		return nil, err
	}
	if err := s.acceptedRepo.Col().FindOne(ctx, bson.M{"_id": objID}).Decode(&svc); err != nil {
		return nil, errors.New("failed to reload service after update")
	}
	var complaintId any = nil
	if svc.ComplaintProvider != nil && *svc.ComplaintProvider != primitive.NilObjectID {
		complaintId = svc.ComplaintProvider.Hex()
	}
	user, _ := s.userRepo.GetByID(ctx, svc.User)
	provider, _ := s.providerRepo.FindByID(ctx, domain.ProviderID(svc.Provider.Hex()))
	userLat := svc.UserLocation.Lat
	userLong := svc.UserLocation.Long
	var distanceKm float64
	var eta int64
	distKey := "service:dist:" + svc.ID.Hex() + ":" + svc.Provider.Hex()
	if d, err := s.rdb.Get(ctx, distKey).Float64(); err == nil && d > 0 {
		distanceKm = d
	} else if lat != 0 && long != 0 {
		// fallback for distance
		distanceKm = distanceKmHaversine(lat,long,userLat,userLong)
	}
	eta = estimateETA(distanceKm)
	distanceKmRound := math.Round(distanceKm*100) / 100
	gst := svc.FinalPrice * 18 / 100
	totalAmount := gst + svc.FinalPrice

	payload := map[string]any{
		"serviceId":     svc.ServiceNumber,
		"serviceIdMain": svc.ID.Hex(),
		"oldStatus":     prevStatus,
		"newStatus":     newStatus,
		"date":          svc.CreatedAt.Format("2006-01-02"),
		"complaintProviderId": complaintId,
		"user": map[string]any{
			"id":    svc.User.Hex(),
			"name":  user.Name,
			"phone": user.Phone,
			"lat":   userLat,
			"lon":   userLong,
			"profileUrl": user.ImageUrl,
		},
		"provider": map[string]any{
			"id":     provider.ID,
			"name":   provider.Name,
			"rating": provider.Rating,
			"phone":  provider.Phone,
			"profileUrl" : provider.ProfileURL,
		},
		"vehicle": map[string]any{
			"fuelType":      svc.FuelType,
			"vehicleType":   svc.VehicleType,
			"vehicleNumber": svc.VehicleNumber,
			"brand":         svc.Brand,
			"modelYear":     svc.ModelYear,
			"model":         svc.Model,
		},
		"issues": svc.Issues,

		"billing": map[string]any{
			"serviceAmount": svc.FinalPrice,
			"gst":           gst,
			"totalAmount":   totalAmount,
			"currency":      "INR",
		},

		"timestamps": svc.Timestamps,
	}

	payload["distanceKm"] = distanceKmRound
	payload["eta"] = eta

	payload["providerLocation"] = map[string]any{
		"lat":  lat,
		"long": long,
	}
	userRoom := "user:" + svc.ID.Hex()
	providerRoom := "provider:" + svc.Provider.Hex()
	socketPayload := map[string]any{"serviceId": svc.ID.Hex(),"status":  newStatus}
	s.socket.EmitWithRetry(userRoom, "service:status_update", socketPayload, 1)
	s.socket.EmitWithRetry(providerRoom, "service:status_update", socketPayload, 1)
	s.sendStatusNotification(&svc, newStatus)

	if newStatus == domain.StatusCompleted {
		s.socket.Emit(userRoom, "service:closed", nil)
		s.socket.Emit(providerRoom, "service:closed", nil)
		go func() {
			time.Sleep(400 * time.Millisecond)
			s.socket.CloseRoom(userRoom)
			s.socket.CloseRoom(providerRoom)
		}()
	}

	return payload, nil
}



func (s *ServiceTrackingService) VerifyOTP(ctx context.Context,serviceID string,inputOTP string,lat float64,long float64) (map[string]any, error) {
	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, err
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().FindOne(ctx, bson.M{"_id": objID}).Decode(&svc); err != nil {
		return nil, errors.New("service not found")
	}

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return nil, err
	}

	if user.ServiceOTP != inputOTP {
		return nil, errors.New("invalid otp")
	}

	now := time.Now()
	if _, err := s.acceptedRepo.Col().UpdateByID(
		ctx,
		objID,
		bson.M{
			"$set": bson.M{
				"otp.verified":   true,
				"otp.verifiedAt": now,
			},
		},
	); err != nil {
		return nil, err
	}
	return s.UpdateStatus(ctx,serviceID,domain.StatusOTPVerified,lat,long)
}

func (s *ServiceTrackingService) GetInvoiceUrl(ctx context.Context,serviceID string) (*domain.Invoice, error) {

	if serviceID == "" {
		return nil, errors.New("serviceId is required")
	}

	invoice, err := s.invoiceRepo.FindByServiceID(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	if invoice.PDFUrl == "" {
		return nil, errors.New("invoice pdf not generated yet")
	}

	return invoice, nil
}



func (s *ServiceTrackingService) UpdateLiveLocation(
	ctx context.Context,
	serviceID string,
	lat float64,
	long float64,
) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	err = s.acceptedRepo.UpdateByID(ctx, serviceOID, bson.M{
		"$set": bson.M{
			"providerLocation": bson.M{
				"lat":  lat,
				"long": long,
			},
			"updatedAt": time.Now(),
		},
	})
	if err != nil {
		return err
	}
	s.socket.Emit(
		"user:"+serviceID,
		"service:status_update",
		map[string]interface{}{
			"serviceId": serviceID,
			"lat":       lat,
			"long":      long,
		},
	)
	return nil
}
