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
  filterCPAAccounts,
  getCPAAccountDisplayName,
  getCPAAccountState,
  isCPAEndpointChanged,
  normalizeCPAEndpoint,
  summarizeCPAAccounts,
} from './cpaDashboard.utils.js';

assert.equal(
  normalizeCPAEndpoint(' HTTPS ', 'CPA.Example.com:8317/'),
  'https://cpa.example.com:8317',
);
assert.equal(
  isCPAEndpointChanged(
    { scheme: 'https', host: 'CPA.Example.com:8317' },
    'HTTPS',
    'cpa.example.com:8317/',
  ),
  false,
);
assert.equal(
  isCPAEndpointChanged(
    { scheme: 'https', host: 'cpa.example.com:8317' },
    'http',
    'cpa.example.com:8317',
  ),
  true,
);

const now = Date.parse('2026-07-28T08:00:00Z');
const accounts = [
  { id: 'available', status: 'active', label: 'Account A', site_id: 1 },
  {
    id: 'limited',
    status: 'active',
    unavailable: true,
    next_retry_after: '2026-07-28T09:00:00Z',
    site_id: 1,
  },
  { id: 'disabled', disabled: true, site_id: 2 },
  { id: 'failed', status: 'error', site_id: 2 },
  { id: 'pending', status: 'pending', site_id: 3 },
  { id: 'refreshing', status: 'refreshing', site_id: 3 },
  { id: 'status-disabled', status: 'disabled', site_id: 3 },
  { id: 'unknown', status: 'unknown', site_id: 3 },
  { id: 'empty-status', site_id: 3 },
  { id: 'future-status', status: 'warming_up', site_id: 3 },
];

assert.equal(getCPAAccountState(accounts[0], now), 'available');
assert.equal(getCPAAccountState(accounts[1], now), 'limited');
assert.equal(getCPAAccountState(accounts[2], now), 'disabled');
assert.equal(getCPAAccountState(accounts[3], now), 'abnormal');
assert.equal(getCPAAccountState(accounts[4], now), 'limited');
assert.equal(getCPAAccountState(accounts[5], now), 'limited');
assert.equal(getCPAAccountState(accounts[6], now), 'disabled');
assert.equal(getCPAAccountState(accounts[7], now), 'unknown');
assert.equal(getCPAAccountState(accounts[8], now), 'unknown');
assert.equal(getCPAAccountState(accounts[9], now), 'unknown');
assert.deepEqual(summarizeCPAAccounts(accounts, now), {
  total: 10,
  available: 1,
  limited: 3,
  abnormal: 1,
  disabled: 2,
  unknown: 3,
});
assert.equal(getCPAAccountDisplayName(accounts[0]), 'Account A');
assert.deepEqual(
  filterCPAAccounts(accounts, { keyword: 'account a', siteId: 1 }, now).map(
    (account) => account.id,
  ),
  ['available'],
);
assert.deepEqual(
  filterCPAAccounts(accounts, { state: 'disabled' }, now).map(
    (account) => account.id,
  ),
  ['disabled', 'status-disabled'],
);
assert.deepEqual(
  filterCPAAccounts(accounts, { state: 'unknown' }, now).map(
    (account) => account.id,
  ),
  ['unknown', 'empty-status', 'future-status'],
);
