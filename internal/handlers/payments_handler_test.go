package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/service"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetPaymentHandler(t *testing.T) {
	payment := models.PostPaymentResponse{
		Id:                 "test-id",
		PaymentStatus:      "test-successful-status",
		CardNumberLastFour: "1234",
		ExpiryMonth:        10,
		ExpiryYear:         2035,
		Currency:           "GBP",
		Amount:             100,
	}
	ps := repository.NewPaymentsRepository()
	ps.AddPayment(payment)

	payments := NewPaymentsHandler(ps, service.NewPaymentService(ps, approvingBank{}))

	r := chi.NewRouter()
	r.Get("/api/payments/{id}", payments.GetHandler())

	t.Run("PaymentFound", func(t *testing.T) {
		// Create a new HTTP request for testing
		req, _ := http.NewRequest("GET", "/api/payments/test-id", nil)

		// Create a new HTTP request recorder for recording the response
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Check the body is not nil
		assert.NotNil(t, w.Body)

		// Check the HTTP status code in the response
		if status := w.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, http.StatusOK)
		}
	})
	t.Run("PaymentNotFound", func(t *testing.T) {
		// Create a new HTTP request for testing with a non-existing payment ID
		req, _ := http.NewRequest("GET", "/api/payments/NonExistingID", nil)

		// Create a new HTTP request recorder for recording the response
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		// Check the HTTP status code in the response
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func testPostPaymentCreated(t *testing.T) {

	body := strings.NewReader(fmt.Sprintf(`{
		"card_number": "2222405343248877",
		"expiry_month": 12,
		"expiry_year": %d,
		"currency": "GBP",
		"amount": 100,
		"cvv": "123"
	}`, time.Now().Year()+1))
	request := httptest.NewRequest(http.MethodPost, "/api/payments", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	repo := repository.NewPaymentsRepository()
	paymentService := service.NewPaymentService(repo, approvingBank{})
	handler := NewPaymentsHandler(repo, paymentService)
	handler.PostHandler()(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code, recorder.Body.String())
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	var payment models.PostPaymentResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payment))
	assert.NotEmpty(t, payment.Id)
	assert.Equal(t, "Authorized", payment.PaymentStatus)
	assert.Equal(t, "8877", payment.CardNumberLastFour)
	assert.Equal(t, 100, payment.Amount)
	assert.Equal(t, &payment, repo.GetPayment(payment.Id))
	assert.NotContains(t, recorder.Body.String(), "2222405343248877")
	assert.NotContains(t, recorder.Body.String(), "cvv")
}

// The application uses the supplied simulator; this test double keeps unit tests independent of Docker.
type approvingBank struct{}

func (approvingBank) Authorize(context.Context, models.PostPaymentRequest) (bool, error) {
	return true, nil
}

type failingBank struct{}

func (failingBank) Authorize(context.Context, models.PostPaymentRequest) (bool, error) {
	return false, errors.New("bank unavailable")
}

func testPostPaymentMalformedJSON(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	handler := NewPaymentsHandler(repo, service.NewPaymentService(repo, approvingBank{}))
	recorder := httptest.NewRecorder()
	handler.PostHandler()(recorder, httptest.NewRequest(http.MethodPost, "/api/payments", strings.NewReader("{")))
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func testPostPaymentBankFailure(t *testing.T) {
	repo := repository.NewPaymentsRepository()
	handler := NewPaymentsHandler(repo, service.NewPaymentService(repo, failingBank{}))
	body := fmt.Sprintf(`{"card_number":"2222405343248877","expiry_month":12,"expiry_year":%d,"currency":"GBP","amount":100,"cvv":"123"}`, time.Now().Year()+1)
	recorder := httptest.NewRecorder()
	handler.PostHandler()(recorder, httptest.NewRequest(http.MethodPost, "/api/payments", strings.NewReader(body)))
	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestPostPaymentHandler(t *testing.T) {
	t.Run("Created", testPostPaymentCreated)
	t.Run("MalformedJSON", testPostPaymentMalformedJSON)
	t.Run("BankFailure", testPostPaymentBankFailure)
	t.Run("RejectsInvalidAmount", func(t *testing.T) {
		repo := repository.NewPaymentsRepository()
		bank := &countingBank{}
		handler := NewPaymentsHandler(repo, service.NewPaymentService(repo, bank))
		body := fmt.Sprintf(`{"card_number":"2222405343248877","expiry_month":12,"expiry_year":%d,"currency":"GBP","amount":-100,"cvv":"123"}`, time.Now().Year()+1)
		request := httptest.NewRequest(http.MethodPost, "/api/payments", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.PostHandler()(recorder, request)
		assert.Zero(t, bank.calls, "invalid payments must not reach the bank")
		require.Equal(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		var response struct {
			Error         string `json:"error"`
			PaymentStatus string `json:"payment_status"`
		}
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, "Rejected", response.PaymentStatus)
		assert.Contains(t, response.Error, "amount must be positive")
	})
}

type countingBank struct{ calls int }

func (b *countingBank) Authorize(context.Context, models.PostPaymentRequest) (bool, error) {
	b.calls++
	return true, nil
}
