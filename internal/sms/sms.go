package sms

import (
	"context"
	"log"
)

type DummySMS struct{}

func NewDummySMS() *DummySMS {
	return &DummySMS{}
}

func (d *DummySMS) SendOTP(ctx context.Context, phone, msg string) error {
	log.Println("[SMS]", phone, msg)
	return nil
}
