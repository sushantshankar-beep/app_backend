package payment

import (
	"net/http"
	"os"
	"time"

	"app_backend/internal/repository"
)

type Service struct {
	repo       *repository.TransactionRepo
	httpClient *http.Client
	key        string
	salt       string
	baseURL    string
	baseAppURL string
	notifyCh   chan any
}

func NewService(repo *repository.TransactionRepo) *Service {
	s := &Service{
		repo: repo,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		key:        os.Getenv("PAYU_KEY"),
		salt:       os.Getenv("PAYU_SALT"),
		baseURL:    os.Getenv("PAYU_BASE_URL"),
		baseAppURL: os.Getenv("BASE_URL"),
		notifyCh:   make(chan any, 200),
	}

	go s.notificationWorker()
	return s
}
