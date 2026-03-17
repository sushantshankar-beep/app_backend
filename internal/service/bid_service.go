
package service
import (
	"context"
	"fmt"
	"time"

	"app_backend/internal/domain"
	"app_backend/internal/repository"
	"app_backend/internal/socket"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"errors"
	// "sync"
	"app_backend/internal/ports"
	"strconv"
	// "strings"
	"encoding/json" 
	"math"
)
type CleanupJob struct {
	ServiceID string
}

type CleanupWorkerPool struct {
	queue chan CleanupJob
}

type BiddingService struct {
	rdb          *redis.Client
	socket       *socket.Emitter
	acceptedRepo *repository.AcceptedServiceRepo
	userRepo     *repository.UserRepo
	bidRepo      *repository.BidRepo
	providerRepo *repository.ProviderRepo
	counterRepo  *repository.CounterRepo
	notify       ports.NotificationService
	paymentRepo *repository.PaymentRepository
	refundRepo   *repository.RefundRepo
	zoneService  *ZoneService
	cleanupPool *CleanupWorkerPool
	
}

func NewBiddingService(
	rdb *redis.Client,
	socket *socket.Emitter,
	acceptedRepo *repository.AcceptedServiceRepo,
	userRepo     *repository.UserRepo,
	bidRepo *repository.BidRepo,
	providerRepo *repository.ProviderRepo,
	counterRepo *repository.CounterRepo,
	notify ports.NotificationService,
	paymentRepo *repository.PaymentRepository,
	refundRepo   *repository.RefundRepo,
	zoneService *ZoneService,
) *BiddingService {
	return &BiddingService{
		rdb:          rdb,
		socket:       socket,
		acceptedRepo: acceptedRepo,
		userRepo: userRepo,
		bidRepo: bidRepo,
		providerRepo: providerRepo,
		counterRepo: counterRepo,
		notify:       notify,
		paymentRepo: paymentRepo,
		refundRepo: refundRepo,
		zoneService:  zoneService,
	}
}
var ErrServiceAlreadyAssigned = errors.New("service already assigned")


func (s *BiddingService) StartCleanupWorkers(workerCount int) {

	s.cleanupPool = &CleanupWorkerPool{
		queue: make(chan CleanupJob, 1000),
	}

	for i := 0; i < workerCount; i++ {
		go s.cleanupWorker(i, s.cleanupPool.queue)
	}
}

/* ==============CANCEL WORKER =================== */

func (s *BiddingService) cleanupWorker(id int, jobs <-chan CleanupJob) {

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for job := range jobs {

		<-ticker.C
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Println("worker panic recovered:", r)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			lockKey := "cleanup:lock:" + job.ServiceID
			ok, _ := s.rdb.SetNX(ctx, lockKey, "1", 5*time.Minute).Result()
			if !ok {
				return
			}
			for i := 0; i < 3; i++ {

				err := s.cleanupServiceKeysCancellation(job.ServiceID)

				if err == nil {
					return
				}

				log.Println("retry cleanup:", job.ServiceID, "attempt:", i+1)

				time.Sleep(time.Duration(i+1) * time.Second)
			}

			log.Println("cleanup failed:", job.ServiceID)

		}()
	}
}
/* ============= QUEUE DECLEARATION ================== */
func (s *BiddingService) enqueueCleanup(serviceID string) {

	if s.cleanupPool == nil {
		log.Println("cleanup pool not initialized")
		return
	}

	select {
	case s.cleanupPool.queue <- CleanupJob{ServiceID: serviceID}:
	default:
		log.Println("entering into default cleanup queue full:", serviceID)
	}
}

/* ================= START SEARCH ================= */

func (s *BiddingService) StartSearch(ctx context.Context,userID domain.UserID,vehicleType string,vehicleNumber string,brand string,modelYear int,fuelType string,serviceType string,issues []string,lat, lng float64,model string) (string, error){
	userOID, _ := primitive.ObjectIDFromHex(string(userID))
	activeStatuses := []domain.ServiceStatus{
		domain.StatusSearching,
		domain.StatusConfirmed,
		domain.StatusStarted,
		domain.StatusReachedLocation,
		domain.StatusOTPVerified,
		domain.StatusInProgress,
		domain.StatusNotStarted,
	}
	// zone, err := s.zoneService.CheckServiceable(ctx, lat, lng)
	// if err != nil {
	// 	return "", err
	// }

	// if zone == nil {
	// 	return "", errors.New("We are  not available in your area")
	// }

	cnt, err := s.acceptedRepo.Count(ctx, bson.M{
		"user": userOID,
		"status": bson.M{
			"$in": activeStatuses,
		},
	})

	if err != nil {
		return "", err
	}

	if cnt > 0 {
		return "", errors.New("you already have an active service")
	}

	seq, _ := s.counterRepo.Next(ctx, "service")
	serviceNumber := fmt.Sprintf("VHBK%05d", seq)
	svc := &domain.AcceptedService{
		User:          userOID,
		Status:        domain.StatusSearching,
		VehicleType:   vehicleType,
		VehicleNumber: vehicleNumber,
		ServiceNumber: serviceNumber,
		Brand:         brand,
		ModelYear:     modelYear,
		FuelType:      fuelType,
		ServiceType:   serviceType,
		Issues:        issues,
		Model:         model,
		UserLocation: &domain.UserLocation{
			Lat:  lat,
			Long: lng,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	now := time.Now()
	svc.Timestamps = &domain.ServiceTimestamps{
		CreatedAt: &now,
	}
	if err := s.acceptedRepo.Create(ctx, svc); err != nil {
		return "", err
	}
	serviceID := svc.ID.Hex()
	// here i have store all details which i have to send in socket in redis
	s.rdb.HSet(ctx, "service:meta:"+svc.ID.Hex(), map[string]any{
		"userId":        userID,
		"vehicleType":   vehicleType,
		"vehicleNumber": vehicleNumber,
		"brand":         brand,
		"modelYear":     modelYear,
		"fuelType":      fuelType,
		"model" :         model,
		"serviceType":   serviceType,
	})
	searchCtx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)

	// 🔥 Start provider discovery
	go func() {
		defer cancel()
		s.findProviders(
			searchCtx,
			serviceID,
			lat,
			lng,
			issues,
			vehicleType,
			vehicleNumber,
			brand,
			modelYear,
			fuelType,
			serviceType,
			model,
		)
	}()

	return serviceID, nil
}

/* ================= FIND PROVIDERS ================= */
const luaFindProviders = `
-- KEYS[1] = providers:geo

-- ARGV:
-- 1 = lng
-- 2 = lat
-- 3 = radiusKm
-- 4 = serviceID
-- 5 = cooldownSec
-- 6 = maxSend
-- 7 = ttlSec

local serviceId = ARGV[4]

-- =====================================
-- 🔴 HARD STOP CHECK (VERY IMPORTANT)
-- =====================================

if redis.call("EXISTS", "service:stop:" .. serviceId) == 1 then
    return {}
end

if redis.call("EXISTS", "service:locked:" .. serviceId) == 1 then
    return {}
end

-- =====================================
-- 📍 GEO SEARCH
-- =====================================

local results = redis.call(
    "GEORADIUS",
    KEYS[1],
    ARGV[1],
    ARGV[2],
    ARGV[3],
    "km",
    "WITHDIST",
    "COUNT",
    300
)

local out = {}

-- =====================================
-- 🔁 FILTER LOOP
-- =====================================

for i = 1, #results do

    local pid  = results[i][1]
    local dist = results[i][2]

    local skip = false

    -- 🚫 Busy check
    if redis.call("EXISTS", "provider:busy:" .. pid) == 1 then
        skip = true
    end

    -- 🚫 Already active in this service
    if not skip then
        local activeKey = "service:activeProvider:" .. serviceId .. ":" .. pid
        if redis.call("EXISTS", activeKey) == 1 then
            skip = true
        end
    end

    -- ❄️ Cooldown logic
    if not skip then
        local cdKey = "service:cooldown:" .. serviceId .. ":" .. pid
        if redis.call("SETNX", cdKey, "1") == 0 then
            skip = true
        else
            redis.call("EXPIRE", cdKey, ARGV[5])
        end
    end

    -- 📊 Send count limit
    if not skip then
        local scKey = "service:sendcount:" .. serviceId .. ":" .. pid
        local cnt = redis.call("INCR", scKey)

        redis.call("EXPIRE", scKey, ARGV[7])

        if cnt > tonumber(ARGV[6]) then
            skip = true
        end
    end

    -- =====================================
    -- ✅ ACCEPT PROVIDER
    -- =====================================

    if not skip then

        -- Cache distance (air distance)
        redis.call(
            "SET",
            "service:dist:" .. serviceId .. ":" .. pid,
            dist,
            "EX",
            1800
        )

        -- Mark active
        local activeKey = "service:activeProvider:" .. serviceId .. ":" .. pid
        redis.call("SET", activeKey, "1", "EX", ARGV[5])

        -- Return provider + distance
        table.insert(out, pid)
        table.insert(out, dist)
    end
end

return out
`
func (s *BiddingService) findProviders(
	ctx context.Context,
	serviceID string,
	lat, lng float64,
	issues []string,
	vehicleType string,
	vehicleNumber string,
	brand string,
	modelYear int,
	fuelType string,
	serviceType string,
	model string,
) {
	stopKey := "service:stop:" + serviceID
	lockKey := "service:locked:" + serviceID

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return
	}

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {
		return
	}

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return
	}
	const (
		cooldownSec    = 35
		maxSendPerProv = 10
		ttlSec         = 7200
		roundDelay     = 5 * time.Second
		globalTimeout  = 10 * time.Minute
		maxRadius      = 100 // km
	)

	startedAt := time.Now()
	type Candidate struct {
		ID       string
		Distance float64
	}

	for {

		// --------------------------------------------------
		// AUTO CANCEL AFTER 30 MINUTES
		// --------------------------------------------------

		if time.Since(startedAt) > globalTimeout {

			log.Println("bidding timeout", serviceID)

			now := time.Now()
			s.rdb.Set(ctx, stopKey, "1", 10*time.Minute)
			_, err := s.acceptedRepo.Col().UpdateByID(
				ctx,
				serviceOID,
				bson.M{
					"$set": bson.M{
						"status": "cancelled",
						"cancelled.by": "system",
						"cancelled.reason": "No provider accepted the service",
						"timestamps.cancelledAt": now,
						"updatedAt": now,
					},
				},
			)

			if err != nil {
				log.Println("cancel update error:", err)
			}

			// websocket emit to user
			s.socket.Emit(
				"user:"+serviceID,
				"service:cancelled",
				map[string]any{
					"serviceId": serviceID,
					"reason":    "No provider available",
				},
			)

			// optional push notification
			go s.notify.SendToUser(
				context.Background(),
				string(user.ID),
				serviceID,
				"Service Cancelled",
				"No provider accepted your request. Please try again.",
				map[string]string{
					"serviceId": serviceID,
				},
			)
			s.socket.Emit("user:"+serviceID, "service:closed", nil)
			s.socket.CloseRoom("user:" + serviceID)

			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		// --------------------------------------------------

		if s.rdb.Exists(ctx, stopKey).Val() == 1 ||
			s.rdb.Exists(ctx, lockKey).Val() == 1 {
			log.Println("🛑 dispatch stopped", serviceID)
			return
		}

		var current domain.AcceptedService
		if err := s.acceptedRepo.Col().
			FindOne(ctx, bson.M{"_id": serviceOID}).
			Decode(&current); err != nil {
			return
		}

		if current.Status != domain.StatusSearching {
			return
		}

		if s.rdb.Exists(ctx, "service:bidWindow:"+serviceID).Val() == 1 {
			time.Sleep(1 * time.Second)
			continue
		}
		res, err := s.rdb.Eval(
			ctx,
			luaFindProviders,
			[]string{"providers:geo"},
			lng,
			lat,
			maxRadius,
			serviceID,
			cooldownSec,
			maxSendPerProv,
			ttlSec,
		).Result()

		if err != nil {
			log.Println("lua geo error:", err)
			time.Sleep(roundDelay)
			continue
		}

		list, ok := res.([]interface{})
		if !ok || len(list) == 0 {
			time.Sleep(roundDelay)
			continue
		}

		candidates := make([]Candidate, 0, len(list)/2)

		for i := 0; i < len(list); i += 2 {

			pid := list[i].(string)
			distStr := list[i+1].(string)

			dist, _ := strconv.ParseFloat(distStr, 64)

			candidates = append(candidates, Candidate{
				ID:       pid,
				Distance: dist,
			})
		}

		// progressive dispatch
		batchSizes := []int{15, 30, 60, 100}
		index := 0
		for _, batch := range batchSizes {

			if s.rdb.Exists(ctx, stopKey).Val() == 1 {
				return
			}

			end := index + batch
			if end > len(candidates) {
				end = len(candidates)
			}

			for _, p := range candidates[index:end] {

				payload := map[string]any{
					"user": map[string]any{
						"name":       user.Name,
						"profileUrl": user.ImageUrl,
						"rating":     user.Rating,
					},
					"serviceId": serviceID,
					"vehicle": map[string]any{
						"type":   vehicleType,
						"number": vehicleNumber,
						"brand":  brand,
						"year":   modelYear,
						"fuel":   fuelType,
						"model":  model,
					},
					"serviceType": serviceType,
					"issues":      issues,
					"distanceKm":  round2(p.Distance),
					"etaMin":      estimateETA(p.Distance),
					"expiresIn":   30,
				}

				s.socket.Emit("provider:"+p.ID, "bid:request", payload)
				go s.notify.SendToProvider(
					context.Background(),
					p.ID,
					serviceID,
					"New Service Request",
					"Nearby user needs help. Open app to bid.",
					map[string]string{
						"serviceId": serviceID,
					},
				)
			}

			index = end

			if index >= len(candidates) {
				break
			}
		}

		time.Sleep(roundDelay)
	}
}



/* ================= PLACE BID ================= */

func (s *BiddingService) PlaceBid(
	ctx context.Context,
	serviceID string,
	providerID string,
	price int,
) (string, error) {
	// 🚫 service cancelled?
	if s.rdb.Exists(ctx, "service:stop:"+serviceID).Val() == 1 {
		return "", errors.New("service is no longer available")
	}

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return "",err
	}
	var svc1 domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc1); err != nil {
		return "",err
	}
	if svc1.Status == domain.StatusConfirmed||
	   svc1.Status == domain.StatusStarted ||
	   svc1.Status == domain.StatusCompleted {

		return "",errors.New("service is no longer available")
	}


	if s.rdb.Exists(ctx, "service:locked:"+serviceID).Val() == 1 {
		return "", errors.New("service is no longer available")
	}
	providerLock := "service:providerBidLock:" + serviceID + ":" + providerID

	ok, err := s.rdb.SetNX(ctx, providerLock, "1", 30*time.Second).Result()
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("provider already placed bid — wait")
	}

	// ===============================
	// 🧊 START GLOBAL BID WINDOW
	// ===============================

	windowKey := "service:bidWindow:" + serviceID

	started, err := s.rdb.SetNX(ctx, windowKey, "1", 30*time.Second).Result()
	if err != nil {
		return "", err
	}

	if started {
		log.Println("bid window started for service:", serviceID)
	}
	providerOID, _ := primitive.ObjectIDFromHex(providerID)

	bid := &domain.BidLog{
		ServiceID:  serviceOID,
		ProviderID: providerOID,
		Price:      price,
		CreatedAt:  time.Now(),
	}
		provider, err := s.providerRepo.FindByID(
		ctx,
		domain.ProviderID(providerID),
	)
	if err != nil {
		return "", err
	}

	bidOID, err := s.bidRepo.Insert(ctx, bid)
	if err != nil {
		return "", err
	}

	distance, _ := s.rdb.Get(
		ctx,
		"service:dist:"+serviceID+":"+providerID,
	).Float64()

	eta := estimateETA(distance)

	// 🔥 SEND BID TO USER
	s.socket.Emit(
		"user:"+serviceID,
		"bid:update",
		map[string]any{
			"bidId": bidOID.Hex(),
			"serviceId":serviceID,
			"price": price,
			"provider": map[string]any{
				"id":         providerID,
				"distanceKm": round2(distance),
				"name":       provider.Name,
				"etaMin":     eta,
				"profileUrl": provider.ProfileURL,
				"rating":     provider.Rating,
			},
		},
	)
	payload := map[string]any{
		"bidId": bidOID.Hex(),
		"serviceId": serviceID,
		"price": price,
		"provider": map[string]any{
			"id":         providerID,
			"distanceKm": round2(distance),
			"name":       provider.Name,
			"etaMin":     eta,
			"profileUrl": provider.ProfileURL,
			"rating":     provider.Rating,
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	svc, _ := s.acceptedRepo.FindByID(ctx, serviceID)
	go s.notify.SendToUser(
		context.Background(),
		svc.User.Hex(),
		serviceID,
		"New Bid Received",
		fmt.Sprintf("%s placed a bid of ₹%d", provider.Name, price),
		map[string]string{
			"type": "NEW_BID",
			"serviceId": serviceID,
			"payload": string(payloadBytes), 
		},
	)


	return bidOID.Hex(), nil
}
func (s *BiddingService) AcceptBid(
	ctx context.Context,
	serviceID string,
	bidID string,
	providerID string,
	price float64,
) (map[string]string, error) {

		serviceKey := "service:locked:" + serviceID
		busyKey := "provider:busy:" + providerID
		stopKey := "service:stop:" + serviceID

		online, err := s.rdb.Get(ctx, "provider:online:"+providerID).Result()
		if err != nil || online != "1" {
			return map[string]string{
				"message": "Mechanic not available",
				"bidId":   bidID,
			}, errors.New("Mechanic not available")
		}
		ok, err := s.rdb.Eval(ctx, `
		-- ARGV[1] = serviceID
		-- ARGV[2] = providerID

		local serviceLock = "service:locked:"..ARGV[1]
		local providerBusy = "provider:busy:"..ARGV[2]

		if redis.call("EXISTS", serviceLock) == 1 then
			return 0
		end

		if redis.call("EXISTS", providerBusy) == 1 then
			return 0
		end

		redis.call("SET", serviceLock, "1", "EX", 1800)
		redis.call("SET", providerBusy, ARGV[1], "EX", 7200)

		return 1
		`, []string{}, serviceID, providerID).Int()

		if err != nil || ok != 1 {
			return map[string]string{
				"message": "provider already assigned in some other service",
				"bidId":   bidID,
			}, ErrServiceAlreadyAssigned
		}
		// stop provider search loops immediately
		s.rdb.Set(ctx, stopKey, "1", 5*time.Minute)

		pos, err := s.rdb.GeoPos(ctx, "providers:geo", providerID).Result()
		if err != nil {
			log.Println("redis geopos failed:", err)
		}
		providerOID, _ := primitive.ObjectIDFromHex(providerID)

		var lat, long float64
		if len(pos) > 0 && pos[0] != nil {
			long = pos[0].Longitude
			lat = pos[0].Latitude
		}
		provider, err := s.providerRepo.FindByID(ctx, domain.ProviderID(providerOID.Hex()))
		if err != nil {
			log.Println("failed to load provider:", err)
		}

		// =====================================================
		// 🗄️ MONGO CONDITIONAL ASSIGN
		// =====================================================

		serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
		bidOID, _ := primitive.ObjectIDFromHex(bidID)

		now := time.Now()

		set := bson.M{
			"provider":      providerOID,
			"acceptedBid":   bidOID,
			"finalPrice":    price,
			"status":        domain.StatusProviderAssigned,
			"paymentStatus": domain.PaymentPending,
			"updatedAt":     now,
		}

		if lat != 0 && long != 0 {
			set["providerLocation"] = bson.M{
				"lat":       lat,
				"long":      long,
				"updatedAt": now,
			}
		}

		res, err := s.acceptedRepo.Col().UpdateOne(
			ctx,
			bson.M{
				"_id":      serviceOID,
				"status":   domain.StatusSearching,
				"provider": primitive.NilObjectID,
			},
			bson.M{"$set": set},
		)

		if err != nil || res.ModifiedCount == 0 {
			s.rdb.Del(ctx, serviceKey, busyKey)
			return map[string]string{
				"message": "provider already assigned in some other service",
				"bidId":   bidID,
			}, ErrServiceAlreadyAssigned
		}

		s.rdb.ZRem(ctx, "providers:geo", providerID)

		// =====================================================
		// 📡 SOCKET EVENTS
		// =====================================================

		s.socket.Emit(
			"provider:"+providerID,
			"bid:accepted",
			map[string]any{
				"serviceId": serviceID,
				"price":     price,
				"profileUrl": provider.ProfileURL,
			},
		)

		s.socket.Emit(
			"user:"+serviceID,
			"service:assigned",
			map[string]any{
				"serviceId":  serviceID,
				"providerId": providerID,
				"price":      price,
			},
		)
		providerPayload := map[string]any{
			"serviceId": serviceID,
			"price":     price,
			"profileUrl": provider.ProfileURL,
		}
		provBytes, _ := json.Marshal(providerPayload)

		go s.notify.SendToProvider(
			context.Background(),
			providerID,
			serviceID,
			"Bid Accepted 🎉",
			"User accepted your bid.",
			map[string]string{
				"serviceId": serviceID,
				"type":      "bid_accepted",
				"payload":   string(provBytes),
			},
		)

		// ✅ SUCCESS
		return map[string]string{
			"message": "",
			"bidId":   bidID,
		}, nil
}



func (s *BiddingService) RejectBid(
	ctx context.Context,
	serviceID string,
	providerID string,
	price int,
) error {

	log.Printf("rejected provider=%s service=%s", providerID, serviceID)
	if s.rdb.Exists(ctx, "provider:busy:"+providerID).Val() == 1 {
		log.Println("⚠ RejectBid skipped — provider busy", providerID)
		return nil
	}
	

	svc, err := s.acceptedRepo.FindByID(ctx, serviceID)
	if err != nil {
		return err
	}
	if svc.Status != domain.StatusSearching {
		log.Println(
			"⚠ RejectBid ignored — service not in SEARCHING:",
			serviceID,
			svc.Status,
		)
		return nil
	}
	windowKey := "service:bidWindow:" + serviceID
	s.rdb.Set(ctx, windowKey, "1", 30*time.Second)

	keys := []string{
		"service:providerBidLock:" + serviceID + ":" + providerID,
		"service:cooldown:" + serviceID + ":" + providerID,
		"service:sendcount:" + serviceID + ":" + providerID,
		"service:activeProvider:" + serviceID + ":" + providerID,
		"service:dist:" + serviceID + ":" + providerID,
	}

	s.rdb.Del(ctx, keys...)

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return err
	}
	dist, _ := s.rdb.Get(
		ctx,
		"service:dist:"+serviceID+":"+providerID,
	).Float64()

	s.socket.Emit(
		"provider:"+providerID,
		"bid:request",
		map[string]any{
			"user": map[string]any{
				"name":      user.Name,
				"profileUrl": user.ImageUrl,
			},
			"serviceId": serviceID,
			"price":     price,
			"vehicle": map[string]any{
				"type":   svc.VehicleType,
				"number": svc.VehicleNumber,
				"brand":  svc.Brand,
				"year":   svc.ModelYear,
				"fuel":   svc.FuelType,
				"model":  svc.Model,
			},
			"serviceType": svc.ServiceType,
			"issues":      svc.Issues,
			"distanceKm":  round2(dist),
			"etaMin":      estimateETA(dist),
			"rebid":       true,
			"expiresIn":   30,
		},
	)

	payload := map[string]any{
		"user": map[string]any{
			"name":      user.Name,
			"profileUrl": user.ImageUrl,
		},
		"serviceId": serviceID,
		"price":     price,
		"vehicle": map[string]any{
			"type":   svc.VehicleType,
			"number": svc.VehicleNumber,
			"brand":  svc.Brand,
			"year":   svc.ModelYear,
			"fuel":   svc.FuelType,
			"model":  svc.Model,
		},
		"serviceType": svc.ServiceType,
		"issues":      svc.Issues,
		"distanceKm":  dist,
		"etaMin":      estimateETA(dist),
		"rebid":       true,
		"expiresIn":   30,
	}

	payloadBytes, _ := json.Marshal(payload)

	go s.notify.SendToProvider(
		context.Background(),
		providerID,
		serviceID,
		"Bid Rejected",
		"Customer rejected your bid. You can offer a new price.",
		map[string]string{
			"type":      "bid_rejected",
			"serviceId": serviceID,
			"payload":   string(payloadBytes),
		},
	)


	return nil
}




func (s *BiddingService) ProviderCancelService(
	ctx context.Context,
	serviceID string,
	providerID string,
	reason string,
) error {

	serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {
		return errors.New("service not found")
	}
	if svc.Status == domain.StatusCancelled{
		return errors.New("service already cancelled")
	}
	now := time.Now()
// here we have taken the snapshot of service because we have to show it provider cancel booking
	snap := svc 
	snap.Status = domain.StatusCancelled
	snap.CancelledAt = &now
	snap.CancelledBy = "provider"

	snap.Cancelled = &domain.CancelInfo{
		By:     "provider",
		Reason: reason,
	}
	snap.UpdatedAt = now

	doc := bson.M{
		"_id":        primitive.NewObjectID(),
		"serviceId":  svc.ID,
		"snapshotAt": now,
		"service":    snap,
	}

	_, err1 := s.acceptedRepo.Col().
		Database().
		Collection("accepted_service_snapshots").
		InsertOne(ctx, doc)

	if err1 != nil {
		log.Println("❌ snapshot insert failed:", err1)
	}

	fixedPrice := svc.FinalPrice
	stopKey := "service:stop:" + serviceID
	lockKey := "service:locked:" + serviceID
	s.rdb.Del(ctx, lockKey)
	s.rdb.Del(ctx, stopKey)
	s.rdb.Del(ctx, "provider:busy:"+providerID)
	s.rdb.Del(ctx, "provider:busy:"+providerID)
	pos, _ := s.rdb.GeoPos(ctx, "providers:geo", providerID).Result()
	if len(pos) == 0 || pos[0] == nil {
		if svc.ProviderLocation != nil && svc.CancelledProviderID == ""{
			s.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      providerID,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}
	}
	_, err := s.acceptedRepo.Col().UpdateByID(
		ctx,
		serviceOID,
		bson.M{
			"$set": bson.M{
				"status":        "searching_after_cancel",
				"timestamps.cancelledAt":now,
				"cancelled.by": "provider",
				"cancelled.reason": reason,
				"provider":      primitive.NilObjectID,
				"acceptedBid":   primitive.NilObjectID,
				"fixedPrice":    fixedPrice,
				"updatedAt":     time.Now(),
				"cancelledByProvider": true,
				"cancelledProviderID":providerID,
			},
		},
	)
	if err != nil {
		return err
	}
	s.socket.Emit(
		"user:"+serviceID,
		"provider:cancelled",
		map[string]any{
			"serviceId": serviceID,
			"price":     fixedPrice,
			"status": "searching_after_cancel",
		},
	)

	go s.notify.SendToUser(
		context.Background(),
		svc.User.Hex(),
		serviceID,
		"Provider cancelled",
		"Searching new provider at same price.",
		map[string]string{
			"serviceId": serviceID,
		},
	)
	s.rdb.Del(ctx,
	    "service:stop:"+serviceID,
	    "service1:locked:"+serviceID,
	    "service:bidWindow:"+serviceID,
	)

	// from here fixed price bidding started
	go s.findProvidersFixedPrice(
		serviceID,
		svc.UserLocation.Lat,
		svc.UserLocation.Long,
		svc.Issues,
		svc.VehicleType,
		svc.VehicleNumber,
		svc.Brand,
		svc.ModelYear,
		svc.FuelType,
		svc.ServiceType,
		svc.Model,
		fixedPrice,
		providerID,
	)

	return nil
}
const luaFindProvidersFixedFast = `
	-- KEYS[1] = providers:geo

	-- ARGV:
	-- 1 = lng
	-- 2 = lat
	-- 3 = radiusKm
	-- 4 = serviceID
	-- 5 = cooldownSec
	-- 6 = maxSend
	-- 7 = ttlSec
	-- 8 = excludeProviderID

	if redis.call("EXISTS", "service:stop:"..ARGV[4]) == 1 then
		return {}
	end

	if redis.call("EXISTS", "service:locked:"..ARGV[4]) == 1 then
		return {}
	end

	local res = redis.call(
		"GEORADIUS",
		KEYS[1],
		ARGV[1],
		ARGV[2],
		ARGV[3],
		"km",
		"WITHDIST",
		"COUNT",
		300
	)

	local out = {}

	for i=1,#res do

		local pid = res[i][1]
		local dist = res[i][2]

		local skip = false

		if pid == ARGV[8] then
			skip = true
		end

		-- busy?
		if not skip and redis.call("EXISTS", "provider:busy:"..pid) == 1 then
			skip = true
		end

		-- already active
		if not skip then
			local activeKey = "service:activeProvider:"..ARGV[4]..":"..pid
			if redis.call("EXISTS", activeKey) == 1 then
				skip = true
			end
		end

		-- cooldown
		if not skip then
			local cdKey = "service:cooldown:"..ARGV[4]..":"..pid
			if redis.call("SETNX", cdKey, "1") == 0 then
				skip = true
			else
				redis.call("EXPIRE", cdKey, ARGV[5])
			end
		end

		-- send count
		if not skip then
			local scKey = "service:sendcount:"..ARGV[4]..":"..pid
			local cnt = redis.call("INCR", scKey)
			redis.call("EXPIRE", scKey, ARGV[7])

			if cnt > tonumber(ARGV[6]) then
				skip = true
			end
		end

		if not skip then

			redis.call(
				"SET",
				"service:dist:"..ARGV[4]..":"..pid,
				dist,
				"EX",
				1800
			)

			local activeKey = "service:activeProvider:"..ARGV[4]..":"..pid
			redis.call("SET", activeKey, "1", "EX", ARGV[5])

			table.insert(out, pid)
			table.insert(out, dist)
		end
	end

	return out
	`
func (s *BiddingService) findProvidersFixedPrice(
	serviceID string,
	lat, lng float64,
	issues []string,
	vehicleType string,
	vehicleNumber string,
	brand string,
	modelYear int,
	fuelType string,
	serviceType string,
	model string,
	fixedPrice float64,
	excludeProviderID string,
) {

	ctx := context.Background()

	stopKey := "service:stop:" + serviceID
	lockKey := "service:locked:" + serviceID

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return
	}
	fixedPriceGst := math.Round(fixedPrice*1.18*100) / 100

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {
		return
	}

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return
	}

	radiusSteps := []float64{25, 50, 100}

	const (
		cooldownSec    = 35
		maxSendPerProv = 10
		ttlSec         = 7200
		roundDelay     = 5 * time.Second
		globalTimeout  = 30 * time.Minute
	)

	for {

		if s.rdb.Exists(ctx, stopKey).Val() == 1 ||
			s.rdb.Exists(ctx, lockKey).Val() == 1 {
			log.Println("fixed-price dispatch stopped", serviceID)
			return
		}

		var current domain.AcceptedService
		if err := s.acceptedRepo.Col().
			FindOne(ctx, bson.M{"_id": serviceOID}).
			Decode(&current); err != nil {
			return
		}

		if current.Status != "searching_after_cancel" {
			return
		}

		for _, radius := range radiusSteps {

			if s.rdb.Exists(ctx, stopKey).Val() == 1 {
				return
			}

			res, err := s.rdb.Eval(
				ctx,
				luaFindProvidersFixedFast,
				[]string{"providers:geo"},
				lng,
				lat,
				radius,
				serviceID,
				cooldownSec,
				maxSendPerProv,
				ttlSec,
				excludeProviderID,
			).Result()

			if err != nil {
				log.Println("lua fixed fast error:", err)
				continue
			}

			list, ok := res.([]interface{})
			if !ok || len(list) == 0 {
				continue
			}

			for i := 0; i < len(list); i += 2 {

				pid := list[i].(string)
				distStr := list[i+1].(string)
				dist, _ := strconv.ParseFloat(distStr, 64)

				go func(providerID string, distance float64, r float64) {

					if s.rdb.Exists(ctx, stopKey).Val() == 1 {
						return
					}

					s.socket.Emit(
						"provider:"+providerID,
						"bid:request:fixed",
						map[string]any{
							"user": map[string]any{
								"name":       user.Name,
								"profileUrl": user.ImageUrl,
							},
							"serviceId":  serviceID,
							"fixedPrice": fixedPriceGst,
							"serviceType": "fixedPrice",
							"vehicle": map[string]any{
								"type":   vehicleType,
								"number": vehicleNumber,
								"brand":  brand,
								"year":   modelYear,
								"fuel":   fuelType,
								"model":  model,
							},
							"issues":     issues,
							"distanceKm": distance,
							"etaMin":     estimateETA(distance),
							"radiusKm":   r,
							"expiresIn":  30,
						},
					)

				}(pid, dist, radius)
			}
		}

		time.Sleep(roundDelay)
	}
}
func (s *BiddingService) CancelService(
	ctx context.Context,
	serviceID string,
	userID string,
	reason string,
) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	now := time.Now()

	lockKey := "service:locked:" + serviceID
	notifiedKey := "service:notified:" + serviceID

	// ===========================
	// 📦 FETCH SERVICE
	// ===========================

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {
		return errors.New("service not found")
	}
	if svc.Status == domain.StatusStarted ||
	   svc.Status == domain.StatusReachedLocation ||
	   svc.Status == domain.StatusOTPVerified {

		return errors.New("service already started, cancellation not allow")
	}

	if svc.User.Hex() != userID {
		return errors.New("not allowed")
	}

	// ===========================
	// 🔒 STOP SEARCH
	// ===========================

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, lockKey, "1", 15*time.Minute)

	cancelledProvider := ""

	if svc.Provider != primitive.NilObjectID {

		cancelledProvider = svc.Provider.Hex()

		pipe.Del(ctx, "provider:busy:"+cancelledProvider)

		if svc.ProviderLocation != nil {

			pipe.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      cancelledProvider,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	_, err = s.acceptedRepo.Col().UpdateByID(
		ctx,
		serviceOID,
		bson.M{
			"$set": bson.M{
				"status":                  domain.StatusCancelled,
				"timestamps.cancelledAt": now,
				"cancelled.by":           "user",
				"cancelled.reason":       reason,
				"cancelledAt":            now,
				"updatedAt":              now,
				"reason":                 reason,
			},
		},
	)
	if err != nil {
		return err
	}

	providers, _ := s.rdb.SMembers(ctx, notifiedKey).Result()

	for _, pid := range providers {

		pid := pid // avoid goroutine capture

		go s.notify.SendToProvider(
			context.Background(),
			pid,
			serviceID,
			"Service Cancelled By User",
			"User cancelled the service",
			map[string]string{
				"serviceId": serviceID,
			},
		)

		s.socket.Emit(
			"provider:"+pid,
			"service:cancelled",
			map[string]any{
				"serviceId": serviceID,
			},
		)

		s.socket.CloseRoom("provider:" + pid)
	}

	s.rdb.Del(ctx, notifiedKey)

	patterns := []string{
		"service:cooldown:" + serviceID + ":*",
		"service:attempts:" + serviceID + ":*",
		"service:userReject:" + serviceID + ":*",
		"service:providerBidLock:" + serviceID + ":*",
	}

	for _, p := range patterns {

		iter := s.rdb.Scan(ctx, 0, p, 500).Iterator()

		var batch []string

		for iter.Next(ctx) {
			batch = append(batch, iter.Val())

			if len(batch) >= 50 {
				s.rdb.Del(ctx, batch...)
				batch = batch[:0]
			}
		}

		if len(batch) > 0 {
			s.rdb.Del(ctx, batch...)
		}
	}

	log.Println("checking refund for service:", serviceID)

	paymentTxn, err := s.paymentRepo.FindByServiceID(ctx, serviceID)

	if err == nil && paymentTxn != nil {

		log.Println(
			"payment txn found",
			"txnID:", paymentTxn.TxnID,
			"status:", paymentTxn.Status,
			"mihpayid:", paymentTxn.MihPayID,
		)

		if paymentTxn.Status == "paid" && paymentTxn.MihPayID != "" {

			refundLock := "refund:lock:" + paymentTxn.MihPayID

			ok, _ := s.rdb.SetNX(ctx, refundLock, "1", 24*time.Hour).Result()

			if ok {

				log.Println("refund queued:", paymentTxn.MihPayID)

				job := domain.RefundJob{
					ServiceID: serviceID,
					MihPayID:  paymentTxn.MihPayID,
					Amount:    paymentTxn.Amount,
					Retries:   0,
				}

				b, _ := json.Marshal(job)

				s.rdb.RPush(ctx, "refund:queue", b)

				s.refundRepo.Create(ctx, &domain.RefundTransaction{
					TxnID:     paymentTxn.TxnID,
					MihPayID:  paymentTxn.MihPayID,
					ServiceID: serviceID,
					UserID:    paymentTxn.UserID,
					Amount:    paymentTxn.Amount,
					Status:    "pending",
					Reason:    reason,
					CreatedAt: now,
					UpdatedAt: now,
				})
			}

		} else {

			log.Println("refund skipped — payment not completed")
		}
	}

	if cancelledProvider != "" {

		s.socket.Emit(
			"provider:"+cancelledProvider,
			"service:cancelled",
			map[string]any{
				"serviceId": serviceID,
				"reason":    reason,
				"by":        "user",
			},
		)

		go s.notify.SendToProvider(
			context.Background(),
			cancelledProvider,
			serviceID,
			"Booking Cancelled By User",
			"User cancelled the service.",
			map[string]string{
				"type":      "SERVICE_CANCELLED_BY_USER",
				"serviceId": serviceID,
				"reason":    reason,
			},
		)

		s.socket.CloseRoom("provider:" + cancelledProvider)
	}

	s.socket.Emit(
		"user:"+serviceID,
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
		},
	)

	go s.notify.SendToUser(
		context.Background(),
		userID,
		serviceID,
		"Booking Cancelled",
		"Your service has been cancelled.",
		map[string]string{
			"type":      "SERVICE_CANCELLED",
			"serviceId": serviceID,
		},
	)

	s.socket.CloseRoom("user:" + serviceID)

	cleanupServiceKeys(ctx, s.rdb, serviceID)

	return nil
}
	func (s *BiddingService) cleanupServiceKeysCancellation(serviceID string) error{

		ctx := context.Background()

		patterns := []string{
			"service:cooldown:" + serviceID + ":*",
			"service:attempts:" + serviceID + ":*",
			"service:userReject:" + serviceID + ":*",
			"service:providerBidLock:" + serviceID + ":*",
			"service:activeProvider:" + serviceID + ":*",
			"service:sendcount:" + serviceID + ":*",
			"service:dist:" + serviceID + ":*",
		}

		for _, pattern := range patterns {

			iter := s.rdb.Scan(ctx, 0, pattern, 1000).Iterator()
			keys := make([]string, 0, 200)

			for iter.Next(ctx) {
				keys = append(keys, iter.Val())

				if len(keys) >= 200 {
					s.rdb.Del(ctx, keys...)
					keys = keys[:0]
				}
			}

			if len(keys) > 0 {
				s.rdb.Del(ctx, keys...)
			}
		}
		return nil
}

func (s *BiddingService) CancelSearchingServiceBeforeBid(
	ctx context.Context,
	serviceID string,
	userID string,
) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	lockKey := "service:locked:" + serviceID
	stopKey := "service:stop:" + serviceID

	if _, err := s.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, lockKey, "1", 15*time.Minute)
		pipe.Set(ctx, stopKey, "1", 30*time.Minute)
		return nil
	}); err != nil {
		return err
	}
	var svc domain.AcceptedService

	err = s.acceptedRepo.Col().
		FindOneAndUpdate(
			ctx,
			bson.M{
				"_id":  serviceOID,
				"user": userOID,
			},
			bson.M{
				"$set": bson.M{
					"status":      domain.StatusCancelled,
					"cancelledBy": "user",
					"cancelledAt": time.Now(),
					"updatedAt":   time.Now(),
				},
			},
		).
		Decode(&svc)

	if err != nil {
		return errors.New("service not found or already cancelled")
	}
	userRoom := "user:" + serviceID
	s.socket.CloseRoom(userRoom)
	if svc.UserLocation != nil {

		results, err := s.rdb.GeoRadius(
			ctx,
			"providers:geo",
			svc.UserLocation.Long,
			svc.UserLocation.Lat,
			&redis.GeoRadiusQuery{
				Radius: 100,
				Unit:   "km",
			},
		).Result()

		if err != nil {
			log.Println("geo cancel error:", err)
		} else {

			seen := make(map[string]struct{}, len(results))

			for _, loc := range results {

				providerID := loc.Name

				if _, exists := seen[providerID]; exists {
					continue
				}
				seen[providerID] = struct{}{}

				s.socket.Emit(
					"provider:"+providerID,
					"service:cancelled",
					map[string]any{
						"serviceId": serviceID,
						"reason":    "cancelled_by_user",
					},
				)
			}
		}
	}
	s.enqueueCleanup(serviceID)

	return nil
}

func (s *BiddingService) CancelSearchingService(
	ctx context.Context,
	serviceID string,
	userID string,
) error {
	fmt.Println(serviceID,userID)

	return s.CancelSearchingServiceBeforeBid(ctx, serviceID, userID)
}

func cleanupServiceKeys(ctx context.Context, rdb *redis.Client, serviceID string) {

	patterns := []string{
		"service:cooldown:" + serviceID + ":*",
		"service:activeProvider:" + serviceID + ":*",
		"service:sendcount:" + serviceID + ":*",
		"service:dist:" + serviceID + ":*",
		"service:providerBidLock:" + serviceID + ":*",
		"service:stop:" + serviceID,
		"service:locked:" + serviceID,
	}

	for _, pattern := range patterns {
		iter := rdb.Scan(ctx, 0, pattern, 500).Iterator()
		for iter.Next(ctx) {
			rdb.Del(ctx, iter.Val())
		}
	}
}
func (s *BiddingService) HandleProviderTimeout(
	ctx context.Context,
	svc *domain.AcceptedService,
) {

		serviceID := svc.ID.Hex()
		providerID := svc.Provider.Hex()
		if svc.PaymentStatus == "paid" {
			log.Println("timeout ignored payment done", serviceID)
			return
		}
		lockKey := "timeout:lock:" + serviceID

		ok, _ := s.rdb.SetNX(ctx, lockKey, "1", 2*time.Minute).Result()
		if !ok {
			return
		}

		log.Println("releasing timed-out provider", providerID, "service", serviceID)
		keys := []string{
			"service:locked:" + serviceID,
			"service:stop:" + serviceID,
			"provider:busy:" + providerID,
		}

		s.rdb.Del(ctx, keys...)

		cleanupServiceKeys(ctx, s.rdb, serviceID)

		if svc.ProviderLocation != nil {

			s.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      providerID,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}
		_ = s.acceptedRepo.UpdateByID(
			ctx,
			svc.ID,
			bson.M{
				"$set": bson.M{
					"status": domain.StatusCancelled,
					"provider": primitive.NilObjectID,
					"acceptedBid": primitive.NilObjectID,
					"updatedAt": time.Now(),
				},
			},
		)
		s.socket.Emit(
			"user:"+serviceID,
			"provider:timeout",
			map[string]any{
				"serviceId": serviceID,
			},
		)
	}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func estimateETA(distanceKm float64) int64 {
	const avgSpeed = 30.0
	return int64((distanceKm / avgSpeed) * 60)
}