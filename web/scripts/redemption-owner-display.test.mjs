import assert from 'node:assert/strict';
import { getRedemptionDisplayName } from '../src/components/table/redemptions/redemptionDisplay.js';

const t = (key) => (key === '用户 ID' ? 'User ID' : key);

assert.equal(
  getRedemptionDisplayName(
    { name: '用户钱包兑换码', funding_source: 'wallet', user_id: 5703 },
    t,
  ),
  '用户钱包兑换码（User ID: 5703）',
);

assert.equal(
  getRedemptionDisplayName(
    { name: '管理员兑换码', funding_source: 'admin', user_id: 1 },
    t,
  ),
  '管理员兑换码',
);

assert.equal(
  getRedemptionDisplayName(
    { name: '用户钱包兑换码', funding_source: 'wallet', user_id: 0 },
    t,
  ),
  '用户钱包兑换码',
);

console.log('redemption owner display tests passed');
