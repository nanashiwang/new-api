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
import dayjs from 'dayjs';

export const createDefaultUpstreamAccountDraft = () => ({
  id: 0,
  name: '',
  remark: '',
  account_type: 'newapi',
  base_url: '',
  user_id: 0,
  email: '',
  access_token: '',
  access_token_masked: '',
  password: '',
  password_masked: '',
  resource_display_mode: 'both',
  enabled: true,
});

export const normalizeUpstreamAccountType = (value) =>
  String(value || '')
    .trim()
    .toLowerCase() === 'sub2api'
    ? 'sub2api'
    : 'newapi';

export const normalizeUpstreamAccountResourceDisplayMode = (
  value,
  accountType = 'newapi',
) => {
  if (normalizeUpstreamAccountType(accountType) === 'sub2api') {
    return 'wallet';
  }
  switch (String(value || '').trim()) {
    case 'wallet':
      return 'wallet';
    case 'subscription':
      return 'subscription';
    default:
      return 'both';
  }
};

export const normalizeUpstreamAccountBaseUrl = (value) => {
  let next = String(value || '').trim();
  if (!next) return '';
  if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(next)) {
    next = `https://${next}`;
  }
  return next.replace(/\/+$/, '');
};

export const getUpstreamAccountSuggestedName = (baseUrl) => {
  const normalized = normalizeUpstreamAccountBaseUrl(baseUrl);
  if (!normalized) return '';
  try {
    return new URL(normalized).host.replace(/^www\./i, '');
  } catch (error) {
    return normalized
      .replace(/^[a-z][a-z0-9+.-]*:\/\//i, '')
      .split('/')[0]
      .replace(/^www\./i, '');
  }
};

export const prepareUpstreamAccountDraftForSave = (
  draft,
  { allowSuggestedName = true } = {},
) => {
  const baseUrl = normalizeUpstreamAccountBaseUrl(draft?.base_url);
  const suggestedName = getUpstreamAccountSuggestedName(baseUrl);
  const accountType = normalizeUpstreamAccountType(draft?.account_type);
  return {
    ...draft,
    account_type: accountType,
    name:
      String(draft?.name || '').trim() ||
      (allowSuggestedName ? suggestedName : ''),
    remark: String(draft?.remark || '').trim(),
    base_url: baseUrl,
    email: String(draft?.email || '').trim(),
    access_token: String(draft?.access_token || '').trim(),
    password: String(draft?.password || '').trim(),
    user_id: accountType === 'sub2api' ? 0 : Number(draft?.user_id || 0),
    resource_display_mode: normalizeUpstreamAccountResourceDisplayMode(
      draft?.resource_display_mode,
      accountType,
    ),
  };
};

export const getUpstreamAccountDraftValidation = (draft, options) => {
  const prepared = prepareUpstreamAccountDraftForSave(draft, options);
  const errors = {};
  if (!prepared.name) {
    errors.name = '请输入账户名称';
  }
  if (!prepared.base_url) {
    errors.base_url = '请输入上游地址';
  } else {
    try {
      const parsed = new URL(prepared.base_url);
      if (!['http:', 'https:'].includes(parsed.protocol)) {
        errors.base_url = '请输入有效的 URL';
      }
    } catch (error) {
      errors.base_url = '请输入有效的 URL';
    }
  }
  if (prepared.account_type === 'sub2api') {
    if (!prepared.email) {
      errors.email = '请输入邮箱';
    }
    if (!prepared.password && !String(prepared?.password_masked || '').trim()) {
      errors.password = '请输入密码';
    }
  } else {
    if (!Number.isInteger(prepared.user_id) || prepared.user_id <= 0) {
      errors.user_id = '请输入有效的用户 ID';
    }
    if (
      !prepared.access_token &&
      !String(prepared?.access_token_masked || '').trim()
    ) {
      errors.access_token = '请输入 access token';
    }
  }
  const firstError = Object.values(errors)[0] || '';
  return {
    prepared,
    errors,
    isValid: !firstError,
    firstError,
  };
};

export const getWalletStatusMeta = (status, t) =>
  ({
    ready: { color: 'green', label: t('运行中') },
    needs_baseline: { color: 'orange', label: t('建立基线中') },
    failed: { color: 'red', label: t('异常') },
    not_configured: { color: 'grey', label: t('未配置') },
    disabled: { color: 'grey', label: t('已停用') },
  })[status] || { color: 'grey', label: t('待同步') };

const RESOURCE_RISK_TONES = {
  critical: {
    accentBorderColor: '#ef4444',
    amountTone: 'text-red-600 dark:text-red-400',
    badgeTone:
      'bg-red-500/10 text-red-700 dark:bg-red-500/15 dark:text-red-300',
    statusBarTone:
      'border border-red-200 bg-red-50 text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200',
    priority: 40,
  },
  warning: {
    accentBorderColor: '#f59e0b',
    amountTone: 'text-amber-600 dark:text-amber-400',
    badgeTone:
      'bg-amber-500/10 text-amber-700 dark:bg-amber-500/15 dark:text-amber-200',
    statusBarTone:
      'border border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-400/30 dark:bg-amber-500/10 dark:text-amber-200',
    priority: 20,
  },
  healthy: {
    accentBorderColor: '#10b981',
    amountTone: 'text-emerald-600 dark:text-emerald-400',
    badgeTone:
      'bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200',
    statusBarTone: '',
    priority: 0,
  },
  neutral: {
    accentBorderColor: 'var(--semi-color-text-2)',
    amountTone: 'text-semi-color-text-0',
    badgeTone: 'bg-semi-color-fill-1 text-semi-color-text-1',
    statusBarTone: '',
    priority: -1,
  },
};

const getResourceRiskTone = (level = 'neutral') =>
  RESOURCE_RISK_TONES[level] || RESOURCE_RISK_TONES.neutral;

const getWalletRiskMeta = (account) => {
  const balance = Number(account?.wallet_balance_usd || 0);
  if (balance < 10) {
    return {
      kind: 'wallet',
      level: 'critical',
      label: '余额告急',
      statusLabel: '告急',
      statusText: '钱包余额告急',
    };
  }
  if (balance < 50) {
    return {
      kind: 'wallet',
      level: 'warning',
      label: '余额偏低',
      statusLabel: '偏低',
      statusText: '钱包余额偏低',
    };
  }
  return {
    kind: 'wallet',
    level: 'healthy',
    label: '余额正常',
    statusLabel: '正常',
    statusText: '',
  };
};

const getSubscriptionRiskMeta = (account) => {
  if (!account?.has_subscription_data) {
    return {
      kind: 'subscription',
      level: 'neutral',
      label: '未获取到订阅数据',
      statusLabel: '未知',
      statusText: '',
    };
  }
  const expireAt = Number(account?.subscription_earliest_expire_at || 0);
  if (expireAt <= 0) {
    return {
      kind: 'subscription',
      level: 'healthy',
      label: '订阅正常',
      statusLabel: '正常',
      statusText: '',
    };
  }
  const hoursLeft = dayjs.unix(expireAt).diff(dayjs(), 'hour', true);
  if (hoursLeft <= 24) {
    return {
      kind: 'subscription',
      level: 'critical',
      label: '订阅即将到期',
      statusLabel: '1天内到期',
      statusText: '订阅即将到期',
    };
  }
  if (hoursLeft <= 24 * 7) {
    return {
      kind: 'subscription',
      level: 'warning',
      label: '订阅临近到期',
      statusLabel: '7天内到期',
      statusText: '订阅临近到期',
    };
  }
  return {
    kind: 'subscription',
    level: 'healthy',
    label: '订阅正常',
    statusLabel: '正常',
    statusText: '',
  };
};

const getRiskPriority = (risk) => {
  const tone = getResourceRiskTone(risk?.level);
  const kindBias = risk?.kind === 'subscription' ? 2 : 1;
  return tone.priority + kindBias;
};

export const formatUpstreamExpiryDate = (timestamp, t) => {
  const next = Number(timestamp || 0);
  if (next <= 0) return t('无到期时间');
  return dayjs.unix(next).format('YYYY-MM-DD');
};

export const formatUpstreamSubscriptionRemaining = (account, status, t) => {
  if (!account?.has_subscription_data) return '--';
  if (account?.subscription_has_unlimited) return t('不限额');
  return formatMoney(account?.subscription_remaining_quota_usd, status);
};

export const buildAccountResourceMetrics = (account, status, t) => {
  const accountStatus = account?.status || '';
  const hasFailedSync = accountStatus === 'failed';
  const isDisabled =
    accountStatus === 'disabled' || accountStatus === 'not_configured';
  const resourceDisplayMode = normalizeUpstreamAccountResourceDisplayMode(
    account?.resource_display_mode,
    account?.account_type,
  );
  const showWallet =
    resourceDisplayMode === 'both' || resourceDisplayMode === 'wallet';
  const showSubscription =
    resourceDisplayMode === 'both' || resourceDisplayMode === 'subscription';

  const buildWalletMetric = (risk, overrides = {}) => {
    const tone = getResourceRiskTone(risk?.level);
    return {
      key: 'wallet',
      kind: 'wallet',
      risk,
      title: t('钱包余额'),
      value:
        overrides.value ?? formatMoney(account?.wallet_balance_usd, status),
      valueTone: overrides.valueTone || tone.amountTone,
      badgeTone: overrides.badgeTone || tone.badgeTone,
      statusLabel: overrides.statusLabel || t(risk?.statusLabel || '正常'),
      metaItems: [
        {
          label: t('累计已用'),
          value: formatMoney(account?.wallet_used_total_usd, status),
        },
      ],
    };
  };

  const buildSubscriptionMetric = (risk, overrides = {}) => {
    const tone = getResourceRiskTone(risk?.level);
    const hasSubscriptionData = !!account?.has_subscription_data;
    return {
      key: 'subscription',
      kind: 'subscription',
      risk,
      title: t('订阅剩余'),
      value:
        overrides.value ??
        (hasSubscriptionData
          ? formatUpstreamSubscriptionRemaining(account, status, t)
          : '--'),
      valueTone: overrides.valueTone || tone.amountTone,
      badgeTone: overrides.badgeTone || tone.badgeTone,
      statusLabel:
        overrides.statusLabel ||
        (hasSubscriptionData ? t(risk?.statusLabel || '正常') : t('未获取')),
      metaItems: hasSubscriptionData
        ? [
            {
              label: t('订阅已用'),
              value: formatMoney(account?.subscription_used_quota_usd, status),
            },
            {
              label: t('最早到期'),
              value: formatUpstreamExpiryDate(
                account?.subscription_earliest_expire_at,
                t,
              ),
            },
          ]
        : [
            {
              label: t('订阅状态'),
              value: t('未获取到订阅数据'),
            },
          ],
    };
  };

  if (hasFailedSync) {
    const neutralTone = getResourceRiskTone('neutral');
    return {
      metrics: [
        ...(showWallet
          ? [
              buildWalletMetric(
                {
                  kind: 'wallet',
                  level: 'neutral',
                  statusLabel: '同步失败',
                },
                {
                  value: '--',
                  valueTone: neutralTone.amountTone,
                  badgeTone: neutralTone.badgeTone,
                  statusLabel: t('同步失败'),
                },
              ),
            ]
          : []),
        ...(showSubscription
          ? [
              buildSubscriptionMetric(
                {
                  kind: 'subscription',
                  level: 'neutral',
                  statusLabel: '同步失败',
                },
                {
                  value: '--',
                  valueTone: neutralTone.amountTone,
                  badgeTone: neutralTone.badgeTone,
                  statusLabel: t('同步失败'),
                },
              ),
            ]
          : []),
      ],
      statusBar: null,
      accentBorderColor: neutralTone.accentBorderColor,
    };
  }

  if (isDisabled) {
    const neutralTone = getResourceRiskTone('neutral');
    return {
      metrics: [
        ...(showWallet
          ? [
              buildWalletMetric(
                {
                  kind: 'wallet',
                  level: 'neutral',
                  statusLabel: '已停用',
                },
                {
                  valueTone: neutralTone.amountTone,
                  badgeTone: neutralTone.badgeTone,
                  statusLabel: t('已停用'),
                },
              ),
            ]
          : []),
        ...(showSubscription
          ? [
              buildSubscriptionMetric(
                {
                  kind: 'subscription',
                  level: 'neutral',
                  statusLabel: '已停用',
                },
                {
                  valueTone: neutralTone.amountTone,
                  badgeTone: neutralTone.badgeTone,
                  statusLabel: t('已停用'),
                },
              ),
            ]
          : []),
      ],
      statusBar: null,
      accentBorderColor: neutralTone.accentBorderColor,
    };
  }

  const metrics = [];
  if (showWallet) {
    const walletRisk = getWalletRiskMeta(account);
    metrics.push(buildWalletMetric(walletRisk));
  }

  if (showSubscription) {
    const subscriptionRisk = getSubscriptionRiskMeta(account);
    metrics.push(buildSubscriptionMetric(subscriptionRisk));
  }

  const riskCandidates = metrics
    .map((item) => item.risk)
    .filter(
      (item) => item && (item.level === 'warning' || item.level === 'critical'),
    )
    .sort((left, right) => getRiskPriority(right) - getRiskPriority(left));

  const sharedRisk = riskCandidates[0] || null;
  const sharedRiskTone = getResourceRiskTone(sharedRisk?.level);
  const walletBalanceText = formatMoney(account?.wallet_balance_usd, status);
  const expiryDateText = formatUpstreamExpiryDate(
    account?.subscription_earliest_expire_at,
    t,
  );
  const topAccent =
    metrics
      .map((item) => getResourceRiskTone(item.risk?.level))
      .sort((left, right) => right.priority - left.priority)[0] ||
    getResourceRiskTone('healthy');

  let sharedRiskText = '';
  if (sharedRisk?.kind === 'wallet') {
    sharedRiskText =
      sharedRisk.level === 'critical'
        ? t('钱包余额告急，当前 {{value}}', { value: walletBalanceText })
        : t('钱包余额偏低，当前 {{value}}', { value: walletBalanceText });
  } else if (sharedRisk?.kind === 'subscription') {
    sharedRiskText =
      sharedRisk.level === 'critical'
        ? t('订阅即将到期，最早 {{date}}', { date: expiryDateText })
        : t('订阅临近到期，最早 {{date}}', { date: expiryDateText });
  }

  return {
    metrics,
    statusBar: sharedRisk
      ? {
          text: sharedRiskText,
          tone: sharedRiskTone.statusBarTone,
        }
      : null,
    accentBorderColor: topAccent.accentBorderColor,
  };
};

export const getAccountBalanceVisualMeta = (account, status, t) => {
  const { metrics, statusBar, accentBorderColor } = buildAccountResourceMetrics(
    account,
    status,
    t,
  );
  const walletMetric = metrics.find((item) => item.key === 'wallet');
  return {
    level: walletMetric?.risk?.level || 'neutral',
    label: walletMetric?.risk?.label ? t(walletMetric.risk.label) : '',
    helper: '',
    accentColor: accentBorderColor,
    amountTone: walletMetric?.valueTone || 'text-semi-color-text-0',
    badgeTone:
      walletMetric?.badgeTone || 'bg-semi-color-fill-1 text-semi-color-text-1',
    noticeTone: statusBar?.tone || '',
    showNotice: !!statusBar,
  };
};

export const getAccountResourceSummaryTones = (account) => ({
  wallet: getResourceRiskTone(getWalletRiskMeta(account).level).amountTone,
  subscription: getResourceRiskTone(getSubscriptionRiskMeta(account).level)
    .amountTone,
});

export const getDisplayCurrency = (status) => {
  const displayType = status?.quota_display_type || 'USD';
  if (displayType === 'CNY') {
    return { symbol: '¥', rate: status?.usd_exchange_rate || 1 };
  }
  if (displayType === 'CUSTOM') {
    return {
      symbol: status?.custom_currency_symbol || '¤',
      rate: status?.custom_currency_exchange_rate || 1,
    };
  }
  return { symbol: '$', rate: 1 };
};

export const formatMoney = (value, status, digits = 3) => {
  const amount = Number(value || 0);
  const { symbol, rate } = getDisplayCurrency(status);
  return `${symbol}${(amount * rate).toFixed(digits)}`;
};
