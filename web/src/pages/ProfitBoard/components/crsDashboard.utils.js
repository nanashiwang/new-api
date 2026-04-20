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
const QUOTA_LOW_THRESHOLD = 10;

export const buildCRSGroupOptions = (sites = [], currentGroup = '') => {
  const groups = new Set();
  sites.forEach((site) => {
    const group = String(site?.group || '').trim();
    if (group) groups.add(group);
  });
  const normalizedCurrent = String(currentGroup || '').trim();
  if (normalizedCurrent) groups.add(normalizedCurrent);
  return Array.from(groups)
    .sort((left, right) => left.localeCompare(right, 'zh-CN'))
    .map((group) => ({
      label: group,
      value: group,
    }));
};

export const getCRSQuotaState = (account) => {
  if (account?.quota_unlimited) return 'unlimited';
  const total = Number(account?.quota_total || 0);
  const remaining = Number(account?.quota_remaining || 0);
  if (total > 0 && remaining <= 0) return 'empty';
  if (remaining > 0 && remaining <= QUOTA_LOW_THRESHOLD) return 'low';
  return 'normal';
};

export const getCRSPlatformOptions = (accounts = []) =>
  Array.from(
    new Set(
      accounts
        .map((account) => String(account?.platform || '').trim())
        .filter(Boolean),
    ),
  )
    .sort((left, right) => left.localeCompare(right, 'en'))
    .map((platform) => ({
      label: platform,
      value: platform,
    }));

export const getCRSLatestSyncAt = (sites = []) =>
  sites.reduce((latest, site) => {
    const value = Number(site?.last_synced_at || 0);
    return value > latest ? value : latest;
  }, 0);

export const filterCRSAccounts = (
  accounts = [],
  { keyword = '', platform = '', quotaState = '' } = {},
) => {
  const normalizedKeyword = String(keyword || '')
    .trim()
    .toLowerCase();
  const normalizedPlatform = String(platform || '').trim();
  const normalizedQuotaState = String(quotaState || '').trim();

  return accounts.filter((account) => {
    if (normalizedPlatform && account?.platform !== normalizedPlatform) {
      return false;
    }

    if (
      normalizedQuotaState &&
      getCRSQuotaState(account) !== normalizedQuotaState
    ) {
      return false;
    }

    if (!normalizedKeyword) return true;

    const haystack = [
      account?.name,
      account?.remote_account_id,
      account?.subscription_plan,
      account?.status,
    ]
      .map((value) =>
        String(value || '')
          .trim()
          .toLowerCase(),
      )
      .filter(Boolean)
      .join(' ');

    return haystack.includes(normalizedKeyword);
  });
};
