package repository

import (
	"fmt"
	"sync"
	"testing"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentsRepository(t *testing.T) {
	t.Run("SaveAndRetrieve", func(t *testing.T) {
		repo := NewPaymentsRepository()
		payment := models.PostPaymentResponse{
			Id: "payment-1", PaymentStatus: "Authorized", CardNumberLastFour: "0001",
			ExpiryMonth: 12, ExpiryYear: 2030, Currency: "GBP", Amount: 100,
		}
		saved, err := repo.AddPayment(payment)
		require.NoError(t, err)
		assert.Equal(t, payment, saved)
		stored := repo.GetPayment(payment.Id)
		require.NotNil(t, stored)
		assert.Equal(t, payment, *stored)
	})
	t.Run("MissingPayment", func(t *testing.T) {
		repo := NewPaymentsRepository()
		_, err := repo.AddPayment(models.PostPaymentResponse{Id: "existing"})
		require.NoError(t, err)
		assert.Nil(t, repo.GetPayment("missing"))
	})
	t.Run("IndependentPayments", func(t *testing.T) {
		repo := NewPaymentsRepository()
		first := models.PostPaymentResponse{Id: "first", Amount: 100}
		second := models.PostPaymentResponse{Id: "second", Amount: 200}
		_, err := repo.AddPayment(first)
		require.NoError(t, err)
		_, err = repo.AddPayment(second)
		require.NoError(t, err)
		assert.Equal(t, &first, repo.GetPayment(first.Id))
		assert.Equal(t, &second, repo.GetPayment(second.Id))
	})
	t.Run("SameIDReplacesPayment", func(t *testing.T) {
		repo := NewPaymentsRepository()
		payment := models.PostPaymentResponse{Id: "same", Amount: 100}
		_, err := repo.AddPayment(payment)
		require.NoError(t, err)
		payment.Amount = 200
		_, err = repo.AddPayment(payment)
		require.NoError(t, err)
		assert.Equal(t, &payment, repo.GetPayment(payment.Id))
	})
	t.Run("ReturnedRecordIsACopy", func(t *testing.T) {
		repo := NewPaymentsRepository()
		payment := models.PostPaymentResponse{Id: "copy", PaymentStatus: "Declined"}
		_, err := repo.AddPayment(payment)
		require.NoError(t, err)
		retrieved := repo.GetPayment(payment.Id)
		require.NotNil(t, retrieved)
		retrieved.PaymentStatus = "changed"
		assert.Equal(t, &payment, repo.GetPayment(payment.Id))
	})
}

func TestPaymentsRepositoryConcurrentAccess(t *testing.T) {
	repo := NewPaymentsRepository()
	const workers = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			payment := models.PostPaymentResponse{Id: fmt.Sprint(i), Amount: i + 1}
			for attempt := 0; attempt < 20; attempt++ {
				saved, err := repo.AddPayment(payment)
				if err != nil || saved != payment {
					t.Errorf("save %s: result=%+v error=%v", payment.Id, saved, err)
					return
				}
				stored := repo.GetPayment(payment.Id)
				if stored == nil || *stored != payment {
					t.Errorf("incorrect payment for %s: %+v", payment.Id, stored)
					return
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	for i := 0; i < workers; i++ {
		assert.Equal(t, &models.PostPaymentResponse{Id: fmt.Sprint(i), Amount: i + 1}, repo.GetPayment(fmt.Sprint(i)))
	}
}
