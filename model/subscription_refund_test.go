package model

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSubscriptionRefundIsAtomicAndIdempotent(t *testing.T) {
	setupInviteCommissionSubscriptionTest(t)
	require.NoError(t, DB.AutoMigrate(&SubscriptionPreConsumeRecord{}))
	require.NoError(t, DB.Create(&UserSubscription{Id: 771001, UserId: 771002, AmountTotal: 1000, AmountUsed: 300}).Error)
	require.NoError(t, DB.Create(&SubscriptionPreConsumeRecord{RequestId: "refund-atomic", UserId: 771002, UserSubscriptionId: 771001, PreConsumed: 200, Status: "consumed"}).Error)
	const callback = "test:fail_refund_marker"
	injected := errors.New("refund marker update failed")
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "subscription_pre_consume_records" {
			tx.AddError(injected)
		}
	}))
	err := RefundSubscriptionPreConsume("refund-atomic")
	require.NoError(t, DB.Callback().Update().Remove(callback))
	require.ErrorIs(t, err, injected)
	var sub UserSubscription
	var record SubscriptionPreConsumeRecord
	require.NoError(t, DB.First(&sub, 771001).Error)
	require.EqualValues(t, 300, sub.AmountUsed, "failed marker update must roll back the quota refund")
	require.NoError(t, DB.Where("request_id = ?", "refund-atomic").First(&record).Error)
	require.Equal(t, "consumed", record.Status)
	// The test DB has one connection: nested independent transactions would hang.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- RefundSubscriptionPreConsume("refund-atomic") }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.NoError(t, DB.First(&sub, 771001).Error)
	require.EqualValues(t, 100, sub.AmountUsed, "repeated refund must not return quota twice")
	require.NoError(t, DB.Where("request_id = ?", "refund-atomic").First(&record).Error)
	require.Equal(t, "refunded", record.Status)
}
