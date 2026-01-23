package domain

import "errors"

/* ================== SERVICE STATUS ================== */

// type ServiceStatus string

// const (
// 	StatusSearching        ServiceStatus = "searching"
// 	StatusProviderAssigned ServiceStatus = "provider_assigned"

// 	StatusNotStarted      ServiceStatus = "not_started"
// 	StatusStarted         ServiceStatus = "started"
// 	StatusReachedLocation ServiceStatus = "reached_location"
// 	StatusOTPVerified     ServiceStatus = "otp_verified"

// 	StatusInProgress ServiceStatus = "in_progress"
// 	StatusCompleted  ServiceStatus = "completed"
// 	StatusCancelled  ServiceStatus = "cancelled"
// )

/* ================== PAYMENT STATUS ================== */



/* ================== STATE GUARDS ================== */

func CanStartService(s *AcceptedService) error {
	if s.Status != StatusProviderAssigned {
		return errors.New("provider not assigned")
	}
	if s.PaymentStatus != PaymentPaid {
		return errors.New("payment pending")
	}
	return nil
}


func CanCompleteService(s *AcceptedService) error {
	if s.Status != StatusInProgress {
		return errors.New("service not in progress")
	}
	if s.PaymentStatus != PaymentPaid {
		return errors.New("payment not completed")
	}
	return nil
}

func CanEnterPaymentGrace(s *AcceptedService) error {
	if s.PaymentStatus != PaymentPending {
		return errors.New("cannot enter payment grace")
	}
	return nil
}

func CanReleaseProviderAfterFailure(s *AcceptedService) error {
	if s.PaymentStatus != PaymentFailedGrace {
		return errors.New("not in grace period")
	}
	return nil
}
