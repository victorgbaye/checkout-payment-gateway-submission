package repository

import (
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

type PaymentsRepository struct {
	payments map[string]models.PostPaymentResponse
}

func NewPaymentsRepository() *PaymentsRepository {
	return &PaymentsRepository{
		payments: map[string]models.PostPaymentResponse{},
	}
}

func (ps *PaymentsRepository) GetPayment(id string) *models.PostPaymentResponse {
	for _, element := range ps.payments {
		if element.Id == id {
			return &element
		}
	}
	return nil
}

func (ps *PaymentsRepository) AddPayment(payment models.PostPaymentResponse) (models.PostPaymentResponse, error) {
	ps.payments[payment.Id] = payment
	return payment, nil
}
