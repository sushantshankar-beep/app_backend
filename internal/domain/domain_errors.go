package domain

import "errors"

var (
	ErrNotFound   = errors.New("not found")
	ErrOTPInvalid = errors.New("invalid otp")
	ErrOTPExpired = errors.New("otp expired")
)
