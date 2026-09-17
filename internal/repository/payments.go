package repository

import (
	"sync"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

type PaymentsRepository struct {
	mu       sync.RWMutex
	payments map[string]models.PostPaymentResponse
}

func NewPaymentsRepository() *PaymentsRepository {
	return &PaymentsRepository{
		payments: make(map[string]models.PostPaymentResponse),
	}
}

func (ps *PaymentsRepository) GetPayment(id string) *models.PostPaymentResponse {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	payment, ok := ps.payments[id]
	if !ok {
		return nil
	}
	return &payment
}

func (ps *PaymentsRepository) AddPayment(payment models.PostPaymentResponse) (models.PostPaymentResponse, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.payments[payment.Id] = payment
	return payment, nil
}
