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

const CRS_PLATFORM_DISPLAY_NAMES = {
  claude: 'Claude',
  'claude-console': 'Claude Console',
  openai: 'OpenAI',
  gemini: 'Gemini',
  ccr: 'CCR',
  bedrock: 'Bedrock',
  aws: 'Bedrock',
  'azure-openai': 'Azure OpenAI',
  azure_openai: 'Azure OpenAI',
  azure: 'Azure OpenAI',
  'openai-api': 'OpenAI API',
  'openai-responses': 'OpenAI Responses',
  droid: 'Droid',
  'droid-cli': 'Droid',
};

const CRS_ACCOUNT_TYPE_DISPLAY_NAMES = {
  shared: '共享',
  dedicated: '专属',
  group: '分组',
  team: '团队',
  workspace: '工作区',
  sub: '子账号',
};

const CRS_TONE_LEVELS = ['muted', 'success', 'info', 'warning', 'danger'];

const normalizeText = (value) => String(value || '').trim();

const clampProgress = (value) => {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return null;
  return Math.min(100, Math.max(0, numeric));
};

const normalizeTone = (tone, progress) => {
  const normalizedTone = normalizeText(tone).toLowerCase();
  if (CRS_TONE_LEVELS.includes(normalizedTone)) {
    return normalizedTone;
  }

  if (progress === null) return 'muted';
  if (progress >= 100) return 'danger';
  if (progress >= 85) return 'warning';
  if (progress >= 45) return 'info';
  return 'success';
};

const pickFirstText = (...values) =>
  values.map(normalizeText).find(Boolean) || '';

const formatWindowRemainingText = (value) => {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return `${value}`;
  }
  return normalizeText(value);
};

const getCRSSubscriptionDisplayName = (account) => {
  const planText = pickFirstText(
    account?.subscription_info?.accountType,
    account?.subscription_info?.plan,
    account?.subscription_info?.planName,
    account?.subscription_plan,
  ).toLowerCase();

  if (!planText) return '';
  if (planText.includes('max')) return 'Claude Max';
  if (planText.includes('pro')) return 'Claude Pro';
  return '';
};

const getCRSPlatformDisplayName = (platform) => {
  const normalizedPlatform = normalizeText(platform).toLowerCase();
  return CRS_PLATFORM_DISPLAY_NAMES[normalizedPlatform] || normalizeText(platform);
};

const getCRSAccountTypeDisplayName = (account) => {
  const normalizedAccountType = normalizeText(
    account?.account_type || account?.accountType,
  ).toLowerCase();

  if (!normalizedAccountType) return '共享';
  return (
    CRS_ACCOUNT_TYPE_DISPLAY_NAMES[normalizedAccountType] ||
    normalizeText(account?.account_type || account?.accountType)
  );
};

const normalizeUsageWindow = (window, source = 'usage_windows') => {
  const progress = clampProgress(window?.progress);
  return {
    key: pickFirstText(window?.key, window?.label, source),
    label: pickFirstText(window?.label, 'Window'),
    progress,
    remainingText: formatWindowRemainingText(
      window?.remainingText ?? window?.remaining_text ?? '',
    ),
    resetAt: pickFirstText(window?.resetAt, window?.reset_at),
    tone: normalizeTone(window?.tone, progress),
    source,
  };
};

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

export const splitCRSHostPort = (value = '') => {
  const normalizedValue = normalizeText(value);
  if (!normalizedValue) {
    return { host: '', port: '' };
  }

  const matches = normalizedValue.match(/^([^:]+):(\d{1,5})$/);
  if (!matches) {
    return { host: normalizedValue, port: '' };
  }

  const [, host, port] = matches;
  if (!isValidCRSPort(port)) {
    return { host: normalizedValue, port: '' };
  }

  return { host: normalizeText(host), port };
};

export const isValidCRSPort = (value = '') => {
  if (!/^\d+$/.test(normalizeText(value))) return false;
  const port = Number(value);
  return Number.isInteger(port) && port >= 1 && port <= 65535;
};

export const joinCRSHostPort = (host = '', port = '') => {
  const normalizedHost = normalizeText(host);
  const normalizedPort = normalizeText(port);
  if (!normalizedPort) return normalizedHost;
  return `${normalizedHost}:${normalizedPort}`;
};

export const formatCRSDailyCost = (cost, currency = 'USD') => {
  const numeric = Number(cost);
  if (!Number.isFinite(numeric) || numeric <= 0) return '';
  const symbol = currency === 'USD' ? '$' : '';
  const fixed = numeric >= 1 ? numeric.toFixed(2) : numeric.toFixed(4);
  const [intPart, decPart] = fixed.split('.');
  const withCommas = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return `${symbol}${withCommas}.${decPart}`;
};

export const formatCRSTokenCount = (tokens) => {
  const numeric = Number(tokens);
  if (!Number.isFinite(numeric) || numeric <= 0) return '';
  if (numeric >= 1_000_000_000) {
    return `${(numeric / 1_000_000_000).toFixed(2)}B`;
  }
  if (numeric >= 1_000_000) {
    return `${(numeric / 1_000_000).toFixed(2)}M`;
  }
  if (numeric >= 1_000) {
    return `${(numeric / 1_000).toFixed(1)}K`;
  }
  return `${Math.round(numeric)}`;
};

export const formatCRSRequestCount = (count) => {
  const numeric = Number(count);
  if (!Number.isFinite(numeric) || numeric <= 0) return '0';
  return Math.round(numeric).toLocaleString('en-US');
};

export const getCRSPlatformBadgeLabel = (account = {}) => {
  const normalizedPlatform = normalizeText(account?.platform).toLowerCase();
  const displayName =
    normalizedPlatform === 'claude'
      ? getCRSSubscriptionDisplayName(account) ||
        getCRSPlatformDisplayName(normalizedPlatform)
      : getCRSPlatformDisplayName(normalizedPlatform);

  return `${displayName || '-'} / ${getCRSAccountTypeDisplayName(account)}`;
};

export const buildCRSUsageWindows = (account = {}) => {
  const usageWindows = Array.isArray(account?.usage_windows)
    ? account.usage_windows
    : Array.isArray(account?.usageWindows)
      ? account.usageWindows
      : [];

  if (usageWindows.length > 0) {
    return usageWindows.map((window) =>
      normalizeUsageWindow(window, 'usage_windows'),
    );
  }

  const sessionWindowProgress = clampProgress(account?.session_window_progress);
  if (
    account?.session_window_active ||
    sessionWindowProgress !== null ||
    normalizeText(account?.session_window_remaining) ||
    normalizeText(account?.session_window_end_at)
  ) {
    return [
      normalizeUsageWindow(
        {
          key: 'session_window',
          label: '5h',
          progress: sessionWindowProgress,
          remainingText: account?.session_window_remaining,
          resetAt: account?.session_window_end_at,
        },
        'session_window',
      ),
    ];
  }

  const quotaProgress =
    clampProgress(account?.quota_percentage) ??
    (Number(account?.quota_total || 0) > 0
      ? clampProgress(
          (Number(account?.quota_used || 0) / Number(account?.quota_total || 0)) *
            100,
        )
      : null);
  const quotaRemaining = account?.quota_unlimited
    ? 'Unlimited'
    : account?.quota_remaining;
  if (
    account?.quota_unlimited ||
    quotaProgress !== null ||
    Number(account?.quota_total || 0) > 0 ||
    Number(account?.quota_remaining || 0) > 0 ||
    normalizeText(account?.quota_reset_at)
  ) {
    return [
      normalizeUsageWindow(
        {
          key: 'quota',
          label: '额度',
          progress: quotaProgress,
          remainingText: quotaRemaining,
          resetAt: account?.quota_reset_at,
          tone: account?.quota_unlimited ? 'muted' : undefined,
        },
        'quota',
      ),
    ];
  }

  return [];
};

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
