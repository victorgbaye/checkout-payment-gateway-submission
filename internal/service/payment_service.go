package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
)

var ErrValidation = errors.New("invalid payment")

type Bank interface {
	Authorize(
		ctx context.Context,
		request models.PostPaymentRequest,
	) (bool, error)
}

type PaymentService struct {
	repo *repository.PaymentsRepository
	bank Bank
}

func NewPaymentService(
	repo *repository.PaymentsRepository,
	bank Bank,
) *PaymentService {
	return &PaymentService{
		repo: repo,
		bank: bank,
	}
}

func (p *PaymentService) ProcessPayment(
	ctx context.Context,
	request models.PostPaymentRequest,
) (models.PostPaymentResponse, error) {
	if err := validatePayment(request); err != nil {
		return models.PostPaymentResponse{}, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return models.PostPaymentResponse{}, err
	}
	paymentID := hex.EncodeToString(idBytes[:])

	authorized, err := p.bank.Authorize(ctx, request)
	if err != nil {
		return models.PostPaymentResponse{}, err
	}

	status := "Declined"
	if authorized {
		status = "Authorized"
	}

	payment := models.PostPaymentResponse{
		Id:                 paymentID,
		PaymentStatus:      status,
		CardNumberLastFour: request.CardNumber[len(request.CardNumber)-4:],
		ExpiryMonth:        request.ExpiryMonth,
		ExpiryYear:         request.ExpiryYear,
		Currency:           request.Currency,
		Amount:             request.Amount,
	}

	p.repo.AddPayment(payment)
	return payment, nil
}

func validatePayment(request models.PostPaymentRequest) error {
	if len(request.CardNumber) < 14 ||
		len(request.CardNumber) > 19 ||
		!isDigits(request.CardNumber) {
		return errors.New("card number must contain 14 to 19 digits")
	}

	if request.ExpiryMonth < 1 || request.ExpiryMonth > 12 {
		return errors.New("expiry month must be between 1 and 12")
	}

	if request.ExpiryYear < 1 || request.ExpiryYear > 9999 {
		return errors.New("expiry year is invalid")
	}

	// Cards remain valid until the start of the following month.
	expiresAt := time.Date(
		request.ExpiryYear,
		time.Month(request.ExpiryMonth)+1,
		1, 0, 0, 0, 0,
		time.UTC,
	)

	if !expiresAt.After(time.Now()) {
		return errors.New("card has expired")
	}

	switch request.Currency {
	case "GBP", "USD", "EUR":
		// Supported currencies.
	default:
		return errors.New("currency must be GBP, USD, or EUR")
	}

	if request.Amount <= 0 {
		return errors.New("amount must be positive")
	}

	if len(request.Cvv) < 3 ||
		len(request.Cvv) > 4 ||
		!isDigits(request.Cvv) {
		return errors.New("cvv must contain 3 or 4 digits")
	}

	return nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}

	return true
}
