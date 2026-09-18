package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

var ErrInsufficientImageQuota = errors.New("insufficient wallet quota for image request")

// ReserveImageWalletQuota is an immediate conditional write on all supported
// databases. Image requests must be funded before upstream work starts.
func ReserveImageWalletQuota(id, amount int) error {
	if amount < 0 {
		return errors.New("quota cannot be negative")
	}
	if amount == 0 {
		return nil
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := InvalidatePulsePaidFundingTx(tx, id, "unattributed_image_reserve"); err != nil {
			return err
		}
		result := tx.Model(&User{}).Where("id = ? AND quota >= ?", id, amount).Update("quota", gorm.Expr("quota - ?", amount))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrInsufficientImageQuota
		}
		return nil
	})
	if err != nil {
		return err
	}
	gopool.Go(func() {
		if err := cacheDecrUserQuota(id, int64(amount)); err != nil {
			common.SysLog("failed to update reserved image quota cache: " + err.Error())
		}
	})
	return nil
}
