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
export const normalizeCPAEndpoint = (scheme, host) =>
  `${String(scheme || 'https')
    .trim()
    .toLowerCase()}://${String(host || '')
    .trim()
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
    .toLowerCase()}`;

export const isCPAEndpointChanged = (site = {}, scheme, host) =>
  normalizeCPAEndpoint(site?.scheme, site?.host) !==
  normalizeCPAEndpoint(scheme, host);

export const getCPAAccountState = (account = {}, now = Date.now()) => {
  const status = String(account?.status ?? '')
    .trim()
    .toLowerCase();
  if (account?.disabled || status === 'disabled') return 'disabled';
  if (account?.unavailable || status === 'pending' || status === 'refreshing') {
    return 'limited';
  }
  const retryAt = Date.parse(account?.next_retry_after ?? '');
  if (Number.isFinite(retryAt) && retryAt > now) return 'limited';
  if (['error', 'failed', 'invalid', 'unavailable'].includes(status)) {
    return 'abnormal';
  }
  if (status === 'active') return 'available';
  return 'unknown';
};

export const getCPAAccountDisplayName = (account = {}) =>
  [account?.label, account?.account, account?.email, account?.name, account?.id]
    .map((value) => String(value ?? '').trim())
    .find(Boolean) ?? '-';

export const getCPAProviderName = (account = {}) =>
  String(account?.provider || account?.type || '-').trim() || '-';

export const summarizeCPAAccounts = (accounts = [], now = Date.now()) => {
  const summary = {
    total: accounts.length,
    available: 0,
    limited: 0,
    abnormal: 0,
    disabled: 0,
    unknown: 0,
  };
  accounts.forEach((account) => {
    const state = getCPAAccountState(account, now);
    summary[state] += 1;
  });
  return summary;
};

export const filterCPAAccounts = (
  accounts = [],
  { keyword = '', siteId = 0, state = '' } = {},
  now = Date.now(),
) => {
  const normalizedKeyword = String(keyword).trim().toLowerCase();
  const normalizedSiteId = Number(siteId || 0);
  return accounts.filter((account) => {
    if (normalizedSiteId > 0 && Number(account?.site_id) !== normalizedSiteId) {
      return false;
    }
    if (state && getCPAAccountState(account, now) !== state) return false;
    if (!normalizedKeyword) return true;
    return [
      account?.label,
      account?.account,
      account?.email,
      account?.name,
      account?.id,
      account?.auth_index,
      account?.provider,
      account?.type,
      account?.site_name,
    ].some((value) =>
      String(value ?? '')
        .toLowerCase()
        .includes(normalizedKeyword),
    );
  });
};
