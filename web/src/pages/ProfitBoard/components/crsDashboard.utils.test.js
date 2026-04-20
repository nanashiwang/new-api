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
  buildCRSGroupOptions,
  buildCRSUsageWindows,
  filterCRSAccounts,
  getCRSLatestSyncAt,
  getCRSPlatformBadgeLabel,
  getCRSQuotaState,
  isValidCRSPort,
  joinCRSHostPort,
  splitCRSHostPort,
} from './crsDashboard.utils.js';

assert.deepEqual(
  buildCRSGroupOptions([
    { id: 1, group: 'alpha' },
    { id: 2, group: 'beta' },
    { id: 3, group: 'alpha' },
    { id: 4, group: '  ' },
  ]),
  [
    { label: 'alpha', value: 'alpha' },
    { label: 'beta', value: 'beta' },
  ],
);

assert.deepEqual(buildCRSGroupOptions([{ id: 1, group: 'alpha' }], 'gamma'), [
  { label: 'alpha', value: 'alpha' },
  { label: 'gamma', value: 'gamma' },
]);

assert.equal(
  getCRSQuotaState({
    quota_unlimited: true,
    quota_total: 0,
    quota_remaining: 0,
  }),
  'unlimited',
);

assert.equal(
  getCRSQuotaState({
    quota_unlimited: false,
    quota_total: 100,
    quota_remaining: 0,
  }),
  'empty',
);

assert.equal(
  getCRSQuotaState({
    quota_unlimited: false,
    quota_total: 100,
    quota_remaining: 8,
  }),
  'low',
);

assert.equal(
  getCRSQuotaState({
    quota_unlimited: false,
    quota_total: 100,
    quota_remaining: 42,
  }),
  'normal',
);

assert.equal(
  getCRSLatestSyncAt([
    { id: 1, last_synced_at: 100 },
    { id: 2, last_synced_at: 0 },
    { id: 3, last_synced_at: 220 },
  ]),
  220,
);

assert.deepEqual(
  filterCRSAccounts(
    [
      {
        id: 1,
        name: 'Claude Max',
        remote_account_id: 'acct-1',
        platform: 'claude',
        subscription_plan: 'pro',
        quota_unlimited: false,
        quota_total: 100,
        quota_remaining: 6,
      },
      {
        id: 2,
        name: 'OpenAI Pool',
        remote_account_id: 'acct-2',
        platform: 'openai',
        subscription_plan: 'team',
        quota_unlimited: false,
        quota_total: 100,
        quota_remaining: 0,
      },
      {
        id: 3,
        name: 'Gemini Shared',
        remote_account_id: 'acct-3',
        platform: 'gemini',
        subscription_plan: 'unlimited',
        quota_unlimited: true,
        quota_total: 0,
        quota_remaining: 0,
      },
    ],
    {
      keyword: 'claude',
      platform: 'claude',
      quotaState: 'low',
    },
  ).map((item) => item.id),
  [1],
);

assert.deepEqual(
  filterCRSAccounts(
    [
      {
        id: 1,
        name: 'Claude Max',
        remote_account_id: 'acct-1',
        platform: 'claude',
        subscription_plan: 'pro',
        quota_unlimited: false,
        quota_total: 100,
        quota_remaining: 6,
      },
      {
        id: 2,
        name: 'OpenAI Pool',
        remote_account_id: 'acct-2',
        platform: 'openai',
        subscription_plan: 'team',
        quota_unlimited: false,
        quota_total: 100,
        quota_remaining: 0,
      },
      {
        id: 3,
        name: 'Gemini Shared',
        remote_account_id: 'acct-3',
        platform: 'gemini',
        subscription_plan: 'unlimited',
        quota_unlimited: true,
        quota_total: 0,
        quota_remaining: 0,
      },
    ],
    {
      keyword: 'acct-3',
      platform: '',
      quotaState: 'unlimited',
    },
  ).map((item) => item.id),
  [3],
);

assert.deepEqual(splitCRSHostPort('example.com:3000'), {
  host: 'example.com',
  port: '3000',
});

assert.deepEqual(splitCRSHostPort('example.com:not-a-port'), {
  host: 'example.com:not-a-port',
  port: '',
});

assert.equal(joinCRSHostPort('example.com', '3000'), 'example.com:3000');
assert.equal(joinCRSHostPort('example.com', ''), 'example.com');
assert.equal(isValidCRSPort('1'), true);
assert.equal(isValidCRSPort('65535'), true);
assert.equal(isValidCRSPort('0'), false);
assert.equal(isValidCRSPort('65536'), false);

assert.equal(
  getCRSPlatformBadgeLabel({
    platform: 'claude',
    account_type: 'shared',
    subscription_info: {
      accountType: 'max',
    },
  }),
  'Claude Max / 共享',
);

assert.equal(
  getCRSPlatformBadgeLabel({
    platform: 'azure_openai',
    account_type: 'dedicated',
  }),
  'Azure OpenAI / 专属',
);

assert.equal(
  getCRSPlatformBadgeLabel({
    platform: 'openai-responses',
    account_type: 'group',
  }),
  'OpenAI Responses / 分组',
);

assert.deepEqual(
  buildCRSUsageWindows({
    usage_windows: [
      {
        key: 'seven_day',
        label: '7d',
        progress: 82,
        remaining_text: '余 2 天',
        reset_at: '2026-04-27T00:00:00Z',
        tone: 'warning',
      },
    ],
    session_window_progress: 10,
    quota_percentage: 20,
  }),
  [
    {
      key: 'seven_day',
      label: '7d',
      progress: 82,
      remainingText: '余 2 天',
      resetAt: '2026-04-27T00:00:00Z',
      tone: 'warning',
      source: 'usage_windows',
    },
  ],
);

assert.deepEqual(
  buildCRSUsageWindows({
    session_window_active: true,
    session_window_progress: 64.5,
    session_window_remaining: '5h 12m',
    session_window_end_at: '2026-04-20T15:00:00Z',
  }),
  [
    {
      key: 'session_window',
      label: '5h',
      progress: 64.5,
      remainingText: '5h 12m',
      resetAt: '2026-04-20T15:00:00Z',
      tone: 'info',
      source: 'session_window',
    },
  ],
);

assert.deepEqual(
  buildCRSUsageWindows({
    quota_percentage: 42.5,
    quota_remaining: 11.5,
    quota_reset_at: '2026-04-21T00:00:00Z',
  }),
  [
    {
      key: 'quota',
      label: '额度',
      progress: 42.5,
      remainingText: '11.5',
      resetAt: '2026-04-21T00:00:00Z',
      tone: 'success',
      source: 'quota',
    },
  ],
);
