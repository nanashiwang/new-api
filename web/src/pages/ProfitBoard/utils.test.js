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

import {
  buildInvalidBatchSelectionError,
  buildInvalidBatchSelectionErrors,
  groupWarningDetailsByScope,
} from './utils.js';

const grouped = groupWarningDetailsByScope([
  {
    scope_type: 'tag',
    scope_label: 'api.aipaibox.com - 满血反重力',
    scope_value: 'api.aipaibox.com - 满血反重力',
    model_name: 'claude-sonnet-4-6',
    count: 14,
    reason_code: 'site_model_not_found_with_manual_missing',
    reason_label: '智能定价没找到这个模型，手动规则也没匹配到',
  },
  {
    scope_type: 'tag',
    scope_label: 'api.aipaibox.com - 满血反重力',
    scope_value: 'api.aipaibox.com - 满血反重力',
    model_name: 'claude-opus-4-6',
    count: 10,
    reason_code: 'site_model_group_unmatched_with_manual_missing',
    reason_label: '智能定价分组不匹配，手动规则也没匹配到',
  },
  {
    scope_type: 'tag',
    scope_label: 'api.aipaibox.com - 满血反重力',
    scope_value: 'api.aipaibox.com - 满血反重力',
    model_name: 'claude-sonnet-4-6',
    count: 3,
    reason_code: 'log_quota_zero',
    reason_label: '命中的价格是 0',
  },
  {
    scope_type: 'channel',
    scope_label: '同名渠道',
    scope_value: '101',
    model_name: 'gpt-5.4',
    count: 3,
    reason_code: 'manual_missing',
    reason_label: '手动规则没匹配到，也没有默认规则',
  },
  {
    scope_type: 'channel',
    scope_label: '同名渠道',
    scope_value: '202',
    model_name: 'gpt-5.3',
    count: 2,
    reason_code: 'returned_cost_missing',
    reason_label: '上游没返回费用',
  },
]);

assert.equal(grouped.length, 3);

assert.deepEqual(grouped[0], {
  scopeKey: 'tag:api.aipaibox.com - 满血反重力',
  scopeType: 'tag',
  scopeLabel: 'api.aipaibox.com - 满血反重力',
  scopeValue: 'api.aipaibox.com - 满血反重力',
  totalCount: 27,
  displayHint: '',
  models: [
    {
      modelName: 'claude-sonnet-4-6',
      count: 17,
      reasons: [
        {
          reasonCode: 'site_model_not_found_with_manual_missing',
          reasonLabel: '智能定价没找到这个模型，手动规则也没匹配到',
          count: 14,
        },
        {
          reasonCode: 'log_quota_zero',
          reasonLabel: '命中的价格是 0',
          count: 3,
        },
      ],
    },
    {
      modelName: 'claude-opus-4-6',
      count: 10,
      reasons: [
        {
          reasonCode: 'site_model_group_unmatched_with_manual_missing',
          reasonLabel: '智能定价分组不匹配，手动规则也没匹配到',
          count: 10,
        },
      ],
    },
  ],
});

assert.equal(grouped[1].scopeKey, 'channel:101');
assert.equal(grouped[1].displayHint, '#101');
assert.deepEqual(grouped[1].models, [
  {
    modelName: 'gpt-5.4',
    count: 3,
    reasons: [
      {
        reasonCode: 'manual_missing',
        reasonLabel: '手动规则没匹配到，也没有默认规则',
        count: 3,
      },
    ],
  },
]);

assert.equal(grouped[2].scopeKey, 'channel:202');
assert.equal(grouped[2].displayHint, '#202');
assert.deepEqual(grouped[2].models, [
  {
    modelName: 'gpt-5.3',
    count: 2,
    reasons: [
      {
        reasonCode: 'returned_cost_missing',
        reasonLabel: '上游没返回费用',
        count: 2,
      },
    ],
  },
]);

const channelMap = new Map([
  ['1', { id: 1, name: 'alpha' }],
  ['2', { id: 2, name: 'beta' }],
]);
const tagChannelMap = new Map([['tag-valid', ['1']]]);

assert.equal(
  buildInvalidBatchSelectionError(
    {
      name: '失效标签组合',
      scope_type: 'tag',
      tags: ['tag-valid', 'tag-missing'],
    },
    channelMap,
    tagChannelMap,
  ),
  '组合「失效标签组合」引用的标签 「tag-missing」 当前没有任何渠道，请修改后再保存',
);

assert.equal(
  buildInvalidBatchSelectionError(
    {
      name: '失效渠道组合',
      scope_type: 'channel',
      channel_ids: ['1', '3'],
    },
    channelMap,
    tagChannelMap,
  ),
  '组合「失效渠道组合」引用的渠道 #3 已不存在，请修改后再保存',
);

assert.deepEqual(
  buildInvalidBatchSelectionErrors(
    [
      {
        name: '失效标签组合',
        scope_type: 'tag',
        tags: ['tag-missing'],
      },
      {
        name: '失效渠道组合',
        scope_type: 'channel',
        channel_ids: ['3'],
      },
    ],
    channelMap,
    tagChannelMap,
  ),
  [
    '组合「失效标签组合」引用的标签 「tag-missing」 当前没有任何渠道，请修改后再保存',
    '组合「失效渠道组合」引用的渠道 #3 已不存在，请修改后再保存',
  ],
);
