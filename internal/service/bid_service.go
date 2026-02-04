
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
	}
}
var ErrServiceAlreadyAssigned = errors.New("service already assigned")

/* ================= START SEARCH ================= */

func (s *BiddingService) StartSearch(ctx context.Context,userID domain.UserID,vehicleType string,vehicleNumber string,brand string,modelYear int,fuelType string,serviceType string,issues []string,lat, lng float64,model string) (string, error){
	userOID, _ := primitive.ObjectIDFromHex(string(userID))

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
	fmt.Println("AFTER INSERT svc.ID =", svc.ID.Hex())

	// 🔥 Cache service meta (single source for sockets)
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


	// 🔥 Start provider discovery
	go s.findProviders(
		svc.ID.Hex(),
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

	return svc.ID.Hex(), nil
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

for i=1,#results do
  local pid = results[i][1]
  local dist = results[i][2]

  local skip = false

  -- skip busy
  if redis.call("EXISTS", "provider:busy:"..pid) == 1 then
    skip = true
  end

  -- skip already active
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
    -- cache distance
    redis.call(
      "SET",
      "service:dist:"..ARGV[4]..":"..pid,
      dist,
      "EX",
      1800
    )

    -- mark active
    local activeKey = "service:activeProvider:"..ARGV[4]..":"..pid
    redis.call("SET", activeKey, "1", "EX", ARGV[5])

    table.insert(out, pid)
    table.insert(out, dist)
  end
end

return out
`
func (s *BiddingService) findProviders(
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

	ctx := context.Background()

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

	radiusSteps := []float64{10, 25, 50, 100}

	const (
		cooldownSec    = 77
		maxSendPerProv = 8
		ttlSec         = 7200
		roundDelay     = 5 * time.Second
		globalTimeout  = 30 * time.Minute
	)

	startedAt := time.Now()

	for {

		if time.Since(startedAt) > globalTimeout {
			log.Println("⏱ bidding timeout", serviceID)
			return
		}

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
		s.rdb.Del(ctx, "service:bidWindow:"+serviceID)

		for _, radius := range radiusSteps {

			if s.rdb.Exists(ctx, stopKey).Val() == 1 {
				return
			}

			res, err := s.rdb.Eval(
				ctx,
				luaFindProviders,
				[]string{"providers:geo"},
				lng,
				lat,
				radius,
				serviceID,
				cooldownSec,
				maxSendPerProv,
				ttlSec,
			).Result()

			if err != nil {
				log.Println("lua geo error:", err)
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

					go s.notify.SendToProvider(
						context.Background(),
						providerID,
						serviceID,
						"New Service Request",
						"Nearby user needs help. Open app to bid.",
						map[string]string{
							"serviceId": serviceID,
						},
					)

					s.socket.Emit(
						"provider:"+providerID,
						"bid:request",
						map[string]any{
							"user": map[string]any{
								"name": user.Name,
								"profileUrl":user.ImageUrl,
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
							"distanceKm":  distance,
							"etaMin":      estimateETA(distance),
							"radiusKm":    r,
							"expiresIn":   60,
						},
					)

				}(pid, dist, radius)
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

	if s.rdb.Exists(ctx, "service:locked:"+serviceID).Val() == 1 {
		return "", errors.New("service is no longer available")
	}
	providerLock := "service:providerBidLock:" + serviceID + ":" + providerID

	ok, err := s.rdb.SetNX(ctx, providerLock, "1", 60*time.Second).Result()
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

	started, err := s.rdb.SetNX(ctx, windowKey, "1", 60*time.Second).Result()
	if err != nil {
		return "", err
	}

	if started {
		log.Println("⏸ bid window started for service:", serviceID)
	}
	serviceOID, _ := primitive.ObjectIDFromHex(serviceID)
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
				"distanceKm": distance,
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
			"distanceKm": distance,
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

		// =====================================================
		// 🔐 REDIS ATOMIC LOCK (SERVICE + PROVIDER)
		// =====================================================

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

		// =====================================================
		// 📍 FETCH PROVIDER GEO
		// =====================================================

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

			// 🔄 rollback redis locks
			s.rdb.Del(ctx, serviceKey, busyKey)

			return map[string]string{
				"message": "provider already assigned in some other service",
				"bidId":   bidID,
			}, ErrServiceAlreadyAssigned
		}

		// =====================================================
		// 🚫 REMOVE PROVIDER FROM GEO
		// =====================================================

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


		// =====================================================
		// 🔔 PUSH NOTIFICATION
		// =====================================================

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

	log.Printf("👎 rejected provider=%s service=%s", providerID, serviceID)

	// =====================================================
	// 🔍 LOAD SERVICE FIRST (CRITICAL)
	// =====================================================
	if s.rdb.Exists(ctx, "provider:busy:"+providerID).Val() == 1 {
		log.Println("⚠ RejectBid skipped — provider busy", providerID)
		return nil
	}
	

	svc, err := s.acceptedRepo.FindByID(ctx, serviceID)
	if err != nil {
		return err
	}

	// ❗ If service is no longer searching — DO NOTHING
	if svc.Status != domain.StatusSearching {
		log.Println(
			"⚠ RejectBid ignored — service not in SEARCHING:",
			serviceID,
			svc.Status,
		)
		return nil
	}

	// =====================================================
	// 🧊 Re-open bid window ONLY if service unlocked
	// =====================================================

	if s.rdb.Exists(ctx, "service:locked:"+serviceID).Val() == 0 {
		s.rdb.Del(ctx, "service:bidWindow:"+serviceID)
	}

	// =====================================================
	// 🧹 CLEAN PROVIDER-SCOPED REDIS KEYS
	// =====================================================

	keys := []string{
		"service:providerBidLock:" + serviceID + ":" + providerID,
		"service:cooldown:" + serviceID + ":" + providerID,
		"service:sendcount:" + serviceID + ":" + providerID,
		"service:activeProvider:" + serviceID + ":" + providerID,
		"service:dist:" + serviceID + ":" + providerID,
	}

	s.rdb.Del(ctx, keys...)

	// =====================================================
	// 👤 LOAD USER
	// =====================================================

	user, err := s.userRepo.GetByID(ctx, svc.User)
	if err != nil {
		return err
	}

	// =====================================================
	// 📍 DISTANCE
	// =====================================================

	dist, _ := s.rdb.Get(
		ctx,
		"service:dist:"+serviceID+":"+providerID,
	).Float64()

	// =====================================================
	// 📡 SOCKET → PROVIDER REBID
	// =====================================================

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
			"distanceKm":  dist,
			"etaMin":      estimateETA(dist),
			"rebid":       true,
			"expiresIn":   60,
		},
	)

	// =====================================================
	// 🔔 PUSH NOTIFICATION
	// =====================================================

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
		"expiresIn":   60,
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
			"payload":   string(payloadBytes), // ✅ SAME AS SOCKET
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

	// fetch service
	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {
		return errors.New("service not found")
	}

	// only assigned provider can cancel
	if svc.Provider.Hex() != providerID {
		return errors.New("not assigned provider")
	}
	fixedPrice := svc.FinalPrice

	// 🔓 unlock service
	stopKey := "service:stop:" + serviceID
	lockKey := "service:locked:" + serviceID
	s.rdb.Del(ctx, lockKey)
	s.rdb.Del(ctx, stopKey)
	s.rdb.Del(ctx, "provider:busy:"+providerID)

	// 🧹 clear busy provider
	s.rdb.Del(ctx, "provider:busy:"+providerID)

	// 🗑️ put provider back to geo (optional)
	pos, _ := s.rdb.GeoPos(ctx, "providers:geo", providerID).Result()
	if len(pos) == 0 || pos[0] == nil {
		if svc.ProviderLocation != nil {
			s.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      providerID,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}
	}
	now := time.Now()

	// 🔄 reset service state
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

	// 🔔 USER
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

	// 🚀 restart bidding with fixed price
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
const luaFindProvidersFixed = `
-- KEYS[1] = providers:geo

-- ARGV:
-- 1 = lng
-- 2 = lat
-- 3 = radiusKm
-- 4 = serviceID
-- 5 = cooldownSecs
-- 6 = excludeProviderID

if redis.call("EXISTS", "service:stop:"..ARGV[4]) == 1 then
	return {}
end

if redis.call("EXISTS", "service:locked:"..ARGV[4]) == 1 then
	return {}
end

local providersKey = "service:providers:"..ARGV[4]

local res = redis.call(
	"GEORADIUS",
	KEYS[1],
	ARGV[1],
	ARGV[2],
	ARGV[3],
	"km",
	"WITHDIST",
	"COUNT",
	80
)

local out = {}

for i=1,#res do

	local pid = res[i][1]
	local dist = res[i][2]

	if pid ~= ARGV[6] then

		if redis.call("EXISTS", "provider:busy:"..pid) == 0 then

			local activeKey = "service:activeProvider:"..ARGV[4]..":"..pid
			if redis.call("SETNX", activeKey, "1") == 1 then

				redis.call("EXPIRE", activeKey, ARGV[5])

				local cdKey = "service:cooldown:"..ARGV[4]..":"..pid
				if redis.call("SETNX", cdKey, "1") == 1 then

					redis.call("EXPIRE", cdKey, ARGV[5])

					redis.call("SADD", providersKey, pid)
					redis.call("EXPIRE", providersKey, 1800)

					table.insert(out, pid)
					table.insert(out, dist)

				end
			end
		end
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

	// ✅ GST INCLUDED
	fixedPriceGst := math.Round(fixedPrice*1.18*100) / 100

	// ===============================
	// LOAD SERVICE + USER
	// ===============================

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
		roundDelay   = 70 * time.Second
		maxRounds    = 10
		cooldownSecs = 90
	)

	round := 0

	for round < maxRounds {

		round++

		// 🛑 redis kill-switch
		if s.rdb.Exists(ctx, stopKey).Val() == 1 ||
			s.rdb.Exists(ctx, lockKey).Val() == 1 {
			log.Println("🛑 fixed price stopped:", serviceID)
			return
		}
		var current domain.AcceptedService
		if err := s.acceptedRepo.Col().
			FindOne(ctx, bson.M{"_id": serviceOID}).
			Decode(&current); err != nil {
			return
		}

		if current.Status != "searching_after_cancel"{

			log.Println("🛑 fixed-price discovery exit due to status:", current.Status)
			return
		}

		for _, radius := range radiusSteps {
			if s.rdb.Exists(ctx, stopKey).Val() == 1 {
				return
			}
			res, err := s.rdb.Eval(
				ctx,
				luaFindProvidersFixed,
				[]string{"providers:geo"},
				lng,
				lat,
				radius,
				serviceID,
				cooldownSecs,
				excludeProviderID,
			).Result()

			if err != nil {
				log.Println("lua fixed search error:", err)
				continue
			}

			list, ok := res.([]interface{})
			if !ok || len(list) == 0 {
				continue
			}

			for i := 0; i < len(list); i += 2 {
				if s.rdb.Exists(ctx, stopKey).Val() == 1 {
					return
				}

				pid := list[i].(string)

				distStr := list[i+1].(string)
				dist, _ := strconv.ParseFloat(distStr, 64)

				distKm := round2(dist)
				eta := estimateETA(distKm)

				// ===============================
				// SOCKET
				// ===============================

				s.socket.Emit(
					"provider:"+pid,
					"bid:request:fixed",
					map[string]any{
						"user": map[string]any{
							"name":       user.Name,
							"profileUrl": user.ImageUrl,
						},
						"serviceId":  serviceID,
						"fixedPrice": fixedPriceGst,
						"serviceType":"fixedPrice",
						"vehicle": map[string]any{
							"type":   vehicleType,
							"number": vehicleNumber,
							"brand":  brand,
							"year":   modelYear,
							"fuel":   fuelType,
							"model":  model,
						},
						"issues":      issues,
						"distanceKm":  distKm,
						"etaMin":      eta,
						"expiresIn":   60,
					},
				)
			payload := map[string]any{
				"user": map[string]any{
					"name":       user.Name,
					"profileUrl": user.ImageUrl,
				},
				"serviceId":   serviceID,
				"fixedPrice":  fixedPriceGst,
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
				"distanceKm": distKm,
				"etaMin":     eta,
				"expiresIn":  60,
			}

			payloadBytes, _ := json.Marshal(payload)

			go s.notify.SendToProvider(
				context.Background(),
				pid,
				serviceID,
				"Service available",
				fmt.Sprintf("Fixed price ₹%.0f — open app", fixedPriceGst),
				map[string]string{
					"type":      "fixed_price_request",
					"serviceId": serviceID,
					"payload":   string(payloadBytes), // ✅ SAME AS SOCKET
				},
			)
		}

		time.Sleep(roundDelay)
	}
}
}
func (s *BiddingService) CancelService(ctx context.Context,serviceID string,userID string,reason string) error {

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	lockKey := "service:locked:" + serviceID

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {
		return errors.New("service not found")
	}

	if svc.User.Hex() != userID {
		return errors.New("not allowed")
	}
	now := time.Now()

	// 🔒 STOP search goroutines
	s.rdb.Set(ctx, lockKey, "1", 15*time.Minute)
	cancelledProvider := ""
	if svc.Provider != primitive.NilObjectID {

		cancelledProvider = svc.Provider.Hex()

		// remove busy lock
		s.rdb.Del(ctx, "provider:busy:"+cancelledProvider)

		// restore geo
		if svc.ProviderLocation != nil {

			s.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      cancelledProvider,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}
	}


	// ❌ DB UPDATE
	_, err = s.acceptedRepo.Col().UpdateByID(
		ctx,
		serviceOID,
		bson.M{
			"$set": bson.M{
				"status":      domain.StatusCancelled,
				"timestamps.cancelledAt": now,
				"cancelled.by": "user",
				"cancelled.reason": reason,
				"cancelledAt": time.Now(),
				"updatedAt":   time.Now(),
				"reason" : reason,

			},
		},
	)
	if err != nil {
		return err
	}

	// ===========================
	// 📡 INFORM PROVIDERS
	// ===========================

	notifiedKey := "service:notified:" + serviceID

	providers, _ := s.rdb.SMembers(ctx, notifiedKey).Result()

	for _, pid := range providers {
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

	// ===========================
	// 🧹 REDIS CLEANUP
	// ===========================

	patterns := []string{
		"service:cooldown:" + serviceID + ":*",
		"service:attempts:" + serviceID + ":*",
		"service:userReject:" + serviceID + ":*",
		"service:providerBidLock:" + serviceID + ":*",
	}

	for _, p := range patterns {
		iter := s.rdb.Scan(ctx, 0, p, 200).Iterator()
		for iter.Next(ctx) {
			s.rdb.Del(ctx, iter.Val())
		}
	}
	paymentTxn, err := s.paymentRepo.FindByServiceID(ctx, serviceID)
	if err == nil {

		refundLock := "refund:lock:" + paymentTxn.MihPayID

		ok, _ := s.rdb.SetNX(ctx, refundLock, "1", 24*time.Hour).Result()

		if ok {

			job := domain.RefundJob{
				ServiceID: serviceID,
				MihPayID: paymentTxn.MihPayID,
				Amount: paymentTxn.Amount,
				Retries: 0,
			}

			b, _ := json.Marshal(job)

			s.rdb.RPush(ctx, "refund:queue", b)

			s.refundRepo.Create(ctx, &domain.RefundTransaction{
				TxnID: paymentTxn.TxnID,
				MihPayID: paymentTxn.MihPayID,
				ServiceID: serviceID,
				UserID: paymentTxn.UserID,
				Amount: paymentTxn.Amount,
				Status: "pending",
				Reason: reason,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			})
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

		s.socket.CloseRoom("provider:" + cancelledProvider)
	}


	// ===========================
	// 👤 USER ROOM
	// ===========================

	s.socket.Emit(
		"user:"+serviceID,
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
		},
	)
	s.socket.CloseRoom("user:" + serviceID)
	cleanupServiceKeys(ctx, s.rdb, serviceID)

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

	// ===========================
	// 🚀 FAST: Redis pipeline
	// ===========================

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, lockKey, "1", 15*time.Minute)
	pipe.Set(ctx, stopKey, "1", 30*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	// ===========================
	// 🚀 Atomic DB update
	// ===========================

	res, err := s.acceptedRepo.Col().UpdateOne(
		ctx,
		bson.M{
			"_id":  serviceOID,
			"user": userOID,
			"status": bson.M{
				"$ne": domain.StatusCancelled,
			},
		},
		bson.M{
			"$set": bson.M{
				"status":      domain.StatusCancelled,
				"cancelledBy": "user",
				"cancelledAt": time.Now(),
				"updatedAt":   time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return errors.New("service not found or already cancelled")
	}

	// ===========================
	// 📡 INFORM USER
	// ===========================

	userRoom := "user:" + serviceID

	s.socket.Emit(
		userRoom,
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
		},
	)

	s.socket.CloseRoom(userRoom)

	// =====================================================
	// 📡 INFORM PROVIDERS — GEO BASED (DEDUPED)
	// =====================================================

	var svc domain.AcceptedService
	if err := s.acceptedRepo.Col().
		FindOne(ctx, bson.M{"_id": serviceOID}).
		Decode(&svc); err != nil {

		log.Println("❌ cancel geo load service failed:", err)
		goto CLEANUP
	}

	if svc.UserLocation != nil {

		radiusSteps := []float64{10, 25, 50, 100}

		sent := make(map[string]struct{})

		for _, radius := range radiusSteps {

			res, err := s.rdb.GeoRadius(
				ctx,
				"providers:geo",
				svc.UserLocation.Long,
				svc.UserLocation.Lat,
				&redis.GeoRadiusQuery{
					Radius: radius,
					Unit:   "km",
				},
			).Result()

			if err != nil {
				log.Println("❌ cancel geo query error:", err)
				continue
			}

			for _, loc := range res {

				providerID := loc.Name

				// ✅ prevent duplicates
				if _, ok := sent[providerID]; ok {
					continue
				}
				sent[providerID] = struct{}{}

				room := "provider:" + providerID

				s.socket.Emit(
					room,
					"service:cancelled",
					map[string]any{
						"serviceId": serviceID,
						"reason":    "cancelled_by_user",
					},
				)

				go s.notify.SendToProvider(
					context.Background(),
					providerID,
					serviceID,
					"Service Cancelled",
					"User cancelled the request.",
					map[string]string{
						"serviceId": serviceID,
					},
				)
			}
		}
	}

CLEANUP:

	// ===========================
	// 🧹 BACKGROUND REDIS CLEANUP
	// ===========================

	go func(serviceID string) {

		ctxBg := context.Background()

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

			iter := s.rdb.Scan(ctxBg, 0, pattern, 500).Iterator()

			batch := make([]string, 0, 100)

			for iter.Next(ctxBg) {

				batch = append(batch, iter.Val())

				if len(batch) >= 100 {
					s.rdb.Del(ctxBg, batch...)
					batch = batch[:0]
				}
			}

			if len(batch) > 0 {
				s.rdb.Del(ctxBg, batch...)
			}
		}

	}(serviceID)

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
			log.Println("⛔ timeout ignored — payment done", serviceID)
			return
		}

		// 🔒 redis lock so two workers don't race
		lockKey := "timeout:lock:" + serviceID

		ok, _ := s.rdb.SetNX(ctx, lockKey, "1", 2*time.Minute).Result()
		if !ok {
			return
		}

		log.Println("⚠ releasing timed-out provider", providerID, "service", serviceID)

		// =========================
		// 🔓 redis cleanup
		// =========================

		keys := []string{
			"service:locked:" + serviceID,
			"service:stop:" + serviceID,
			"provider:busy:" + providerID,
		}

		s.rdb.Del(ctx, keys...)

		cleanupServiceKeys(ctx, s.rdb, serviceID)

		// =========================
		// 🌍 re-add provider geo
		// =========================

		if svc.ProviderLocation != nil {

			s.rdb.GeoAdd(ctx, "providers:geo", &redis.GeoLocation{
				Name:      providerID,
				Longitude: svc.ProviderLocation.Long,
				Latitude:  svc.ProviderLocation.Lat,
			})
		}

		// =========================
		// 🔄 DB reset
		// =========================

		_ = s.acceptedRepo.UpdateByID(
			ctx,
			svc.ID,
			bson.M{
				"$set": bson.M{
					"status": domain.StatusSearching,
					"provider": primitive.NilObjectID,
					"acceptedBid": primitive.NilObjectID,
					"updatedAt": time.Now(),
				},
			},
		)

		// =========================
		// 🔔 notify user
		// =========================

		s.socket.Emit(
			"user:"+serviceID,
			"provider:timeout",
			map[string]any{
				"serviceId": serviceID,
			},
		)
	}







/* ================= HELPERS ================= */
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func estimateETA(distanceKm float64) int {
	if distanceKm <= 1 {
		return 5
	}
	return int(distanceKm*4 + 5)
}