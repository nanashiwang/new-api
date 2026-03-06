package controller

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// resolveSubscriptionPurchaseModeAndTarget 统一解析购买模式，并在续费模式下定位续费目标。
// 约束：续费仅允许同 plan_id 的生效订阅，默认命中最早到期那条。
func resolveSubscriptionPurchaseModeAndTarget(userId int, planId int, rawMode string, rawTargetSubId int) (string, int, error) {
	modeTrimmed := strings.TrimSpace(rawMode)
	if modeTrimmed != "" &&
		modeTrimmed != model.SubscriptionPurchaseModeStack &&
		modeTrimmed != model.SubscriptionPurchaseModeRenew &&
		modeTrimmed != model.SubscriptionPurchaseModeRenewExtend {
		return "", 0, errors.New("无效的购买方式")
	}

	mode := model.NormalizeSubscriptionPurchaseMode(modeTrimmed)
	if mode != model.SubscriptionPurchaseModeRenew {
		return mode, 0, nil
	}

	activeSubs, err := model.GetActiveUserSubscriptionsByPlan(userId, planId)
	if err != nil {
		return "", 0, err
	}
	if len(activeSubs) == 0 {
		return "", 0, errors.New("无可续费的同规格订阅")
	}
	if len(activeSubs) == 1 {
		return mode, activeSubs[0].Id, nil
	}
	if rawTargetSubId > 0 {
		// 多条可续费订阅时，仅允许命中“同用户、同套餐、仍生效”的目标。
		for i := range activeSubs {
			if activeSubs[i].Id == rawTargetSubId {
				return mode, rawTargetSubId, nil
			}
		}
		return "", 0, errors.New("续费目标订阅无效或已失效")
	}
	// 兼容旧客户端：未传目标时默认续费最早到期的订阅。
	return mode, activeSubs[0].Id, nil
}

// shouldCheckSubscriptionPurchaseLimit 控制“续费是否受购买上限约束”。
func shouldCheckSubscriptionPurchaseLimit(mode string) bool {
	if mode == model.SubscriptionPurchaseModeRenew && !common.SubscriptionRenewRespectPurchaseLimit {
		return false
	}
	return true
}

// getSubscriptionPurchaseQuantityConfig 读取套餐级购买数量配置，并做兜底归一化。
func getSubscriptionPurchaseQuantityConfig(plan *model.SubscriptionPlan) (int, int) {
	minQuantity := 1
	maxQuantity := 12
	if plan != nil {
		minQuantity = plan.PurchaseQuantityMin
		maxQuantity = plan.PurchaseQuantityMax
	}
	if minQuantity < 1 {
		minQuantity = 1
	}
	if maxQuantity < 1 {
		maxQuantity = 12
	}
	if maxQuantity < minQuantity {
		maxQuantity = minQuantity
	}
	return minQuantity, maxQuantity
}

// getSubscriptionPurchaseQuantityRangeForUser 读取用户在该套餐下可购买的动态数量范围。
// 规则：
// - 最小值：套餐配置的最小购买数量；
// - 最大值：套餐配置最大购买数量 - 当前未过期份数（按套餐周期份数计算）。
func getSubscriptionPurchaseQuantityRangeForUser(userId int, plan *model.SubscriptionPlan) (int, int, error) {
	minQuantity, configuredMaxQuantity := getSubscriptionPurchaseQuantityConfig(plan)
	if userId <= 0 || plan == nil || plan.Id <= 0 {
		return minQuantity, configuredMaxQuantity, nil
	}
	activeQuantity, err := model.CountUserActiveSubscriptionQuantityByPlan(userId, plan)
	if err != nil {
		return 0, 0, err
	}
	dynamicMaxQuantity := configuredMaxQuantity - activeQuantity
	if dynamicMaxQuantity < 0 {
		dynamicMaxQuantity = 0
	}
	return minQuantity, dynamicMaxQuantity, nil
}

// normalizeSubscriptionPurchaseQuantity 规范化购买数量，并校验套餐级最小值/最大值。
func normalizeSubscriptionPurchaseQuantity(userId int, quantity int, plan *model.SubscriptionPlan) (int, error) {
	minQuantity, maxQuantity, err := getSubscriptionPurchaseQuantityRangeForUser(userId, plan)
	if err != nil {
		return 0, err
	}
	if maxQuantity <= 0 {
		return 0, errors.New("当前可购买数量为 0，请等待部分订阅到期后再试")
	}
	if maxQuantity < minQuantity {
		return 0, fmt.Errorf("当前最多可购买 %d 份，低于最小购买数量 %d", maxQuantity, minQuantity)
	}
	if quantity <= 0 {
		return minQuantity, nil
	}
	if quantity < minQuantity {
		return 0, errors.New("购买数量低于下限")
	}
	if quantity > maxQuantity {
		return 0, fmt.Errorf("购买数量超出上限，当前最多可购买 %d 份", maxQuantity)
	}
	return quantity, nil
}

// checkSubscriptionOrderLimits 统一校验订阅订单限制：
// - 购买上限：限制历史总购买条数（可配置是否影响续费）
// - 叠加上限：限制 stack / renew_extend 的当前生效订阅数
func checkSubscriptionOrderLimits(userId int, plan *model.SubscriptionPlan, purchaseMode string, purchaseQuantity int) error {
	if userId <= 0 || plan == nil {
		return errors.New("参数错误")
	}
	if purchaseQuantity <= 0 {
		purchaseQuantity = 1
	}
	if plan.MaxPurchasePerUser > 0 && shouldCheckSubscriptionPurchaseLimit(purchaseMode) {
		totalCount, err := model.CountUserSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			return err
		}
		limit := int64(plan.MaxPurchasePerUser)
		if ((purchaseMode == model.SubscriptionPurchaseModeStack || purchaseMode == model.SubscriptionPurchaseModeRenewExtend) &&
			totalCount+int64(purchaseQuantity) > limit) ||
			(purchaseMode == model.SubscriptionPurchaseModeRenew && totalCount >= limit) {
			return errors.New("已达到该套餐购买上限")
		}
	}
	if (purchaseMode == model.SubscriptionPurchaseModeStack || purchaseMode == model.SubscriptionPurchaseModeRenewExtend) && plan.MaxStackPerUser > 0 {
		activeCount, err := model.CountUserActiveSubscriptionsByPlan(userId, plan.Id)
		if err != nil {
			return err
		}
		addedActiveCount := int64(purchaseQuantity)
		if purchaseMode == model.SubscriptionPurchaseModeRenewExtend {
			addedActiveCount = 0
		}
		// renew_extend 无生效订阅时会创建 1 条，再按同条顺延。
		if purchaseMode == model.SubscriptionPurchaseModeRenewExtend && activeCount == 0 && purchaseQuantity > 0 {
			addedActiveCount = 1
		}
		if activeCount+addedActiveCount > int64(plan.MaxStackPerUser) {
			return errors.New("已达到该套餐叠加上限")
		}
	}
	return nil
}
