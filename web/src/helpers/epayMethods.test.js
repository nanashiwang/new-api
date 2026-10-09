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
  EPAY_USDT_TRC20,
  isEpayUsdtEnabled,
  parseEpayMethods,
  setEpayUsdtEnabled,
} from './epayMethods';

describe('Epay USDT configuration', () => {
  test('enabling preserves unrelated methods and does not duplicate USDT', () => {
    const original = [
      { type: 'alipay', name: 'Custom Alipay', min_topup: '12' },
    ];
    const enabled = setEpayUsdtEnabled(original, true);
    expect(original).toHaveLength(1);
    expect(parseEpayMethods(enabled)[0]).toEqual(original[0]);
    expect(parseEpayMethods(enabled)[1].type).toBe('usdt.trc20');
    expect(isEpayUsdtEnabled(enabled)).toBe(true);
    expect(setEpayUsdtEnabled(enabled, true)).toBe(enabled);
  });

  test('recognizes existing custom USDT settings and removes only that type', () => {
    const methods = [
      { type: 'stripe', name: 'Stripe' },
      {
        type: EPAY_USDT_TRC20,
        name: 'My wallet',
        color: 'green',
        min_topup: '25',
      },
      { type: 'custom1', name: 'Other' },
    ];
    expect(parseEpayMethods(setEpayUsdtEnabled(methods, true))).toEqual(
      methods,
    );
    expect(parseEpayMethods(setEpayUsdtEnabled(methods, false))).toEqual([
      methods[0],
      methods[2],
    ]);
    expect(isEpayUsdtEnabled('[{"type":"usdt"}]')).toBe(false);
  });

  test('empty configuration starts disabled', () => {
    expect(isEpayUsdtEnabled('')).toBe(false);
    expect(isEpayUsdtEnabled(setEpayUsdtEnabled('', true))).toBe(true);
    expect(parseEpayMethods(setEpayUsdtEnabled('', false))).toEqual([]);
  });

  test.each([
    '{',
    '{}',
    'null',
    '[null]',
    '[[]]',
    '[{"type":1}]',
    '[{"type":"alipay","min_topup":2}]',
  ])('invalid configuration is never replaced silently: %s', (value) => {
    expect(isEpayUsdtEnabled(value)).toBe(false);
    expect(() => setEpayUsdtEnabled(value, true)).toThrow();
    expect(() => setEpayUsdtEnabled(value, false)).toThrow();
  });
});
