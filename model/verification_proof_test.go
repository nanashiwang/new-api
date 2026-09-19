package model

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestVerificationProofScopeExpiryAndSingleUse(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&VerificationProof{}))
	token, err := IssueVerificationProof(42, "test:ready", time.Minute)
	require.NoError(t, err)
	for _, scope := range []struct {
		user    int
		purpose string
	}{{43, "test:ready"}, {42, "test:other"}} {
		valid, err := ConsumeVerificationProof(token, scope.user, scope.purpose)
		require.NoError(t, err)
		require.False(t, valid)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			valid, err := ConsumeVerificationProof(token, 42, "test:ready")
			if err != nil {
				t.Error(err)
			}
			if valid {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, successes.Load())
	token, err = IssueVerificationProof(42, "test:ready", time.Minute)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&VerificationProof{}).Where("digest = ?", verificationProofDigest(token)).Update("expires_at", time.Now().Unix()).Error)
	valid, err := ConsumeVerificationProof(token, 42, "test:ready")
	require.NoError(t, err)
	require.False(t, valid)
}
