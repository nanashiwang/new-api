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

import { describe, expect, test } from 'bun:test';
import {
  AFF_WITHDRAWAL_MINIMUM_PAYMENT_CNY,
  AFF_WITHDRAWAL_MINIMUM_QUOTA,
  calculateAffWithdrawalPaymentAmount,
  meetsAffWithdrawalMinimum,
} from './withdrawal.constants';

describe('withdrawal minimum', () => {
  test('rejects 249 quota and accepts the exact 250 quota / 50 CNY boundary', () => {
    expect(meetsAffWithdrawalMinimum(249, 100, 20)).toBe(false);
    expect(meetsAffWithdrawalMinimum(250, 100, 20)).toBe(true);
  });

  test('rejects 250 quota when the configured payout is below 50 CNY', () => {
    expect(meetsAffWithdrawalMinimum(250, 100, 19.99)).toBe(false);
  });

  test('uses the same quota and payout constants as the UI', () => {
    expect(AFF_WITHDRAWAL_MINIMUM_QUOTA).toBe(250);
    expect(AFF_WITHDRAWAL_MINIMUM_PAYMENT_CNY).toBe(50);
    expect(calculateAffWithdrawalPaymentAmount(250, 100, 20)).toBe(50);
  });

  test.each([
    [250, 0, 20],
    [250, 100, 0],
    [250, Number.NaN, 20],
    [250, 100, Number.POSITIVE_INFINITY],
  ])(
    'rejects invalid conversion configuration',
    (quota, quotaPerUnit, price) => {
      expect(meetsAffWithdrawalMinimum(quota, quotaPerUnit, price)).toBe(false);
    },
  );
});
