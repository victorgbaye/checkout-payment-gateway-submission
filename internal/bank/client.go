package bank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

var (
	ErrUnavailable     = errors.New("bank unavailable")
	ErrInvalidResponse = errors.New("invalid bank response")
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(
				req *http.Request,
				via []*http.Request,
			) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) Authorize(
	ctx context.Context,
	payment models.PostPaymentRequest,
) (bool, error) {
	payload := struct {
		CardNumber string `json:"card_number"`
		ExpiryDate string `json:"expiry_date"`
		Currency   string `json:"currency"`
		Amount     int    `json:"amount"`
		Cvv        string `json:"cvv"`
	}{
		CardNumber: payment.CardNumber,
		ExpiryDate: fmt.Sprintf(
			"%02d/%04d",
			payment.ExpiryMonth,
			payment.ExpiryYear,
		),
		Currency: payment.Currency,
		Amount:   payment.Amount,
		Cvv:      payment.Cvv,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return false, fmt.Errorf("encode bank request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/payments",
		bytes.NewReader(body),
	)
	if err != nil {
		return false, fmt.Errorf("create bank request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := c.http.Do(req)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 500 {
		return false, fmt.Errorf("%w: HTTP %d", ErrUnavailable, response.StatusCode)
	}
	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("%w: HTTP %d", ErrInvalidResponse, response.StatusCode)
	}

	var result struct {
		Authorized *bool `json:"authorized"`
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&result); err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	if result.Authorized == nil {
		return false, fmt.Errorf("%w: missing authorized", ErrInvalidResponse)
	}

	if err := decoder.Decode(new(any)); err != io.EOF {
		return false, ErrInvalidResponse
	}

	return *result.Authorized, nil
}
