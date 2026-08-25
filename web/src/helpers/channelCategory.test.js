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
  buildChannelCategoryQuery,
  CHANNEL_CATEGORY_ALL,
  CHANNEL_CATEGORY_MIMO,
  channelTypeCategoryKey,
} from './channelCategory';

describe('channel category query', () => {
  test('keeps protocol and vendor categories independent', () => {
    expect(channelTypeCategoryKey(1)).toBe('type:1');
    expect(CHANNEL_CATEGORY_MIMO).toBe('vendor:mimo');
  });

  test('only sends category outside tag mode', () => {
    expect(buildChannelCategoryQuery(CHANNEL_CATEGORY_ALL, false)).toBe('');
    expect(buildChannelCategoryQuery('type:1', false)).toBe(
      '&category=type%3A1',
    );
    expect(buildChannelCategoryQuery(CHANNEL_CATEGORY_MIMO, false)).toBe(
      '&category=vendor%3Amimo',
    );
    expect(buildChannelCategoryQuery(CHANNEL_CATEGORY_MIMO, true)).toBe('');
  });
});
