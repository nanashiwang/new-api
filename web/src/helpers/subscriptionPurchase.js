/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

// 读取套餐维度的数量限制（min/max）并做安全归一化。
// activeQuantity 表示用户在该套餐下当前未过期份数，用于动态降低 max。
export function getPlanPurchaseQuantityConfig(plan, activeQuantity = 0) {
  const min = Math.max(1, Number(plan?.purchase_quantity_min || 1));
  const configuredMax = Math.max(min, Number(plan?.purchase_quantity_max || 12));
  const dynamicMax = Math.max(
    0,
    configuredMax - Math.max(0, Number(activeQuantity || 0)),
  );
  return { min, max: dynamicMax, configuredMax };
}

export function buildPurchaseQuantityOptions(minQuantity, maxQuantity) {
  const min = Number(minQuantity || 0);
  const max = Number(maxQuantity || 0);
  if (max < min || min <= 0) return [];
  return Array.from({ length: max - min + 1 }, (_, index) => {
    const value = min + index;
    return {
      value,
      label: `${value}`,
    };
  });
}

// 仅返回同套餐的生效订阅，按最早到期排序。
export function getRenewableSubscriptionsByPlan(subscriptions, targetPlanId) {
  const planId = Number(targetPlanId || 0);
  if (planId <= 0) return [];
  const nowUnix = Date.now() / 1000;
  return (subscriptions || [])
    .map((item) => item?.subscription)
    .filter(Boolean)
    .filter((sub) => Number(sub?.plan_id || 0) === planId)
    .filter(
      (sub) => sub?.status === 'active' && Number(sub?.end_time || 0) > nowUnix,
    )
    .sort((a, b) => {
      const endA = Number(a?.end_time || 0);
      const endB = Number(b?.end_time || 0);
      if (endA !== endB) return endA - endB;
      return Number(a?.id || 0) - Number(b?.id || 0);
    });
}
