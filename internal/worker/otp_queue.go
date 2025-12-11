package worker

import (
	"context"
	"log"

	"app_backend/internal/ports"
)

type OTPJob struct {
	Phone string
	Msg   string
}

type OTPQueue struct {
	sms  ports.SMSClient
	jobs chan OTPJob
	stop context.CancelFunc
}

func NewOTPQueue(sms ports.SMSClient) *OTPQueue {
	return &OTPQueue{
		sms:  sms,
		jobs: make(chan OTPJob, 50),
	}
}

func (q *OTPQueue) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	q.stop = cancel

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-q.jobs:
				if err := q.sms.SendOTP(ctx, job.Phone, job.Msg); err != nil {
					log.Println("OTP send failed:", err)
				}
			}
		}
	}()
}

func (q *OTPQueue) Stop() {
	if q.stop != nil {
		q.stop()
	}
}

func (q *OTPQueue) Enqueue(job OTPJob) {
	select {
	case q.jobs <- job:
	default:
		log.Println("OTP queue full — dropping job")
	}
}