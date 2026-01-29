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

	  -- skip busy
	  if redis.call("EXISTS", "provider:busy:"..pid) == 1 then
	    goto continue
	  end

	  -- skip already active
	  local activeKey = "service:activeProvider:"..ARGV[4]..":"..pid
	  if redis.call("EXISTS", activeKey) == 1 then
	    goto continue
	  end

	  -- cooldown
	  local cdKey = "service:cooldown:"..ARGV[4]..":"..pid
	  if redis.call("SETNX", cdKey, "1") == 0 then
	    goto continue
	  end

	  redis.call("EXPIRE", cdKey, ARGV[5])

	  -- send count
	  local scKey = "service:sendcount:"..ARGV[4]..":"..pid
	  local cnt = redis.call("INCR", scKey)
	  redis.call("EXPIRE", scKey, ARGV[7])

	  if cnt > tonumber(ARGV[6]) then
	    goto continue
	  end

	  -- cache distance
	  redis.call(
	    "SET",
	    "service:dist:"..ARGV[4]..":"..pid,
	    dist,
	    "EX",
	    1800
	  )

	  -- mark active
	  redis.call("SET", activeKey, "1", "EX", ARGV[5])

	  table.insert(out, pid)
	  table.insert(out, dist)

	  ::continue::
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
		cooldownSec    = 45
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
							"expiresIn":   cooldownSec,
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
	svc, _ := s.acceptedRepo.FindByID(ctx, serviceID)
	log.Println("this is user id before sending notify",svc.User.Hex())
	go s.notify.SendToUser(
		context.Background(),
		svc.User.Hex(),
		"New Bid Received",
		"A provider placed a bid.",
		map[string]string{
			"serviceId": serviceID,
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
) error {
	assignKey := "service:assigning:" + serviceID
	stopKey := "service:stop:" + serviceID
	lockKey := "service:locked:" + serviceID
	ok, err := s.rdb.SetNX(ctx, assignKey, "1", 10*time.Second).Result()
	if err != nil || !ok {
		return ErrServiceAlreadyAssigned
	}
	defer s.rdb.Del(ctx, assignKey)

	// already closed?
	if s.rdb.Exists(ctx, lockKey).Val() == 1 {
		return ErrServiceAlreadyAssigned
	}

	// STOP findProviders immediately
	s.rdb.Set(ctx, stopKey, "1", 5*time.Minute)

	serviceOID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return err
	}

	bidOID, err := primitive.ObjectIDFromHex(bidID)
	if err != nil {
		return err
	}

	providerOID, err := primitive.ObjectIDFromHex(providerID)
	if err != nil {
		return err
	}
	res, err := s.acceptedRepo.Col().UpdateOne(
		ctx,
		bson.M{
			"_id":    serviceOID,
		},
		bson.M{
			"$set": bson.M{
				"provider":      providerOID,
				"acceptedBid":   bidOID,
				"finalPrice":    price,
				"status":        domain.StatusProviderAssigned,
				"paymentStatus": domain.PaymentPending,
				"updatedAt":     time.Now(),
			},
		},
	)

	if err != nil {
		return err
	}

	if res.MatchedCount == 0 {
		return ErrServiceAlreadyAssigned
	}
	s.rdb.Set(ctx,
		"service:locked:"+serviceID,
		"1",
		30*time.Minute,
	)
	pos, err := s.rdb.GeoPos(ctx, "providers:geo", providerID).Result()
	if err != nil {
		log.Println("redis geopos failed:", err)
	}
	var lat, long float64
	if len(pos) > 0 && pos[0] != nil {
		long = pos[0].Longitude
		lat = pos[0].Latitude
	}
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
	// 🔒 Assign provider & lock service
	if err := s.acceptedRepo.UpdateByID(
		ctx,
		serviceOID,
		bson.M{"$set": set},
	); err != nil {
		return err
	}
	s.rdb.Set(ctx,
		"provider:busy:"+providerID,
		serviceID,
		2*time.Hour, // auto release safety
	)

	// ✅ REMOVE PROVIDER FROM GEO (VERY IMPORTANT)
	s.rdb.ZRem(ctx, "providers:geo", providerID)

	// 🔔 Notify PROVIDER
	s.socket.Emit(
		"provider:"+providerID,
		"bid:accepted",
		map[string]any{
			"serviceId": serviceID,
			"price":     price,
		},
	)

	// 🔔 Notify USER (recommended)
	s.socket.Emit(
		"user:"+serviceID,
		"service:assigned",
		map[string]any{
			"serviceId":  serviceID,
			"providerId": providerID,
			"price":      price,
		},
	)
	log.Println("this is provider id before placing bid",providerID)
	go s.notify.SendToProvider(
		context.Background(),
		providerID,
		"Bid Accepted 🎉",
		"User accepted your bid.",
		map[string]string{
			"serviceId": serviceID,
		},
	)


	return nil
}
func (s *BiddingService) RejectBid(
	ctx context.Context,
	serviceID string,
	providerID string,
	price int,
) error {

	log.Printf("👎 rejected provider=%s service=%s", providerID, serviceID)

	activeKey := "service:activeProvider:" + serviceID + ":" + providerID

	// allow rebid window
	s.rdb.Set(ctx, activeKey, "1", 60*time.Second)

	svc, err := s.acceptedRepo.FindByID(ctx, serviceID)
	if err != nil {
		return err
	}

	dist, _ := s.rdb.Get(
		ctx,
		"service:dist:"+serviceID+":"+providerID,
	).Float64()

	// 🔔 PROVIDER
	s.socket.Emit(
		"provider:"+providerID,
		"bid:rejected",
		map[string]any{
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
		},
	)

	go s.notify.SendToProvider(
		context.Background(),
		providerID,
		"Bid Rejected",
		"Customer rejected your bid. You can offer a new price.",
		map[string]string{
			"serviceId": serviceID,
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
	lockKey := "service:locked:" + serviceID
	s.rdb.Del(ctx, lockKey)

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
				"status":        domain.StatusSearching,
				"timestamps.cancelledAt":now,
				"cancelled.by": "provider",
				"cancelled.reason": reason,
				"provider":      primitive.NilObjectID,
				"acceptedBid":   primitive.NilObjectID,
				"fixedPrice":    fixedPrice,
				"updatedAt":     time.Now(),
				"cancelledByProvider": true,
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
		},
	)

	go s.notify.SendToUser(
		context.Background(),
		svc.User.Hex(),
		"Provider cancelled",
		"Searching new provider at same price.",
		map[string]string{
			"serviceId": serviceID,
		},
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
	)

	return nil
}
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
) {

	ctx := context.Background()

	lockKey := "service:locked:" + serviceID

	radiusSteps := []float64{5, 10, 20, 50}

	for _, radius := range radiusSteps {

		if s.rdb.Exists(ctx, lockKey).Val() == 1 {
			return
		}

		providers, err := s.rdb.GeoRadius(
		    ctx,
		    "providers:geo",
		    lng,
		    lat,
		    &redis.GeoRadiusQuery{
		        Radius: radius,
		        Unit:   "km",
		        Count:  50,
		    },
		).Result()

		if err != nil || len(providers) == 0 {
		    continue
		}


		for _, p := range providers {

			pid := p.Name

			s.socket.Emit(
				"provider:"+pid,
				"bid:request:fixed",
				map[string]any{
					"serviceId": serviceID,
					"fixedPrice": fixedPrice,
					"vehicle": map[string]any{
						"type":   vehicleType,
						"number": vehicleNumber,
						"brand":  brand,
						"year":   modelYear,
						"fuel":   fuelType,
						"model":  model,
					},
					"issues": issues,
				},
			)

			go s.notify.SendToProvider(
				context.Background(),
				pid,
				"Service available",
				fmt.Sprintf("Fixed price ₹%.0f — open app", fixedPrice),
				map[string]string{
					"serviceId": serviceID,
				},
			)
		}
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
	}

	for _, p := range patterns {
		iter := s.rdb.Scan(ctx, 0, p, 200).Iterator()
		for iter.Next(ctx) {
			s.rdb.Del(ctx, iter.Val())
		}
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
	// 🚀 FAST: Atomic DB update
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
	// 📡 INFORM USER IMMEDIATELY
	// ===========================

	s.socket.Emit(
		"user:"+serviceID,
		"service:cancelled",
		map[string]any{
			"serviceId": serviceID,
		},
	)

	s.socket.CloseRoom("user:" + serviceID)

	// ===========================
	// 🧹 BACKGROUND REDIS CLEANUP
	// ===========================

	go func(serviceID string) {

		ctxBg := context.Background()

		patterns := []string{
			"service:cooldown:" + serviceID + ":*",
			"service:attempts:" + serviceID + ":*",
			"service:userReject:" + serviceID + ":*",
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






/* ================= HELPERS ================= */

func estimateETA(distanceKm float64) int {
	if distanceKm <= 1 {
		return 5
	}
	return int(distanceKm*4 + 5)
}