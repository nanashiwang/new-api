package model

import (
	"errors"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WalletTransferLock serializes every create, redeem, and reversal operation
// for wallet-funded redemption codes owned by the same creator. A dedicated
// row avoids lock-order cycles between two users who redeem each other's codes.
type WalletTransferLock struct {
	UserId    int   `json:"-" gorm:"primaryKey;autoIncrement:false;column:user_id"`
	CreatedAt int64 `json:"-" gorm:"bigint"`
}

func lockWalletTransferTx(tx *gorm.DB, userID int) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	lockRow := &WalletTransferLock{UserId: userID, CreatedAt: common.GetTimestamp()}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(lockRow).Error; err != nil {
		return err
	}
	query := tx.Where("user_id = ?", userID)
	if !common.UsingSQLite {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return query.First(&WalletTransferLock{}).Error
}

// EffectiveTransferableQuota returns the verified paid portion that is still
// backed by the user's current wallet balance. The stored value may temporarily
// exceed Quota after ordinary consumption; every grant and transfer operation
// normalizes it before creating new transferable value.
func EffectiveTransferableQuota(quota int, transferableQuota int) int {
	if quota <= 0 || transferableQuota <= 0 {
		return 0
	}
	if transferableQuota > quota {
		return quota
	}
	return transferableQuota
}

// GrantUserQuotaTx grants wallet quota and explicitly declares how much of the
// grant is eligible to fund a user-created redemption code.
func GrantUserQuotaTx(tx *gorm.DB, userID int, quota int, transferableQuota int) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	if quota < 0 || transferableQuota < 0 || transferableQuota > quota {
		return errors.New("invalid quota grant")
	}
	if quota == 0 {
		return nil
	}
	// Update transferability first. This order matters on MySQL, where SET
	// expressions can observe columns assigned earlier in the same statement.
	// Separate statements inside one transaction preserve the old quota for
	// normalization while the first update also acquires the user row lock.
	effectiveExpr := "CASE WHEN transferable_quota <= 0 OR quota <= 0 THEN 0 WHEN transferable_quota > quota THEN quota ELSE transferable_quota END"
	if err := tx.Model(&User{}).Where("id = ?", userID).
		UpdateColumn("transferable_quota", gorm.Expr("("+effectiveExpr+") + ?", transferableQuota)).Error; err != nil {
		return err
	}
	result := tx.Model(&User{}).Where("id = ?", userID).
		UpdateColumn("quota", gorm.Expr("quota + ?", quota))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func GrantUserQuota(userID int, quota int, transferableQuota int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return GrantUserQuotaTx(tx, userID, quota, transferableQuota)
	})
}

// RevokeTransferableQuotaGrantTx reverses a paid wallet grant across every
// place where the creator can move that value: their wallet, active codes, and
// recipients of already-used codes. Wallet codes are one-hop transfers, so the
// existing redemption row is sufficient to identify the final recipient.
func RevokeTransferableQuotaGrantTx(tx *gorm.DB, userID int, quota int) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	if quota <= 0 {
		return nil
	}
	if err := lockWalletTransferTx(tx, userID); err != nil {
		return err
	}

	var redemptions []*Redemption
	redemptionQuery := tx.Unscoped().
		Where("user_id = ? AND funding_source = ? AND transferable_quota > ?", userID, RedemptionFundingSourceWallet, 0).
		Order("id ASC")
	if !common.UsingSQLite {
		redemptionQuery = redemptionQuery.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := redemptionQuery.Find(&redemptions).Error; err != nil {
		return err
	}

	userIDs := map[int]struct{}{userID: {}}
	for _, redemption := range redemptions {
		if redemption == nil || redemption.Status != common.RedemptionCodeStatusUsed ||
			redemption.UsedUserId <= 0 || redemption.UsedUserId == userID {
			continue
		}
		userIDs[redemption.UsedUserId] = struct{}{}
	}
	orderedUserIDs := make([]int, 0, len(userIDs))
	for id := range userIDs {
		orderedUserIDs = append(orderedUserIDs, id)
	}
	sort.Ints(orderedUserIDs)

	var users []*User
	userQuery := tx.Unscoped().
		Select("id", "quota", "transferable_quota").
		Where("id IN ?", orderedUserIDs).
		Order("id ASC")
	if !common.UsingSQLite {
		userQuery = userQuery.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := userQuery.Find(&users).Error; err != nil {
		return err
	}
	userByID := make(map[int]*User, len(users))
	for _, user := range users {
		if user != nil {
			userByID[user.Id] = user
		}
	}
	creator := userByID[userID]
	if creator == nil {
		return gorm.ErrRecordNotFound
	}

	remaining := quota
	creatorTransferable := EffectiveTransferableQuota(creator.Quota, creator.TransferableQuota)
	walletDebit := minInt(remaining, creatorTransferable)
	creator.Quota -= walletDebit
	creator.TransferableQuota = creatorTransferable - walletDebit
	remaining -= walletDebit
	creatorDebt := 0

	for _, redemption := range redemptions {
		if remaining <= 0 {
			break
		}
		if redemption == nil || redemption.UserId != userID {
			continue
		}
		available := redemption.TransferableQuota
		if available <= 0 {
			continue
		}
		if redemption.Quota > 0 && available > redemption.Quota {
			available = redemption.Quota
		}
		debit := minInt(remaining, available)
		if debit <= 0 {
			continue
		}

		updates := map[string]any{"transferable_quota": available - debit}
		switch redemption.Status {
		case common.RedemptionCodeStatusEnabled:
			newQuota := redemption.Quota - debit
			if newQuota < 0 {
				newQuota = 0
			}
			updates["quota"] = newQuota
			if newQuota == 0 {
				updates["status"] = common.RedemptionCodeStatusDisabled
			}
		case common.RedemptionCodeStatusUsed:
			if redemption.UsedUserId == userID {
				// A self-redeemed code already returned to the creator wallet and
				// must not be reclaimed a second time.
				continue
			}
			recipient := userByID[redemption.UsedUserId]
			if recipient == nil {
				// The transfer record is consumed so a later reversal cannot
				// target it again, but the missing recipient leaves the creator
				// liable for the unrecovered amount below.
				creatorDebt += debit
			} else {
				recipient.Quota -= debit
				recipient.TransferableQuota = EffectiveTransferableQuota(recipient.Quota, recipient.TransferableQuota)
			}
		default:
			continue
		}

		if err := tx.Unscoped().Model(&Redemption{}).
			Where("id = ?", redemption.Id).
			Updates(updates).Error; err != nil {
			return err
		}
		remaining -= debit
	}

	// Any amount no longer present in a wallet or a code was already consumed,
	// so the creator retains the debt. A negative balance is intentional: it
	// prevents a throwaway payer account from externalizing a chargeback loss.
	if remaining > 0 {
		creatorDebt += remaining
	}
	if creatorDebt > 0 {
		creator.Quota -= creatorDebt
		creator.TransferableQuota = EffectiveTransferableQuota(creator.Quota, creator.TransferableQuota)
	}

	for _, id := range orderedUserIDs {
		user := userByID[id]
		if user == nil {
			continue
		}
		if err := tx.Unscoped().Model(&User{}).Where("id = ?", id).Updates(map[string]any{
			"quota":              user.Quota,
			"transferable_quota": EffectiveTransferableQuota(user.Quota, user.TransferableQuota),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
