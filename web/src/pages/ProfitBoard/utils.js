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

export const STORAGE_KEY = 'profit-board:state';
export const REPORT_CACHE_KEY = 'profit-board:report';
export const PROFIT_BOARD_CACHE_VERSION = 1;
export const DETAIL_LIMIT = 600;

export const createMetricOptions = (t) => [
  { value: 'configured_profit_usd', label: t('利润') },
  { value: 'actual_profit_usd', label: t('实际利润') },
  { value: 'actual_site_revenue_usd', label: t('本站实际收入') },
  { value: 'configured_site_revenue_usd', label: t('本站配置收入') },
  { value: 'upstream_cost_usd', label: t('上游费用') },
  { value: 'remote_observed_cost_usd', label: t('上游实际消耗') },
];

/** @deprecated Use createMetricOptions(t) instead */
export const metricOptions = [
  { value: 'configured_profit_usd', label: '利润' },
  { value: 'actual_profit_usd', label: '实际利润' },
  { value: 'actual_site_revenue_usd', label: '本站实际收入' },
  { value: 'configured_site_revenue_usd', label: '本站配置收入' },
  { value: 'upstream_cost_usd', label: '上游费用' },
  { value: 'remote_observed_cost_usd', label: '上游实际消耗' },
];

export const createSitePricingSourceLabelMap = (t) => ({
  manual: t('手动价格'),
  manual_rule: t('手动价格'),
  manual_default: t('手动默认规则'),
  manual_fallback: t('手动价格回退'),
  site_model_standard: t('读取本站模型原价'),
  site_model_recharge: t('读取本站模型充值价'),
  site_model_missing: t('未命中本站模型'),
});

/** @deprecated Use createSitePricingSourceLabelMap(t) instead */
export const sitePricingSourceLabelMap = {
  manual: '手动价格',
  manual_rule: '手动价格',
  manual_default: '手动默认规则',
  manual_fallback: '手动价格回退',
  site_model_standard: '读取本站模型原价',
  site_model_recharge: '读取本站模型充值价',
  site_model_missing: '未命中本站模型',
};

export const createBatchId = () =>
  `batch-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;

export const clampNumber = (value) => {
  const next = Number(value || 0);
  if (!Number.isFinite(next) || next < 0) return 0;
  return next;
};

export const createDefaultUpstreamConfig = () => ({
  cost_source: 'manual_only',
  upstream_mode: 'manual_rules',
  upstream_account_id: 0,
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  fixed_amount: 0,
  fixed_total_amount: 0,
});

export const createDefaultSiteConfig = () => ({
  pricing_mode: 'manual',
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  fixed_amount: 0,
  fixed_total_amount: 0,
  model_names: [],
  group: '',
  use_recharge_price: false,
});

export const createDefaultSharedSiteConfig = (overrides = {}) => ({
  model_names: [],
  group: '',
  use_recharge_price: false,
  ...overrides,
});

export const createDefaultPricingRule = (overrides = {}) => ({
  model_name: '',
  input_price: 0,
  output_price: 0,
  cache_read_price: 0,
  cache_creation_price: 0,
  is_default: false,
  is_custom: false,
  ...overrides,
});

export const createDefaultRemoteObserverConfig = () => ({
  enabled: false,
  base_url: '',
  user_id: 0,
  access_token: '',
  access_token_masked: '',
});

export const createDefaultComboPricingConfig = (
  comboId,
  sharedSite,
  legacySite,
  legacyUpstream,
) => {
  const walletMode = legacyUpstream?.upstream_mode === 'wallet_observer';

  return {
    combo_id: comboId,
    site_mode:
      legacySite?.pricing_mode === 'site_model'
        ? 'shared_site_model'
        : 'manual',
    upstream_mode: walletMode ? 'wallet_observer' : 'manual_rules',
    cost_source: legacyUpstream?.cost_source || 'manual_only',
    upstream_account_id: walletMode
      ? Number(legacyUpstream?.upstream_account_id || 0)
      : 0,
    shared_site: createDefaultSharedSiteConfig({
      model_names: sharedSite?.model_names || legacySite?.model_names || [],
      group: sharedSite?.group || legacySite?.group || '',
      use_recharge_price:
        typeof sharedSite?.use_recharge_price === 'boolean'
          ? sharedSite.use_recharge_price
          : !!legacySite?.use_recharge_price,
    }),
    site_rules: [
      createDefaultPricingRule({
        is_default: true,
        input_price: clampNumber(legacySite?.input_price),
        output_price: clampNumber(legacySite?.output_price),
        cache_read_price: clampNumber(legacySite?.cache_read_price),
        cache_creation_price: clampNumber(legacySite?.cache_creation_price),
      }),
    ],
    upstream_rules: [
      createDefaultPricingRule({
        is_default: true,
        input_price: clampNumber(legacyUpstream?.input_price),
        output_price: clampNumber(legacyUpstream?.output_price),
        cache_read_price: clampNumber(legacyUpstream?.cache_read_price),
        cache_creation_price: clampNumber(legacyUpstream?.cache_creation_price),
      }),
    ],
    site_fixed_total_amount: clampNumber(legacySite?.fixed_total_amount),
    upstream_fixed_total_amount: clampNumber(
      legacyUpstream?.fixed_total_amount,
    ),
    remote_observer: createDefaultRemoteObserverConfig(),
  };
};

const pickMostCommonValue = (values, fallbackValue) => {
  const stats = new Map();
  values.forEach((value) => {
    if (!value) return;
    const current = stats.get(value) || { value, count: 0 };
    current.count += 1;
    stats.set(value, current);
  });
  if (!stats.size) return fallbackValue;
  return Array.from(stats.values()).sort((left, right) => {
    if (right.count !== left.count) return right.count - left.count;
    return left.value.localeCompare(right.value);
  })[0].value;
};

export const createSuggestedComboName = (
  draft,
  channelMap,
  t,
  fallbackName = '',
) => {
  if (!draft) return fallbackName;

  const isTagScope = draft.scope_type === 'tag';
  const rawLabels = isTagScope
    ? draft.tags || []
    : (draft.channel_ids || []).map(
        (id) => channelMap.get(String(id))?.name || `#${id}`,
      );
  const labels = rawLabels.filter(Boolean);

  if (!labels.length) {
    return fallbackName || (isTagScope ? t('标签组合') : t('渠道组合'));
  }

  const preview = labels.slice(0, 2).join(' / ');
  const suffix = `${labels.length} ${isTagScope ? t('标签') : t('渠道')}`;
  return `${preview} · ${suffix}`;
};

export const isLikelyAutoComboName = (name, suggestedName = '') => {
  const trimmedName = name?.trim() || '';
  if (!trimmedName) return true;
  if (suggestedName && trimmedName === suggestedName) return true;
  return /^组合 \d+$/.test(trimmedName);
};

export const pickDominantComboModes = (
  comboConfigs,
  fallbackSiteMode = 'manual',
  fallbackUpstreamMode = 'manual_rules',
) => ({
  site_mode: pickMostCommonValue(
    (comboConfigs || []).map((item) => item.site_mode),
    fallbackSiteMode,
  ),
  upstream_mode: pickMostCommonValue(
    (comboConfigs || []).map((item) => item.upstream_mode),
    fallbackUpstreamMode,
  ),
});

export const pickRecommendedUpstreamAccountId = (
  comboConfigs,
  availableAccountIds,
  excludeComboId = '',
) => {
  const stats = new Map();
  (comboConfigs || []).forEach((item) => {
    if (!item || item.combo_id === excludeComboId) return;
    if (item.upstream_mode !== 'wallet_observer') return;
    const accountId = Number(item.upstream_account_id || 0);
    if (accountId <= 0 || !availableAccountIds.has(accountId)) return;
    const current = stats.get(accountId) || { id: accountId, count: 0 };
    current.count += 1;
    stats.set(accountId, current);
  });
  if (!stats.size) return 0;
  return Array.from(stats.values()).sort((left, right) => {
    if (right.count !== left.count) return right.count - left.count;
    return left.id - right.id;
  })[0].id;
};

export const mergeComboDraftWithTemplate = (draft, templateConfig) => {
  if (!draft || !templateConfig) return draft;

  return {
    ...draft,
    site_mode: templateConfig.site_mode || draft.site_mode,
    upstream_mode: templateConfig.upstream_mode || draft.upstream_mode,
    upstream_account_id: Number(templateConfig.upstream_account_id || 0),
    shared_site: createDefaultSharedSiteConfig({
      ...(templateConfig.shared_site || {}),
    }),
    site_rules: (templateConfig.site_rules || []).map((rule) =>
      createDefaultPricingRule(rule),
    ),
    upstream_rules: (templateConfig.upstream_rules || []).map((rule) =>
      createDefaultPricingRule(rule),
    ),
    site_fixed_total_amount: clampNumber(
      templateConfig.site_fixed_total_amount,
    ),
    upstream_fixed_total_amount: clampNumber(
      templateConfig.upstream_fixed_total_amount,
    ),
  };
};

export const createDefaultDraft = () => ({
  id: '',
  name: '',
  scope_type: 'channel',
  channel_ids: [],
  tags: [],
});

export const createDefaultUpstreamAccountDraft = () => ({
  id: 0,
  name: '',
  remark: '',
  account_type: 'newapi',
  base_url: '',
  user_id: 0,
  access_token: '',
  access_token_masked: '',
  resource_display_mode: 'both',
  low_balance_threshold_usd: 0,
  enabled: true,
});

export const normalizeUpstreamAccountResourceDisplayMode = (value) => {
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
  return {
    ...draft,
    name:
      String(draft?.name || '').trim() ||
      (allowSuggestedName ? suggestedName : ''),
    remark: String(draft?.remark || '').trim(),
    base_url: baseUrl,
    access_token: String(draft?.access_token || '').trim(),
    user_id: Number(draft?.user_id || 0),
    resource_display_mode: normalizeUpstreamAccountResourceDisplayMode(
      draft?.resource_display_mode,
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
  if (!Number.isInteger(prepared.user_id) || prepared.user_id <= 0) {
    errors.user_id = '请输入有效的用户 ID';
  }
  if (
    !prepared.access_token &&
    !String(prepared?.access_token_masked || '').trim()
  ) {
    errors.access_token = '请输入 access token';
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
    badgeTone: 'bg-red-500/10 text-red-700 dark:bg-red-500/15 dark:text-red-300',
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
      value: overrides.value ?? formatMoney(account?.wallet_balance_usd, status),
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
  const topAccent = metrics
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
    badgeTone: walletMetric?.badgeTone || 'bg-semi-color-fill-1 text-semi-color-text-1',
    noticeTone: statusBar?.tone || '',
    showNotice: !!statusBar,
  };
};

export const getAccountResourceSummaryTones = (account) => ({
  wallet: getResourceRiskTone(getWalletRiskMeta(account).level).amountTone,
  subscription: getResourceRiskTone(
    getSubscriptionRiskMeta(account).level,
  ).amountTone,
});

export const createDefaultState = () => {
  const end = new Date();
  const start = dayjs(end).subtract(7, 'day').toDate();
  return {
    batches: [],
    draft: createDefaultDraft(),
    editingBatchId: '',
    dateRange: [start, end],
    granularity: 'day',
    customIntervalMinutes: 15,
    chartTab: 'trend',
    channelGroupMode: 'channel',
    compareMode: 'none',
    comparePeriod: 'previous',
    compareDateRange: [],
    metricKey: 'configured_profit_usd',
    analysisMode: 'business_compare',
    viewBatchId: 'all',
    comboConfigs: [],
    upstreamConfig: createDefaultUpstreamConfig(),
    siteConfig: createDefaultSiteConfig(),
    lastQueryKey: '',
    autoRefreshMode: false,
    hasUnsavedConfigChanges: false,
  };
};

export const safeParse = (raw, fallback) => {
  try {
    return raw ? JSON.parse(raw) : fallback;
  } catch (error) {
    return fallback;
  }
};

export const createBatchCreatedAt = () => dayjs().unix();

export const normalizeBatchForState = (batch, index) => ({
  id: batch?.id || createBatchId(),
  name: batch?.name || `组合 ${index + 1}`,
  scope_type: batch?.scope_type || 'channel',
  channel_ids: (batch?.channel_ids || []).map((item) => item.toString()),
  tags: batch?.tags || [],
  created_at: Number(batch?.created_at || createBatchCreatedAt()),
});

export const normalizeRestoredState = (state) => {
  const defaults = createDefaultState();
  const next = { ...defaults, ...(state || {}) };
  const legacyHasSelection =
    !next.batches?.length &&
    ((next.scopeType === 'channel' &&
      (next.selectedChannels || []).length > 0) ||
      (next.scopeType === 'tag' && (next.selectedTags || []).length > 0));

  if (legacyHasSelection) {
    next.batches = [
      normalizeBatchForState(
        {
          id: createBatchId(),
          name: '组合 1',
          scope_type: next.scopeType || 'channel',
          channel_ids: next.selectedChannels || [],
          tags: next.selectedTags || [],
        },
        0,
      ),
    ];
  } else {
    next.batches = (next.batches || []).map((item, index) =>
      normalizeBatchForState(item, index),
    );
  }

  const [start, end] = next.dateRange || [];
  next.dateRange = [
    start ? new Date(start) : defaults.dateRange[0],
    end ? new Date(end) : defaults.dateRange[1],
  ];
  const [compareStart, compareEnd] = next.compareDateRange || [];
  next.compareDateRange = [
    compareStart ? new Date(compareStart) : null,
    compareEnd ? new Date(compareEnd) : null,
  ].filter(Boolean);
  next.draft = normalizeBatchForState(next.draft || {}, 0);
  next.editingBatchId = next.editingBatchId || '';
  next.customIntervalMinutes = Math.max(
    Number(next.customIntervalMinutes || defaults.customIntervalMinutes),
    1,
  );
  next.channelGroupMode = next.channelGroupMode === 'tag' ? 'tag' : 'channel';
  next.compareMode = next.compareMode || 'none';
  next.comparePeriod = next.comparePeriod || 'previous';
  next.upstreamConfig = {
    ...createDefaultUpstreamConfig(),
    ...(next.upstreamConfig || {}),
    upstream_mode: next.upstreamConfig?.upstream_mode || 'manual_rules',
    fixed_amount: 0,
  };
  if (next.upstreamConfig.upstream_mode !== 'wallet_observer') {
    next.upstreamConfig.upstream_account_id = 0;
  }
  next.siteConfig = {
    ...createDefaultSiteConfig(),
    ...(next.siteConfig || {}),
    model_names: next.siteConfig?.model_names || [],
    fixed_amount: 0,
  };
  next.comboConfigs = (next.comboConfigs || []).map((item) => ({
    ...createDefaultComboPricingConfig(
      item?.combo_id || '',
      item?.shared_site,
      next.siteConfig,
      next.upstreamConfig,
    ),
    ...item,
    shared_site: createDefaultSharedSiteConfig(item?.shared_site || {}),
    site_rules: (item?.site_rules || []).map((rule) =>
      createDefaultPricingRule(rule),
    ),
    upstream_rules: (item?.upstream_rules || []).map((rule) =>
      createDefaultPricingRule(rule),
    ),
    remote_observer: {
      ...createDefaultRemoteObserverConfig(),
      ...(item?.remote_observer || {}),
    },
  }));
  next.viewBatchId = next.viewBatchId || 'all';
  next.lastQueryKey = next.lastQueryKey || '';
  next.analysisMode = next.analysisMode || 'business_compare';
  next.autoRefreshMode = !!next.autoRefreshMode;
  next.hasUnsavedConfigChanges = !!next.hasUnsavedConfigChanges;
  return next;
};

export const getUpstreamCostSourceLabel = (costSource, t) => {
  switch (costSource) {
    case 'returned_cost_first':
      return t('优先用上游返回费用');
    case 'returned_cost_only':
      return t('只用上游返回费用');
    case 'manual_only':
    default:
      return t('只用手动成本规则');
  }
};

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

export const createPresetRanges = (t) => {
  const now = dayjs();
  const label = typeof t === 'function' ? t : (s) => s;
  return [
    {
      label: label('今天'),
      value: [now.startOf('day').toDate(), now.endOf('day').toDate()],
    },
    {
      label: label('最近 24 小时'),
      value: [now.subtract(24, 'hour').toDate(), now.toDate()],
    },
    {
      label: label('近 7 天'),
      value: [now.subtract(7, 'day').toDate(), now.toDate()],
    },
    {
      label: label('近 30 天'),
      value: [now.subtract(30, 'day').toDate(), now.toDate()],
    },
    {
      label: label('本月'),
      value: [now.startOf('month').toDate(), now.endOf('month').toDate()],
    },
    {
      label: label('上月'),
      value: [
        now.subtract(1, 'month').startOf('month').toDate(),
        now.subtract(1, 'month').endOf('month').toDate(),
      ],
    },
  ];
};

export const formatRangeLabel = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return '-';
  return `${dayjs(range[0]).format('YYYY-MM-DD HH:mm')} ~ ${dayjs(range[1]).format('YYYY-MM-DD HH:mm')}`;
};

const hoursToFixed = (minutes) => (minutes % 60 === 0 ? 0 : 1);

export const formatRangeDuration = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return '-';
  const minutes = dayjs(range[1]).diff(dayjs(range[0]), 'minute');
  if (minutes < 60) return `${minutes} 分钟`;
  const hours = (minutes / 60).toFixed(hoursToFixed(minutes));
  if (minutes < 1440) return `${hours} 小时`;
  return `${(minutes / 1440).toFixed(1)} 天`;
};

export const buildQueryKey = (payload) => JSON.stringify(payload);

export const normalizeCachedReportBundle = (raw) => {
  if (!raw) return null;
  if (raw.report) {
    return {
      report: raw.report,
      queryKey: raw.queryKey || '',
      activityWatermark:
        raw.activityWatermark || raw.report?.meta?.activity_watermark || '',
      generatedAt: raw.generatedAt || raw.report?.meta?.generated_at || 0,
    };
  }
  return {
    report: raw,
    queryKey: '',
    activityWatermark: raw?.meta?.activity_watermark || '',
    generatedAt: raw?.meta?.generated_at || 0,
  };
};

export const formatRatio = (value) =>
  `${(Number(value || 0) * 100).toFixed(1)}%`;

export const formatBucketLabel = (
  timestamp,
  granularity,
  customIntervalMinutes,
) => {
  const current = dayjs.unix(timestamp);
  if (granularity === 'hour') return current.format('YYYY-MM-DD HH:00');
  if (granularity === 'week')
    return current.startOf('week').add(1, 'day').format('GGGG-[W]WW');
  if (granularity === 'month')
    return current.startOf('month').format('YYYY-MM');
  if (granularity === 'custom') {
    const interval = Math.max(Number(customIntervalMinutes || 1), 1);
    const totalMinutes = current.hour() * 60 + current.minute();
    const alignedMinutes = Math.floor(totalMinutes / interval) * interval;
    return current
      .startOf('day')
      .add(alignedMinutes, 'minute')
      .format('YYYY-MM-DD HH:mm');
  }
  return current.format('YYYY-MM-DD');
};

export const buildPreviousPeriodRange = (range) => {
  if (!Array.isArray(range) || !range[0] || !range[1]) return [];
  const start = dayjs(range[0]);
  const end = dayjs(range[1]);
  const duration = Math.max(end.diff(start, 'second'), 0);
  const previousEnd = start.subtract(1, 'second');
  const previousStart = previousEnd.subtract(duration, 'second');
  return [previousStart.toDate(), previousEnd.toDate()];
};

export const aggregateBreakdownRows = (rows, viewBatchId, metricKey) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();
  filtered.forEach((item) => {
    const key = item.label || item.key;
    const current = grouped.get(key) || { label: key, value: 0 };
    current.value += Number(item[metricKey] || 0);
    grouped.set(key, current);
  });
  return Array.from(grouped.values())
    .sort((a, b) => b.value - a.value)
    .slice(0, 12);
};

export const aggregateChannelRowsByTag = (
  rows,
  viewBatchId,
  metricKey,
  channelTagMap,
  emptyTagLabel,
) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();

  filtered.forEach((item) => {
    const tagLabel = channelTagMap.get(String(item.key)) || emptyTagLabel;
    const current = grouped.get(tagLabel) || {
      label: tagLabel,
      key: tagLabel,
      value: 0,
      batch_id: item.batch_id || null,
    };
    current.value += Number(item[metricKey] || 0);
    if (current.batch_id && current.batch_id !== item.batch_id) {
      current.batch_id = null;
    }
    grouped.set(tagLabel, current);
  });

  return Array.from(grouped.values())
    .sort((a, b) => b.value - a.value)
    .slice(0, 12);
};

export const combineTimeseriesMetrics = (rows, viewBatchId, metrics) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();
  filtered.forEach((item) => {
    metrics.forEach((metric) => {
      const key = `${item.bucket}::${metric.key}`;
      const current = grouped.get(key) || {
        bucket: item.bucket,
        value: 0,
        series: metric.label,
      };
      current.value += Number(item[metric.key] || 0);
      grouped.set(key, current);
    });
  });
  return Array.from(grouped.values()).sort((a, b) => {
    if (a.bucket === b.bucket) return a.series.localeCompare(b.series);
    return a.bucket.localeCompare(b.bucket);
  });
};

export const combineBreakdownMetrics = (rows, viewBatchId, metrics) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();
  filtered.forEach((item) => {
    const label = item.label || item.key;
    metrics.forEach((metric) => {
      const key = `${label}::${metric.key}`;
      const current = grouped.get(key) || {
        label,
        value: 0,
        series: metric.label,
      };
      current.value += Number(item[metric.key] || 0);
      grouped.set(key, current);
    });
  });
  return Array.from(grouped.values())
    .sort((a, b) => {
      if (a.label === b.label) return a.series.localeCompare(b.series);
      return b.value - a.value;
    })
    .slice(0, 24);
};

export const combineChannelMetricsByTag = (
  rows,
  viewBatchId,
  metrics,
  channelTagMap,
  emptyTagLabel,
) => {
  const filtered =
    viewBatchId === 'all'
      ? rows || []
      : (rows || []).filter((item) => item.batch_id === viewBatchId);
  const grouped = new Map();

  filtered.forEach((item) => {
    const tagLabel = channelTagMap.get(String(item.key)) || emptyTagLabel;
    metrics.forEach((metric) => {
      const key = `${tagLabel}::${metric.key}`;
      const current = grouped.get(key) || {
        label: tagLabel,
        key: tagLabel,
        value: 0,
        series: metric.label,
        batch_id: item.batch_id || null,
      };
      current.value += Number(item[metric.key] || 0);
      if (current.batch_id && current.batch_id !== item.batch_id) {
        current.batch_id = null;
      }
      grouped.set(key, current);
    });
  });

  return Array.from(grouped.values())
    .sort((a, b) => {
      if (a.label === b.label) return a.series.localeCompare(b.series);
      return b.value - a.value;
    })
    .slice(0, 24);
};

export const createTrendSpec = (rows, metricLabel, status, t) => {
  const isSparse = rows.length <= 3;
  const hasSeries = rows.some((item) => item.series);
  return {
    type: 'line',
    background: 'transparent',
    data: [{ id: 'trend', values: rows }],
    xField: 'bucket',
    yField: 'value',
    seriesField: hasSeries ? 'series' : undefined,
    legends: { visible: hasSeries },
    point: {
      visible: true,
      style: { size: isSparse ? 10 : 5 },
    },
    line: {
      style: { curveType: 'monotone', lineWidth: isSparse ? 3 : 2 },
    },
    label: isSparse
      ? {
          visible: true,
          position: 'top',
          formatter: (datum) => formatMoney(datum.value, status),
          style: { fontSize: 12 },
        }
      : { visible: false },
    axes: [
      {
        orient: 'bottom',
        type: 'band',
        label: { visible: true, style: { angle: -18 } },
      },
      { orient: 'left', nice: true },
    ],
    title: { visible: true, text: t('收益趋势'), subtext: metricLabel },
    tooltip: {
      mark: {
        content: [
          { key: t('时间桶'), value: (datum) => datum.bucket },
          ...(hasSeries
            ? [{ key: t('组合'), value: (datum) => datum.series }]
            : []),
          {
            key: metricLabel,
            value: (datum) => formatMoney(datum.value, status),
          },
        ],
      },
    },
  };
};

export const createBarSpec = (title, rows, metricLabel, status, t) => ({
  type: 'bar',
  background: 'transparent',
  data: [{ id: 'bar', values: rows }],
  xField: 'label',
  yField: 'value',
  seriesField: rows.some((item) => item.series) ? 'series' : undefined,
  legends: { visible: rows.some((item) => item.series) },
  axes: [
    {
      orient: 'bottom',
      type: 'band',
      label: { visible: true, style: { angle: -20 } },
    },
    { orient: 'left', nice: true },
  ],
  title: { visible: true, text: title, subtext: metricLabel },
  tooltip: {
    mark: {
      content: [
        { key: t('名称'), value: (datum) => datum.label },
        ...(rows.some((item) => item.series)
          ? [{ key: t('对比项'), value: (datum) => datum.series }]
          : []),
        {
          key: metricLabel,
          value: (datum) => formatMoney(datum.value, status),
        },
      ],
    },
  },
});

export const createAccountUsageTrendSpec = (rows, status, t) => ({
  type: 'line',
  background: 'transparent',
  height: 240,
  padding: { top: 24, right: 18, bottom: 36, left: 48 },
  data: [{ id: 'wallet-trend', values: rows }],
  xField: 'bucket',
  yField: 'period_used_usd',
  point: { visible: true, style: { size: rows.length <= 4 ? 7 : 5 } },
  line: { style: { curveType: 'monotone', lineWidth: 2.5 } },
  axes: [
    {
      orient: 'bottom',
      type: 'band',
      label: { visible: true, style: { angle: rows.length > 5 ? -18 : 0 } },
    },
    {
      orient: 'left',
      nice: true,
      label: {
        formatter: (value) => formatMoney(value, status, 2),
      },
    },
  ],
  title: {
    visible: true,
    text: t('近 7 天已用趋势'),
    subtext: t('按同步快照增量统计'),
  },
  tooltip: {
    mark: {
      content: [
        { key: t('时间'), value: (datum) => datum.bucket },
        {
          key: t('近 7 天已用'),
          value: (datum) => formatMoney(datum.period_used_usd, status, 3),
        },
      ],
    },
  },
});
