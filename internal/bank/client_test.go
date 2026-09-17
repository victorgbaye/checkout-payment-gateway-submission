package bank

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPayment() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber: "0222405343240001", ExpiryMonth: 4, ExpiryYear: 2030,
		Currency: "GBP", Amount: 100, Cvv: "012",
	}
}

func TestAuthorizePayloadAndDecision(t *testing.T) {
	for _, authorized := range []bool{true, false} {
		name := "Declined"
		if authorized {
			name = "Authorized"
		}
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/payments", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				if !assert.NoError(t, err) {
					w.WriteHeader(500)
					return
				}
				assert.JSONEq(t, `{"card_number":"0222405343240001","expiry_date":"04/2030","currency":"GBP","amount":100,"cvv":"012"}`, string(body))
				_ = json.NewEncoder(w).Encode(map[string]any{"authorized": authorized, "authorization_code": "example"})
			}))
			defer server.Close()
			// A trailing slash on the base URL must not produce a double slash in the path.
			got, err := NewClient(server.URL+"/").Authorize(context.Background(), testPayment())
			require.NoError(t, err)
			assert.Equal(t, authorized, got)
		})
	}
}

func TestAuthorizeTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	client := NewClient(server.URL)
	require.Positive(t, client.http.Timeout, "bank requests must have a finite timeout")
	client.http.Timeout = 50 * time.Millisecond
	// The outer deadline bounds the test if the client timeout stops working.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	authorized, err := client.Authorize(ctx, testPayment())
	assert.False(t, authorized)
	require.ErrorIs(t, err, ErrUnavailable)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.NoError(t, ctx.Err(), "the client timeout should fire before the outer deadline")
}

func TestAuthorizeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		// Cancel only after the bank receives the request.
		cancel()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	authorized, err := NewClient(server.URL).Authorize(ctx, testPayment())
	assert.False(t, authorized)
	require.ErrorIs(t, err, ErrUnavailable)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestAuthorizeDoesNotFollowRedirects(t *testing.T) {
	var forwarded atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded.Add(1)
		_, _ = w.Write([]byte(`{"authorized":true}`))
	}))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	authorized, err := NewClient(server.URL).Authorize(context.Background(), testPayment())
	assert.False(t, authorized)
	assert.ErrorIs(t, err, ErrInvalidResponse)
	assert.Zero(t, forwarded.Load(), "card details must not be forwarded to a redirect destination")
}
