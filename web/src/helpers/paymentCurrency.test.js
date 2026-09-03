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
  createPaymentQuote,
  formatPaymentMoney,
  getLegacyPaymentCurrency,
  normalizePaymentCurrency,
} from './paymentCurrency';

describe('payment currency', () => {
  test('normalizes ISO currency codes', () => {
    expect(normalizePaymentCurrency(' usd ')).toBe('USD');
    expect(normalizePaymentCurrency('RMB')).toBe('CNY');
    expect(normalizePaymentCurrency('US')).toBe('');
  });

  test('formats CNY and USD with their own symbols', () => {
    expect(formatPaymentMoney(0.2, 'CNY', 'zh-CN')).toBe('¥0.20');
    expect(formatPaymentMoney(0.2, 'USD', 'zh-CN')).toBe('US$0.20');
  });

  test('keeps compatibility with legacy quote responses', () => {
    expect(getLegacyPaymentCurrency('stripe')).toBe('USD');
    expect(getLegacyPaymentCurrency('alipay')).toBe('CNY');
    expect(createPaymentQuote('0.20', undefined, 'stripe')).toEqual({
      amount: 0.2,
      currency: 'USD',
      estimated: true,
    });
  });

  test('does not silently replace an explicitly invalid currency', () => {
    expect(createPaymentQuote('0.20', 'US', 'stripe')).toEqual({
      amount: 0.2,
      currency: '',
      estimated: true,
    });
  });
});
