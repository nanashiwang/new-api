package model

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// External DSNs must name disposable databases. No production migration is run.
func TestImageWalletReserveDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open(":memory:")
			case "mysql":
				dsn := os.Getenv("TEST_COMPAT_MYSQL_DSN")
				if dsn == "" {
					t.Skip("disposable MySQL DSN not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_COMPAT_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("disposable PostgreSQL DSN not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			if dialect == "sqlite" {
				sqlDB.SetMaxOpenConns(1)
			}
			oldDB, oldSQLite, oldMySQL, oldPostgres := DB, common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL
			oldRedis, oldBatch := common.RedisEnabled, common.BatchUpdateEnabled
			DB = db
			common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = dialect == "sqlite", dialect == "mysql", dialect == "postgres"
			common.RedisEnabled, common.BatchUpdateEnabled = false, true
			initCol()
			t.Cleanup(func() {
				DB = oldDB
				common.UsingSQLite, common.UsingMySQL, common.UsingPostgreSQL = oldSQLite, oldMySQL, oldPostgres
				common.RedisEnabled, common.BatchUpdateEnabled = oldRedis, oldBatch
				initCol()
				_ = sqlDB.Close()
			})
			require.NoError(t, db.AutoMigrate(&User{}))
			user := &User{Id: 91301, Username: "compat-image-reserve", AffCode: "cp91301", Quota: 100}
			require.NoError(t, db.Create(user).Error)
			t.Cleanup(func() { require.NoError(t, db.Unscoped().Delete(&User{}, user.Id).Error) })
			require.NoError(t, ReserveImageWalletQuota(user.Id, 40))
			require.ErrorIs(t, ReserveImageWalletQuota(user.Id, 70), ErrInsufficientImageQuota)
			var quota int
			require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).Pluck("quota", &quota).Error)
			require.Equal(t, 60, quota) // immediate even when general batch updates are on
			require.NoError(t, IncreaseUserQuota(user.Id, 40, true))
			require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).Pluck("quota", &quota).Error)
			require.Equal(t, 100, quota)
			require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).Update("quota", 5).Error)
			var wg sync.WaitGroup
			var successes atomic.Int32
			for range 20 {
				wg.Go(func() {
					if ReserveImageWalletQuota(user.Id, 1) == nil {
						successes.Add(1)
					}
				})
			}
			wg.Wait()
			require.EqualValues(t, 5, successes.Load())
			require.NoError(t, db.Model(&User{}).Where("id = ?", user.Id).Pluck("quota", &quota).Error)
			require.Zero(t, quota)
		})
	}
}
