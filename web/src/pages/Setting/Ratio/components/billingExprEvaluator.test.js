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
import { evalBillingExprLocally } from './billingExprEvaluator';

const expression = `
  ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) ||
    (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))
    ? tier("峰时", p * 15 + c * 45 + cr * 0.5)
    : tier("谷时", p * 7.5 + c * 22.5 + cr * 0.25)
`;

const tokens = {
  cacheReadTokens: 0,
  cacheCreateTokens: 0,
  cacheCreate1hTokens: 0,
  cr: 100,
};

describe('billing expression local evaluator', () => {
  test('evaluates Shanghai peak hours', () => {
    const result = evalBillingExprLocally(
      expression,
      1000,
      200,
      tokens,
      new Date('2026-08-22T02:00:00Z'),
    );

    expect(result.error).toBeNull();
    expect(result.matchedTier).toBe('峰时');
    expect(result.cost).toBe(24050);
  });

  test('evaluates Shanghai off-peak hours', () => {
    const result = evalBillingExprLocally(
      expression,
      1000,
      200,
      tokens,
      new Date('2026-08-22T05:00:00Z'),
    );

    expect(result.error).toBeNull();
    expect(result.matchedTier).toBe('谷时');
    expect(result.cost).toBe(12025);
  });

  test('supports all backend time helpers and falls back to UTC', () => {
    const result = evalBillingExprLocally(
      `hour("Invalid/Zone") == 2 && minute("UTC") == 30 && weekday("UTC") == 6 && month("UTC") == 8 && day("UTC") == 22
        ? tier("matched", p)
        : tier("missed", c)`,
      10,
      20,
      tokens,
      new Date('2026-08-22T02:30:00Z'),
    );

    expect(result.error).toBeNull();
    expect(result.matchedTier).toBe('matched');
    expect(result.cost).toBe(10);
  });

  test('request helpers do not break preview without request data', () => {
    const result = evalBillingExprLocally(
      `param("service_tier") != nil || header("x-plan") != "" || has(header("x-name"), "pro")
        ? tier("request", p)
        : tier("default", c)`,
      10,
      20,
      tokens,
      new Date('2026-08-22T02:30:00Z'),
    );

    expect(result.error).toBeNull();
    expect(result.matchedTier).toBe('default');
    expect(result.cost).toBe(20);
  });
});
