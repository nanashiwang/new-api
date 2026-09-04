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

export const REDEMPTION_LIMIT_DEFAULTS = Object.freeze({
  RedemptionRateLimitEnabled: false,
  RedemptionRateLimitDurationSeconds: 600,
  RedemptionRateLimitSuccessCount: 0,
  RedemptionRateLimitFailureCount: 0,
  WalletRedemptionDailyCreateLimit: 100,
  WalletRedemptionMinimumQuota: 10,
  WalletRedemptionActiveLimit: 100,
  WalletRedemptionDailyQuotaLimit: 5000,
  WalletRedemptionReviewDistinctCreatorThreshold: 3,
  WalletRedemptionReviewSmallQuotaLimit: 100,
  WalletRedemptionAutoBindMaxAgeHours: 24,
});

export function resolveRedemptionLimitInputs(options = {}) {
  const resolved = { ...REDEMPTION_LIMIT_DEFAULTS };
  for (const key of Object.keys(resolved)) {
    const value = options[key];
    if (value !== undefined && value !== null && value !== '') {
      resolved[key] = value;
    }
  }
  return resolved;
}

export function validateRedemptionPolicyInputs(inputs) {
  const values = {
    minimumQuota: Number(inputs.WalletRedemptionMinimumQuota),
    activeLimit: Number(inputs.WalletRedemptionActiveLimit),
    dailyCreateLimit: Number(inputs.WalletRedemptionDailyCreateLimit),
    dailyQuotaLimit: Number(inputs.WalletRedemptionDailyQuotaLimit),
    creatorThreshold: Number(
      inputs.WalletRedemptionReviewDistinctCreatorThreshold,
    ),
    smallQuotaLimit: Number(inputs.WalletRedemptionReviewSmallQuotaLimit),
  };
  if (
    Object.values(values).some((value) => !Number.isInteger(value) || value < 0)
  ) {
    return { errorKey: '兑换码限制必须是非负整数', values };
  }
  if (values.creatorThreshold === 1) {
    return {
      errorKey: '人工复查不同创建人数必须为 0 或至少 2',
      values,
    };
  }
  if (
    values.creatorThreshold > 0 &&
    values.smallQuotaLimit > 0 &&
    values.smallQuotaLimit < values.minimumQuota
  ) {
    return {
      errorKey: '人工复查小额上限不能低于单个兑换码最低额度',
      values,
    };
  }
  return { errorKey: '', values };
}
