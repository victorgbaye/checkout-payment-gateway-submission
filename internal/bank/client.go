package bank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
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
		return false, fmt.Errorf("call bank: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return false, fmt.Errorf(
			"bank returned HTTP %d",
			response.StatusCode,
		)
	}

	var result struct {
		Authorized *bool `json:"authorized"`
	}

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decode bank response: %w", err)
	}

	if result.Authorized == nil {
		return false, fmt.Errorf("bank response is missing authorized")
	}

	return *result.Authorized, nil
}