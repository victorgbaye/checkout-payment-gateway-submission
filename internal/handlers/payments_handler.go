package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/service"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/go-chi/chi/v5"
)

type PaymentsHandler struct {
	storage *repository.PaymentsRepository
	service *service.PaymentService
}

func NewPaymentsHandler(storage *repository.PaymentsRepository, paymentService *service.PaymentService) *PaymentsHandler {
	return &PaymentsHandler{
		storage: storage,
		service: paymentService,
	}
}

// GetHandler returns an http.HandlerFunc that handles HTTP GET requests.
// It retrieves a payment record by its ID from the storage.
// The ID is expected to be part of the URL.
func (h *PaymentsHandler) GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		payment := h.storage.GetPayment(id)

		if payment != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(payment); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (h *PaymentsHandler) PostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request models.PostPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid payment JSON", http.StatusBadRequest)
			return
		}
		payment, err := h.service.ProcessPayment(r.Context(), request)
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":          err.Error(),
					"payment_status": "Rejected",
				})
				return
			}
			http.Error(w, "unable to process payment", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(payment)
	}
}
