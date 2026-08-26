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

import assert from 'node:assert/strict';
import { describe, test } from 'node:test';

import {
  TOKEN_GROUP_OTHER_VENDOR,
  buildTokenGroupVendorOptions,
  filterTokenGroupsByVendor,
  formatTokenGroupSelectedLabel,
  resolveTokenGroupVendor,
  shouldClearTokenGroupForVendor,
  sortTokenGroupOptions,
} from './tokenGroupUtils.js';

describe('tokenGroupUtils', () => {
  test('resolves configured and legacy vendor group names', () => {
    assert.equal(resolveTokenGroupVendor('OpenAI · 优质'), 'OpenAI');
    assert.equal(resolveTokenGroupVendor('Claude · 企业专属'), 'Claude');
    assert.equal(
      resolveTokenGroupVendor('Deepseek · 优质（第三方）'),
      'DeepSeek',
    );
    assert.equal(resolveTokenGroupVendor('gemini'), 'Gemini');
    assert.equal(resolveTokenGroupVendor('Anthropic · 优质'), 'Anthropic');
  });

  test('keeps unscoped and automatic groups under the fallback vendor', () => {
    assert.equal(resolveTokenGroupVendor('官方渠道'), TOKEN_GROUP_OTHER_VENDOR);
    assert.equal(resolveTokenGroupVendor('default'), TOKEN_GROUP_OTHER_VENDOR);
    assert.equal(resolveTokenGroupVendor('auto'), TOKEN_GROUP_OTHER_VENDOR);
    assert.equal(resolveTokenGroupVendor(''), '');
  });

  test('builds compact vendor options and keeps fallback last', () => {
    assert.deepEqual(
      buildTokenGroupVendorOptions(
        [
          { value: 'OpenAI · 优质' },
          { value: 'OpenAI · 企业专属' },
          { value: 'Claude · 优质' },
          { value: 'default' },
        ],
        '其他',
      ),
      [
        { value: 'OpenAI', label: 'OpenAI · 2' },
        { value: 'Claude', label: 'Claude · 1' },
        { value: TOKEN_GROUP_OTHER_VENDOR, label: '其他 · 1' },
      ],
    );
  });

  test('sorts vendors and service tiers in a stable business order', () => {
    const groups = [
      { value: 'Claude · 特价（第三方）' },
      { value: 'MiMo · 企业专属' },
      { value: 'OpenAI · 优质（第三方）' },
      { value: 'default' },
      { value: 'OpenAI · 企业专属' },
      { value: 'Claude · 优质' },
      { value: 'OpenAI · 优质' },
      { value: 'Deepseek · 标准（第三方）' },
    ];

    assert.deepEqual(
      sortTokenGroupOptions(groups).map((group) => group.value),
      [
        'OpenAI · 企业专属',
        'OpenAI · 优质',
        'OpenAI · 优质（第三方）',
        'Claude · 优质',
        'Claude · 特价（第三方）',
        'Deepseek · 标准（第三方）',
        'MiMo · 企业专属',
        'default',
      ],
    );
  });

  test('builds vendor options in the same fixed order', () => {
    const groups = [
      { value: 'MiMo · 优质' },
      { value: 'Claude · 优质' },
      { value: 'OpenAI · 优质' },
      { value: 'default' },
      { value: 'Gemini · 优质' },
    ];

    assert.deepEqual(
      buildTokenGroupVendorOptions(groups, '其他').map((item) => item.value),
      ['OpenAI', 'Claude', 'Gemini', 'MiMo', TOKEN_GROUP_OTHER_VENDOR],
    );
  });

  test('filters groups without changing their configured order', () => {
    const groups = [
      { value: 'Claude · 企业专属' },
      { value: 'OpenAI · 优质' },
      { value: 'Claude · 优质' },
    ];

    assert.deepEqual(filterTokenGroupsByVendor(groups, 'Claude'), [
      { value: 'Claude · 企业专属' },
      { value: 'Claude · 优质' },
    ]);
    assert.deepEqual(filterTokenGroupsByVendor(groups, ''), []);
  });

  test('optionally includes common groups after the selected vendor', () => {
    const groups = sortTokenGroupOptions([
      { value: '企业专属' },
      { value: 'OpenAI · 优质' },
      { value: 'Claude · 优质' },
      { value: 'default' },
    ]);

    assert.deepEqual(
      filterTokenGroupsByVendor(groups, 'OpenAI', true).map(
        (group) => group.value,
      ),
      ['OpenAI · 优质', 'default', '企业专属'],
    );
  });

  test('clears only groups that do not belong to the newly selected vendor', () => {
    assert.equal(
      shouldClearTokenGroupForVendor('Claude · 优质', 'OpenAI'),
      true,
    );
    assert.equal(
      shouldClearTokenGroupForVendor('Claude · 优质', 'Claude'),
      false,
    );
    assert.equal(shouldClearTokenGroupForVendor('', 'Claude'), false);
  });

  test('formats selected groups with numeric and automatic ratios', () => {
    assert.equal(
      formatTokenGroupSelectedLabel({
        value: 'Claude · 优质',
        ratio: 1,
      }),
      'Claude · 优质 · 1x',
    );
    assert.equal(
      formatTokenGroupSelectedLabel({ value: 'auto', ratio: '自动' }),
      'auto · 自动',
    );
    assert.equal(
      formatTokenGroupSelectedLabel({ value: 'default' }),
      'default',
    );
  });
});
