package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingBank struct {
	calls int
}

func (b *countingBank) Authorize(context.Context, models.PostPaymentRequest) (bool, error) {
	b.calls++
	return true, nil
}

func validPaymentRequest() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  "2222405343240001",
		ExpiryMonth: 12,
		ExpiryYear:  time.Now().UTC().Year() + 1,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "012",
	}
}

func TestProcessPaymentValidation(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*models.PostPaymentRequest)
		message string
	}{
		{"MissingCardNumber", func(p *models.PostPaymentRequest) { p.CardNumber = "" }, "card number"},
		{"CardTooShort", func(p *models.PostPaymentRequest) { p.CardNumber = strings.Repeat("1", 13) }, "card number"},
		{"CardTooLong", func(p *models.PostPaymentRequest) { p.CardNumber = strings.Repeat("1", 20) }, "card number"},
		{"CardContainsLetter", func(p *models.PostPaymentRequest) { p.CardNumber = "222240534324887x" }, "card number"},
		{"CardContainsSpace", func(p *models.PostPaymentRequest) { p.CardNumber = "222240534324887 " }, "card number"},
		{"CardContainsUnicodeDigit", func(p *models.PostPaymentRequest) { p.CardNumber = "２222405343248877" }, "card number"},
		{"MissingMonth", func(p *models.PostPaymentRequest) { p.ExpiryMonth = 0 }, "expiry month"},
		{"NegativeMonth", func(p *models.PostPaymentRequest) { p.ExpiryMonth = -1 }, "expiry month"},
		{"MonthAboveTwelve", func(p *models.PostPaymentRequest) { p.ExpiryMonth = 13 }, "expiry month"},
		{"MissingYear", func(p *models.PostPaymentRequest) { p.ExpiryYear = 0 }, "expiry year"},
		{"NegativeYear", func(p *models.PostPaymentRequest) { p.ExpiryYear = -1 }, "expiry year"},
		{"YearAboveMaximum", func(p *models.PostPaymentRequest) { p.ExpiryYear = 10000 }, "expiry year"},
		{"ExpiredYear", func(p *models.PostPaymentRequest) { p.ExpiryYear = time.Now().UTC().Year() - 1 }, "card has expired"},
		{"ExpiredPreviousMonth", func(p *models.PostPaymentRequest) {
			now := time.Now().UTC()
			previousMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
			p.ExpiryMonth = int(previousMonth.Month())
			p.ExpiryYear = previousMonth.Year()
		}, "card has expired"},
		{"MissingCurrency", func(p *models.PostPaymentRequest) { p.Currency = "" }, "currency"},
		{"UnsupportedCurrency", func(p *models.PostPaymentRequest) { p.Currency = "JPY" }, "currency"},
		{"LowercaseCurrency", func(p *models.PostPaymentRequest) { p.Currency = "gbp" }, "currency"},
		{"ShortCurrency", func(p *models.PostPaymentRequest) { p.Currency = "GB" }, "currency"},
		{"LongCurrency", func(p *models.PostPaymentRequest) { p.Currency = "GBPP" }, "currency"},
		// An omitted amount is also zero after decoding into the current int field.
		{"ZeroAmount", func(p *models.PostPaymentRequest) { p.Amount = 0 }, "amount"},
		{"NegativeAmount", func(p *models.PostPaymentRequest) { p.Amount = -100 }, "amount"},
		{"MissingCVV", func(p *models.PostPaymentRequest) { p.Cvv = "" }, "cvv"},
		{"CVVTooShort", func(p *models.PostPaymentRequest) { p.Cvv = "12" }, "cvv"},
		{"CVVTooLong", func(p *models.PostPaymentRequest) { p.Cvv = "12345" }, "cvv"},
		{"CVVContainsLetter", func(p *models.PostPaymentRequest) { p.Cvv = "12x" }, "cvv"},
		{"CVVContainsSpace", func(p *models.PostPaymentRequest) { p.Cvv = "12 " }, "cvv"},
		{"CVVContainsUnicodeDigit", func(p *models.PostPaymentRequest) { p.Cvv = "１" }, "cvv"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validPaymentRequest()
			tt.change(&request)
			bank := &countingBank{}
			paymentService := NewPaymentService(repository.NewPaymentsRepository(), bank)

			payment, err := paymentService.ProcessPayment(context.Background(), request)

			assert.Zero(t, bank.calls, "rejected requests must not reach the bank")
			require.ErrorIs(t, err, ErrValidation)
			assert.Contains(t, err.Error(), tt.message)
			assert.Equal(t, models.PostPaymentResponse{}, payment)
		})
	}
}

func TestProcessPaymentValidBoundaries(t *testing.T) {
	tests := []struct {
		name   string
		change func(*models.PostPaymentRequest)
	}{
		{"DefaultValidRequest", func(*models.PostPaymentRequest) {}},
		{"FourteenCardDigits", func(p *models.PostPaymentRequest) { p.CardNumber = strings.Repeat("1", 14) }},
		{"NineteenCardDigits", func(p *models.PostPaymentRequest) { p.CardNumber = strings.Repeat("1", 19) }},
		{"JanuaryExpiry", func(p *models.PostPaymentRequest) { p.ExpiryMonth = 1 }},
		{"DecemberExpiry", func(p *models.PostPaymentRequest) { p.ExpiryMonth = 12 }},
		{"USD", func(p *models.PostPaymentRequest) { p.Currency = "USD" }},
		{"EUR", func(p *models.PostPaymentRequest) { p.Currency = "EUR" }},
		{"OneMinorUnit", func(p *models.PostPaymentRequest) { p.Amount = 1 }},
		{"ThreeDigitCVV", func(p *models.PostPaymentRequest) { p.Cvv = "012" }},
		{"FourDigitCVV", func(p *models.PostPaymentRequest) { p.Cvv = "0012" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := validPaymentRequest()
			tt.change(&request)
			bank := &countingBank{}
			repo := repository.NewPaymentsRepository()
			paymentService := NewPaymentService(repo, bank)

			payment, err := paymentService.ProcessPayment(context.Background(), request)

			require.NoError(t, err)
			assert.Equal(t, 1, bank.calls)
			assert.Equal(t, "Authorized", payment.PaymentStatus)
			assert.Equal(t, request.CardNumber[len(request.CardNumber)-4:], payment.CardNumberLastFour)
			assert.Equal(t, &payment, repo.GetPayment(payment.Id))
		})
	}
}

// A decline is a valid bank decision, not a bank communication error.
type decliningBank struct {
	calls int
}

func (b *decliningBank) Authorize(context.Context, models.PostPaymentRequest) (bool, error) {
	b.calls++
	return false, nil
}

func TestProcessPaymentDeclinedIsSavedAndRetrievable(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	bank := &decliningBank{}
	paymentService := NewPaymentService(repo, bank)
	request := validPaymentRequest()
	request.CardNumber = "2222405343240002"

	payment, err := paymentService.ProcessPayment(context.Background(), request)

	require.NoError(t, err)
	assert.Equal(t, 1, bank.calls)
	require.NotEmpty(t, payment.Id)
	assert.Equal(t, models.PostPaymentResponse{
		Id:                 payment.Id,
		PaymentStatus:      "Declined",
		CardNumberLastFour: "0002",
		ExpiryMonth:        request.ExpiryMonth,
		ExpiryYear:         request.ExpiryYear,
		Currency:           request.Currency,
		Amount:             request.Amount,
	}, payment)

	stored := repo.GetPayment(payment.Id)
	require.NotNil(t, stored, "declined payments must be saved")
	assert.Equal(t, payment, *stored)
	assert.Equal(t, 1, bank.calls, "retrieval must not contact the bank again")
}
