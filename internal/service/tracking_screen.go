package service

import "app_backend/internal/domain"

func BuildTrackingScreen(
	svc domain.AcceptedService,
	user domain.User,
) map[string]any {

	return map[string]any{
		"screen": "SERVICE_TRACKING",

		"bookingId": svc.NumericID,
		"status":    svc.Status,

		"otp": user.ServiceOTP,

		"timeline": []map[string]any{
			{"label": "Booking Accepted", "done": true},
			{"label": "Mechanic on the Way", "done": svc.ReachedAt != nil},
			{"label": "Service Started", "done": svc.StartedAt != nil},
			{"label": "Service Completed", "done": svc.CompletedAt != nil},
		},

		"actions": []map[string]any{
			{"label": "Enter OTP", "type": "PRIMARY"},
			{"label": "Raise Complaint", "type": "DANGER"},
		},
	}
}
