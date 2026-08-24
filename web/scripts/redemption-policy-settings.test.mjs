import assert from 'node:assert/strict';
import {
  resolveRedemptionLimitInputs,
  validateRedemptionPolicyInputs,
} from '../src/pages/Setting/RateLimit/redemptionPolicy.js';

const defaults = resolveRedemptionLimitInputs({});
assert.equal(defaults.WalletRedemptionDailyCreateLimit, 100);
assert.equal(defaults.WalletRedemptionMinimumQuota, 10);
assert.equal(defaults.WalletRedemptionActiveLimit, 100);
assert.equal(defaults.WalletRedemptionDailyQuotaLimit, 5000);
assert.equal(defaults.WalletRedemptionReviewDistinctCreatorThreshold, 3);
assert.equal(defaults.WalletRedemptionReviewSmallQuotaLimit, 100);

const partial = resolveRedemptionLimitInputs({
  WalletRedemptionDailyCreateLimit: '80',
});
assert.equal(partial.WalletRedemptionDailyCreateLimit, '80');
assert.equal(partial.WalletRedemptionDailyQuotaLimit, 5000);

assert.equal(validateRedemptionPolicyInputs(defaults).errorKey, '');
assert.equal(
  validateRedemptionPolicyInputs({
    ...defaults,
    WalletRedemptionReviewDistinctCreatorThreshold: 1,
  }).errorKey,
  '人工复查不同创建人数必须为 0 或至少 2',
);
assert.equal(
  validateRedemptionPolicyInputs({
    ...defaults,
    WalletRedemptionMinimumQuota: 101,
    WalletRedemptionReviewSmallQuotaLimit: 100,
  }).errorKey,
  '人工复查小额上限不能低于单个兑换码最低额度',
);

console.log('redemption policy settings tests passed');
