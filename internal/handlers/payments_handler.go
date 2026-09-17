package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/service"
	"github.com/go-chi/chi/v5"
)

type PaymentsHandler struct {
	storage *repository.PaymentsRepository
	service *service.PaymentService
}

func NewPaymentsHandler(storage *repository.PaymentsRepository, paymentService *service.PaymentService) *PaymentsHandler {
	return &PaymentsHandler{storage: storage, service: paymentService}
}

func (h *PaymentsHandler) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payment := h.storage.GetPayment(chi.URLParam(r, "id"))
		if payment == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "payment not found"})
			return
		}
		writeJSON(w, http.StatusOK, payment)
	}
}

func (h *PaymentsHandler) PostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request *models.PostPaymentRequest
		if err := decoder.Decode(&request); err != nil {
			rejectBody(w, err)
			return
		}
		if request == nil {
			rejectBody(w, nil)
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF {
			rejectBody(w, err)
			return
		}
		payment, err := h.service.ProcessPayment(r.Context(), *request)
		if err != nil {
			status := http.StatusInternalServerError
			body := map[string]string{"error": "unable to process payment"}
			switch {
			case errors.Is(err, service.ErrValidation):
				status = http.StatusBadRequest
				body["error"] = err.Error()
				body["payment_status"] = "Rejected"
			case errors.Is(err, bank.ErrUnavailable):
				status = http.StatusServiceUnavailable
				body["error"] = "bank unavailable"
			case errors.Is(err, bank.ErrInvalidResponse):
				status = http.StatusBadGateway
				body["error"] = "invalid bank response"
			}
			writeJSON(w, status, body)
			return
		}
		writeJSON(w, http.StatusCreated, payment)
	}
}

func rejectBody(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	message := "body must contain one valid payment JSON object"
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		status = http.StatusRequestEntityTooLarge
		message = "request body exceeds 4096 bytes"
	}
	writeJSON(w, status, map[string]string{"error": message, "payment_status": "Rejected"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	// Encode before sending headers so encoding errors can still return HTTP 500.
	body, err := json.Marshal(value)
	if err != nil {
		status = http.StatusInternalServerError
		body = []byte(`{"error":"unable to encode response"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}
