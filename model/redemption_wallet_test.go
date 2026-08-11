package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEnsureRedemptionColumnsSQLite_AddsWalletFundingColumnsWithLegacyDefault(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE redemptions (id integer primary key, `key` char(32), status integer)").Error)
	require.NoError(t, db.Exec("INSERT INTO redemptions (id, `key`, status) VALUES (1, 'legacy-code', 1)").Error)

	originalDB := DB
	originalUsingSQLite := common.UsingSQLite
	DB = db
	common.UsingSQLite = true
	t.Cleanup(func() {
		DB = originalDB
		common.UsingSQLite = originalUsingSQLite
	})

	require.NoError(t, ensureRedemptionColumnsSQLite())
	assert.True(t, db.Migrator().HasColumn("redemptions", "funding_source"))
	assert.True(t, db.Migrator().HasColumn("redemptions", "create_request_id"))
	var fundingSource string
	require.NoError(t, db.Raw("SELECT funding_source FROM redemptions WHERE id = 1").Scan(&fundingSource).Error)
	assert.Equal(t, RedemptionFundingSourceAdmin, fundingSource)
}

func setupWalletRedemptionTest(t *testing.T) {
	t.Helper()
	setupInviteCommissionSubscriptionTest(t)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&Redemption{}).Error)
}

func setWalletQuota(t *testing.T, userID int, quota int) {
	t.Helper()
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userID).Update("quota", quota).Error)
}

func countInviteCommissionLedgers(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&InviteCommissionLedger{}).Count(&count).Error)
	return count
}

func TestCreateWalletFundedRedemption_DeductsQuotaAndReplaysIdempotently(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_create", 0)
	setWalletQuota(t, creator.Id, 1000)

	first, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-create-request-001")
	require.NoError(t, err)
	require.NotNil(t, first.Redemption)
	assert.False(t, first.Replayed)
	assert.Equal(t, 700, first.RemainingQuota)
	assert.Equal(t, 300, first.Redemption.Quota)

	second, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-create-request-001")
	require.NoError(t, err)
	assert.True(t, second.Replayed)
	assert.Equal(t, first.Redemption.Id, second.Redemption.Id)
	assert.Equal(t, first.Redemption.Key, second.Redemption.Key)
	assert.Equal(t, 700, second.RemainingQuota)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 700, refreshed.Quota)
	var redemptions []Redemption
	require.NoError(t, DB.Find(&redemptions).Error)
	require.Len(t, redemptions, 1)
	assert.Equal(t, RedemptionFundingSourceWallet, redemptions[0].FundingSource)
	assert.NotNil(t, redemptions[0].CreateRequestId)
}

func TestCreateWalletFundedRedemption_RejectsInsufficientQuotaWithoutMutation(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_insufficient", 0)
	setWalletQuota(t, creator.Id, 200)

	_, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-insufficient-001")
	assert.ErrorIs(t, err, ErrRedemptionInsufficientQuota)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 200, refreshed.Quota)
	var count int64
	require.NoError(t, DB.Model(&Redemption{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCreateWalletFundedRedemption_RequestIDIsScopedPerUser(t *testing.T) {
	setupWalletRedemptionTest(t)
	firstCreator := createInviteCommissionTestUser(t, "wallet_request_scope_a", 0)
	secondCreator := createInviteCommissionTestUser(t, "wallet_request_scope_b", 0)
	setWalletQuota(t, firstCreator.Id, 100)
	setWalletQuota(t, secondCreator.Id, 100)

	first, err := CreateWalletFundedRedemption(firstCreator.Id, 100, "shared-wallet-request-001")
	require.NoError(t, err)
	second, err := CreateWalletFundedRedemption(secondCreator.Id, 100, "shared-wallet-request-001")
	require.NoError(t, err)
	assert.NotEqual(t, first.Redemption.Id, second.Redemption.Id)
	assert.NotEqual(t, first.Redemption.Key, second.Redemption.Key)
}

func TestCreateWalletFundedRedemption_EnforcesActiveCodeLimitWithoutDeduction(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_active_limit", 0)
	setWalletQuota(t, creator.Id, 1000)
	for index := 0; index < maxActiveWalletRedemptionsPerUser; index++ {
		requestID := fmt.Sprintf("wallet-active-existing-%03d", index)
		redemption := &Redemption{
			UserId:          creator.Id,
			Key:             common.GetUUID(),
			Status:          common.RedemptionCodeStatusEnabled,
			Name:            "用户钱包兑换码",
			BenefitType:     RedemptionBenefitTypeQuota,
			Quota:           1,
			CreatedTime:     common.GetTimestamp(),
			FundingSource:   RedemptionFundingSourceWallet,
			CreateRequestId: &requestID,
		}
		require.NoError(t, redemption.Insert())
	}

	_, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-active-limit-new-001")
	assert.ErrorIs(t, err, ErrRedemptionActiveLimit)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 1000, refreshed.Quota)
}

func TestWalletRedemption_SelfRedeemOnlyRestoresQuota(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_self", 0)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-self-request-001")
	require.NoError(t, err)

	result, err := RedeemWithResult(created.Redemption.Key, creator.Id)
	require.NoError(t, err)
	assert.Equal(t, 300, result.QuotaAdded)

	var refreshed User
	require.NoError(t, DB.First(&refreshed, creator.Id).Error)
	assert.Equal(t, 1000, refreshed.Quota)
	assert.Zero(t, refreshed.InviterId)
	assert.Zero(t, refreshed.AffCount)
	assert.Zero(t, refreshed.AffQuota)
	assert.Zero(t, refreshed.AffHistoryQuota)
	assert.Zero(t, countInviteCommissionLedgers(t))
}

func TestWalletRedemption_BindsUnboundRedeemerWithoutRewards(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_creator", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_redeemer", 0)
	setWalletQuota(t, creator.Id, 1000)
	setWalletQuota(t, redeemer.Id, 50)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-bind-request-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	var refreshedCreator User
	var refreshedRedeemer User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
	assert.Equal(t, creator.Id, refreshedRedeemer.InviterId)
	assert.Equal(t, 350, refreshedRedeemer.Quota)
	assert.Equal(t, 1, refreshedCreator.AffCount)
	assert.Zero(t, refreshedCreator.AffQuota)
	assert.Zero(t, refreshedCreator.AffHistoryQuota)
	assert.Zero(t, countInviteCommissionLedgers(t))
}

func TestWalletRedemption_DoesNotOverrideExistingInviter(t *testing.T) {
	setupWalletRedemptionTest(t)
	existingInviter := createInviteCommissionTestUser(t, "wallet_existing_inviter", 0)
	creator := createInviteCommissionTestUser(t, "wallet_other_creator", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_already_bound", existingInviter.Id)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 300, "wallet-existing-request-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	var refreshedCreator User
	var refreshedRedeemer User
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
	assert.Equal(t, existingInviter.Id, refreshedRedeemer.InviterId)
	assert.Zero(t, refreshedCreator.AffCount)
}

func TestWalletRedemption_DoesNotCreateInviteCycle(t *testing.T) {
	setupWalletRedemptionTest(t)
	redeemer := createInviteCommissionTestUser(t, "wallet_cycle_parent", 0)
	creator := createInviteCommissionTestUser(t, "wallet_cycle_child", redeemer.Id)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-cycle-request-001")
	require.NoError(t, err)

	_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	var refreshedRedeemer User
	var refreshedCreator User
	require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
	require.NoError(t, DB.First(&refreshedCreator, creator.Id).Error)
	assert.Zero(t, refreshedRedeemer.InviterId)
	assert.Equal(t, redeemer.Id, refreshedCreator.InviterId)
	assert.Zero(t, refreshedCreator.AffCount)
}

func TestWalletRedemption_SkipsBindingForUnavailableCreator(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(t *testing.T, creator *User)
		requestID string
	}{
		{
			name: "disabled",
			mutate: func(t *testing.T, creator *User) {
				require.NoError(t, DB.Model(&User{}).Where("id = ?", creator.Id).Update("status", common.UserStatusDisabled).Error)
			},
			requestID: "wallet-disabled-request-001",
		},
		{
			name: "deleted",
			mutate: func(t *testing.T, creator *User) {
				require.NoError(t, DB.Delete(&User{}, creator.Id).Error)
			},
			requestID: "wallet-deleted-request-001",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setupWalletRedemptionTest(t)
			creator := createInviteCommissionTestUser(t, "wallet_unavailable_creator_"+test.name, 0)
			redeemer := createInviteCommissionTestUser(t, "wallet_unavailable_redeemer_"+test.name, 0)
			setWalletQuota(t, creator.Id, 1000)
			created, err := CreateWalletFundedRedemption(creator.Id, 100, test.requestID)
			require.NoError(t, err)
			test.mutate(t, creator)

			_, err = RedeemWithResult(created.Redemption.Key, redeemer.Id)
			require.NoError(t, err)

			var refreshedRedeemer User
			require.NoError(t, DB.First(&refreshedRedeemer, redeemer.Id).Error)
			assert.Equal(t, 100, refreshedRedeemer.Quota)
			assert.Zero(t, refreshedRedeemer.InviterId)
		})
	}
}

func TestAdminRedemption_DoesNotBindRedeemer(t *testing.T) {
	setupWalletRedemptionTest(t)
	admin := createInviteCommissionTestUser(t, "redemption_admin_source", 0)
	redeemer := createInviteCommissionTestUser(t, "redemption_admin_redeemer", 0)
	redemption := &Redemption{
		UserId:        admin.Id,
		Key:           common.GetUUID(),
		Status:        common.RedemptionCodeStatusEnabled,
		Name:          "管理员余额兑换码",
		BenefitType:   RedemptionBenefitTypeQuota,
		Quota:         100,
		CreatedTime:   common.GetTimestamp(),
		FundingSource: RedemptionFundingSourceAdmin,
	}
	require.NoError(t, redemption.Insert())

	_, err := RedeemWithResult(redemption.Key, redeemer.Id)
	require.NoError(t, err)
	var refreshed User
	require.NoError(t, DB.First(&refreshed, redeemer.Id).Error)
	assert.Zero(t, refreshed.InviterId)
}

func TestWalletRedemption_AdminMutationAndCleanupAreBlocked(t *testing.T) {
	setupWalletRedemptionTest(t)
	creator := createInviteCommissionTestUser(t, "wallet_immutable", 0)
	setWalletQuota(t, creator.Id, 1000)
	created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-immutable-request-001")
	require.NoError(t, err)

	redemption, err := GetRedemptionById(created.Redemption.Id)
	require.NoError(t, err)
	redemption.Name = "tampered"
	assert.ErrorIs(t, redemption.Update(), ErrWalletFundedRedemptionImmutable)
	assert.ErrorIs(t, (&Redemption{Id: redemption.Id, FundingSource: RedemptionFundingSourceAdmin}).Update(), ErrWalletFundedRedemptionImmutable)
	assert.ErrorIs(t, DeleteRedemptionById(redemption.Id), ErrWalletFundedRedemptionImmutable)

	redemption.Status = common.RedemptionCodeStatusUsed
	require.NoError(t, DB.Model(&Redemption{}).Where("id = ?", redemption.Id).Updates(map[string]any{
		"status":        redemption.Status,
		"redeemed_time": common.GetTimestamp(),
	}).Error)
	adminCode := &Redemption{
		UserId:        creator.Id,
		Key:           common.GetUUID(),
		Status:        common.RedemptionCodeStatusUsed,
		Name:          "admin cleanup code",
		BenefitType:   RedemptionBenefitTypeQuota,
		CreatedTime:   common.GetTimestamp(),
		FundingSource: RedemptionFundingSourceAdmin,
	}
	require.NoError(t, adminCode.Insert())

	deleted, err := DeleteInvalidRedemptions()
	require.NoError(t, err)
	assert.EqualValues(t, 1, deleted)
	require.NoError(t, DB.First(&Redemption{}, redemption.Id).Error)
	assert.ErrorIs(t, DB.First(&Redemption{}, adminCode.Id).Error, gorm.ErrRecordNotFound)
}

func TestGetUserWalletRedemptions_IsOwnerScopedAndPrivacySafe(t *testing.T) {
	setupWalletRedemptionTest(t)
	creatorA := createInviteCommissionTestUser(t, "wallet_list_a", 0)
	creatorB := createInviteCommissionTestUser(t, "wallet_list_b", 0)
	redeemer := createInviteCommissionTestUser(t, "wallet_list_redeemer", 0)
	setWalletQuota(t, creatorA.Id, 1000)
	setWalletQuota(t, creatorB.Id, 1000)
	createdA, err := CreateWalletFundedRedemption(creatorA.Id, 100, "wallet-list-a-request-001")
	require.NoError(t, err)
	_, err = CreateWalletFundedRedemption(creatorB.Id, 200, "wallet-list-b-request-001")
	require.NoError(t, err)
	_, err = RedeemWithResult(createdA.Redemption.Key, redeemer.Id)
	require.NoError(t, err)

	items, total, err := GetUserWalletRedemptions(creatorA.Id, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, createdA.Redemption.Key, items[0].Key)
	payload, err := common.Marshal(items)
	require.NoError(t, err)
	jsonText := string(payload)
	assert.False(t, strings.Contains(jsonText, "used_user_id"), jsonText)
	assert.False(t, strings.Contains(jsonText, redeemer.Username), jsonText)
}

func TestWalletRedemption_ConcurrentUseAndCreationDoNotDuplicateOrOverdraw(t *testing.T) {
	t.Run("same code can only be redeemed once", func(t *testing.T) {
		setupWalletRedemptionTest(t)
		creator := createInviteCommissionTestUser(t, "wallet_concurrent_creator", 0)
		firstRedeemer := createInviteCommissionTestUser(t, "wallet_concurrent_first", 0)
		secondRedeemer := createInviteCommissionTestUser(t, "wallet_concurrent_second", 0)
		setWalletQuota(t, creator.Id, 1000)
		created, err := CreateWalletFundedRedemption(creator.Id, 100, "wallet-concurrent-code-001")
		require.NoError(t, err)

		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for _, userID := range []int{firstRedeemer.Id, secondRedeemer.Id} {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, redeemErr := RedeemWithResult(created.Redemption.Key, id)
				errs <- redeemErr
			}(userID)
		}
		wg.Wait()
		close(errs)
		successes := 0
		alreadyUsed := 0
		for redeemErr := range errs {
			switch {
			case redeemErr == nil:
				successes++
			case errors.Is(redeemErr, ErrRedemptionAlreadyUsed):
				alreadyUsed++
			default:
				require.NoError(t, redeemErr)
			}
		}
		assert.Equal(t, 1, successes)
		assert.Equal(t, 1, alreadyUsed)
	})

	t.Run("wallet cannot be overdrawn", func(t *testing.T) {
		setupWalletRedemptionTest(t)
		creator := createInviteCommissionTestUser(t, "wallet_concurrent_balance", 0)
		setWalletQuota(t, creator.Id, 100)

		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				_, createErr := CreateWalletFundedRedemption(creator.Id, 100, fmt.Sprintf("wallet-overdraw-request-%03d", index))
				errs <- createErr
			}(i)
		}
		wg.Wait()
		close(errs)
		successes := 0
		insufficient := 0
		for createErr := range errs {
			switch {
			case createErr == nil:
				successes++
			case errors.Is(createErr, ErrRedemptionInsufficientQuota):
				insufficient++
			default:
				require.NoError(t, createErr)
			}
		}
		assert.Equal(t, 1, successes)
		assert.Equal(t, 1, insufficient)
		var refreshed User
		require.NoError(t, DB.First(&refreshed, creator.Id).Error)
		assert.Zero(t, refreshed.Quota)
	})
}
