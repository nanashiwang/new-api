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
  usagePresetRange,
  validUsageRange,
  formatUsageTime,
  parseUsageTime,
  initialChannelUsage,
  sumChannelUsage,
  channelUsageLogURL,
} from './channelUsage';

describe('channel usage accounting windows', () => {
  const now = Date.parse('2026-09-29T12:00:00+08:00') / 1000;
  test('Beijing dates and exclusive yesterday end', () => {
    const today = usagePresetRange('today', now);
    const yesterday = usagePresetRange('yesterday', now);
    expect(formatUsageTime(today[0])).toBe('2026-09-29 00:00:00');
    expect(yesterday[1]).toBe(today[0]);
    expect(parseUsageTime('2026-09-29 12:00:00')).toBe(now);
    expect(Number.isNaN(parseUsageTime('2026-02-30 12:00:00'))).toBe(true);
    expect(usagePresetRange('7d', now)).toEqual([now - 7 * 86400, now]);
  });
  test('reject invalid windows rather than clamp', () => {
    expect(validUsageRange([now - 30 * 86400, now], now)).toBe(true);
    for (const range of [
      [now, now],
      [now, now - 1],
      [now - 31 * 86400, now],
      [NaN, now],
      [now, now + 60],
    ])
      expect(validUsageRange(range, now)).toBe(false);
  });
  test('links preserve exact range and scoped logs', () => {
    const range = [100, 200];
    expect(
      initialChannelUsage('?view=usage&start_time=100&end_time=200').range,
    ).toEqual(range);
    const url = new URL(
      channelUsageLogURL(
        { children: [{ id: 1 }, { id: 3 }] },
        { start_time: 100, end_time: 200, model: 'gpt', group: 'vip' },
      ),
      'https://example.com',
    );
    expect(url.searchParams.get('channel_ids')).toBe('1,3');
    expect(url.searchParams.get('end_time')).toBe('200');
    expect(url.searchParams.get('model')).toBe('%gpt%');
  });
  test('tag success is weighted by counts, never averaged percentages', () => {
    const total = sumChannelUsage([
      {
        usage: {
          total_requests: 1,
          success_requests: 0,
          failed_requests: 1,
          total_quota: 20,
        },
      },
      {
        usage: {
          total_requests: 9,
          success_requests: 9,
          failed_requests: 0,
          total_quota: 80,
        },
      },
    ]);
    expect(total.total_quota).toBe(100);
    expect(total.success_requests / total.total_requests).toBe(0.9);
  });
});
