package model

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestVerificationProofDatabaseMatrix(t *testing.T) {
	for _, tc := range []struct {
		name, env string
	}{{"mysql", "TEST_MYSQL_DSN"}, {"postgres", "TEST_POSTGRES_DSN"}} {
		t.Run(tc.name, func(t *testing.T) {
			dsn := os.Getenv(tc.env)
			if dsn == "" {
				t.Skip("disposable database DSN not configured")
			}
			var dialect gorm.Dialector = mysql.Open(dsn)
			if tc.name == "postgres" {
				dialect = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
			}
			db, err := gorm.Open(dialect, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			previous := DB
			DB = db
			t.Cleanup(func() { DB = previous; _ = sqlDB.Close() })
			TestVerificationProofScopeExpiryAndSingleUse(t)
		})
	}
}

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
