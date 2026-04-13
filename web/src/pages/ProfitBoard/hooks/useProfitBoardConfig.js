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
import { useCallback, useMemo, useState } from 'react';
import { API, showError, showSuccess } from '../../../helpers';
import { getQuotaPerUnit } from '../../../helpers/quota';
import {
  clampNumber,
  clampPositiveNumber,
  computePackageEffectiveRate,
  createDefaultComboPricingConfig,
  createDefaultPricingRule,
  normalizeSubscriptionPlans,
} from '../utils';

export const useProfitBoardConfig = ({
  batchPayload,
  comboConfigs,
  setComboConfigs,
  restoredState,
  rechargePriceFactor = 1,
  usdExchangeRate = 0,
}) => {
  const [builderLoading, setBuilderLoading] = useState(false);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [options, setOptions] = useState({
    channels: [],
    tags: [],
    groups: [],
    local_models: [],
    site_models: [],
    admin_users: [],
    upstream_accounts: [],
  });
  const [siteConfig, setSiteConfig] = useState(restoredState.siteConfig || {});
  const [excludedUserIDs, setExcludedUserIDs] = useState(
    restoredState.excludedUserIDs || [],
  );
  const [upstreamConfig, setUpstreamConfig] = useState(
    restoredState.upstreamConfig || {},
  );
  const [subscriptionPlans, setSubscriptionPlans] = useState([]);

  const channelOptions = useMemo(
    () =>
      (options.channels || []).map((item) => ({
        label: item.tag ? `${item.name} (${item.tag})` : item.name,
        value: String(item.id),
      })),
    [options.channels],
  );

  const channelMap = useMemo(
    () =>
      new Map((options.channels || []).map((item) => [String(item.id), item])),
    [options.channels],
  );

  const tagChannelMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((item) => {
      const tag = item.tag || '';
      if (!tag) return;
      const current = map.get(tag) || [];
      current.push(String(item.id));
      map.set(tag, current);
    });
    return map;
  }, [options.channels]);

  const channelModelMap = useMemo(() => {
    const map = new Map();
    (options.channels || []).forEach((item) => {
      const models = (item.models || '')
        .split(',')
        .map((m) => m.trim())
        .filter(Boolean);
      if (models.length > 0) {
        map.set(String(item.id), models);
      }
    });
    return map;
  }, [options.channels]);

  const getModelsByChannelIds = useCallback(
    (ids) => {
      const set = new Set();
      (ids || []).forEach((id) => {
        const models = channelModelMap.get(String(id));
        if (models) models.forEach((m) => set.add(m));
      });
      return Array.from(set).sort();
    },
    [channelModelMap],
  );

  const getModelsByTags = useCallback(
    (tags) => {
      const channelIds = new Set();
      (tags || []).forEach((tag) => {
        const ids = tagChannelMap.get(tag);
        if (ids) ids.forEach((id) => channelIds.add(id));
      });
      return getModelsByChannelIds(Array.from(channelIds));
    },
    [tagChannelMap, getModelsByChannelIds],
  );

  const localModelMap = useMemo(
    () =>
      new Map(
        (options.local_models || []).map((item) => [item.model_name, item]),
      ),
    [options.local_models],
  );

  const modelNameOptions = useMemo(
    () =>
      (options.site_models || []).map((item) => ({
        label: item,
        value: item,
      })),
    [options.site_models],
  );

  const configPayload = useMemo(
    () => ({
      batches: batchPayload,
      shared_site: {
        model_names: siteConfig.model_names || [],
        group: siteConfig.group || '',
        use_recharge_price: !!siteConfig.use_recharge_price,
        plan_id: siteConfig.plan_id || 0,
      },
      combo_configs: comboConfigs,
      excluded_user_ids: excludedUserIDs,
      upstream: { ...upstreamConfig, fixed_amount: 0 },
      site: { ...siteConfig, fixed_amount: 0 },
    }),
    [batchPayload, comboConfigs, excludedUserIDs, siteConfig, upstreamConfig],
  );

  const walletModeEnabled = upstreamConfig.upstream_mode === 'wallet_observer';

  const selectedAccount = useMemo(
    () =>
      (options.upstream_accounts || []).find(
        (item) => item.id === Number(upstreamConfig.upstream_account_id || 0),
      ) || null,
    [options.upstream_accounts, upstreamConfig.upstream_account_id],
  );

  const normalizeLoadedConfig = useCallback(
    (config) => ({
      siteConfig: {
        ...(config.shared_site || {}),
        model_names: config.shared_site?.model_names || [],
      },
      excludedUserIDs: (config.excluded_user_ids || []).map(Number).filter(Boolean),
      upstreamConfig: {
        ...(config.upstream || {}),
        cost_source: 'manual_only',
      },
      comboConfigs: (config.combo_configs || []).map((item) => ({
        ...createDefaultComboPricingConfig(
          item.combo_id || '',
          item.shared_site || config.shared_site,
          config.site,
          config.upstream,
        ),
        ...item,
        cost_source: 'manual_only',
        site_rules: (item.site_rules || []).map((rule) =>
          createDefaultPricingRule(rule),
        ),
        upstream_rules: (item.upstream_rules || []).map((rule) =>
          createDefaultPricingRule(rule),
        ),
        site_exchange_rate: clampPositiveNumber(item?.site_exchange_rate, 1),
        upstream_exchange_rate: clampPositiveNumber(
          item?.upstream_exchange_rate,
          1,
        ),
      })),
    }),
    [],
  );

  const applyLoadedConfig = useCallback(
    (config) => {
      if (!config) return;
      const next = normalizeLoadedConfig(config);
      // 直接替换，不做合并——服务器数据为最终来源
      setSiteConfig(next.siteConfig);
      setExcludedUserIDs(next.excludedUserIDs);
      setUpstreamConfig(next.upstreamConfig);
      setComboConfigs(next.comboConfigs);
    },
    [normalizeLoadedConfig, setComboConfigs],
  );

  const loadBuilderOptions = useCallback(async () => {
    setBuilderLoading(true);
    try {
      const res = await API.get('/api/profit_board/options');
      if (!res.data.success)
        throw new Error(res.data.message || '加载选项失败');
      setOptions(
        res.data.data || {
          channels: [],
          tags: [],
          groups: [],
          local_models: [],
          site_models: [],
          admin_users: [],
          upstream_accounts: [],
        },
      );
    } finally {
      setBuilderLoading(false);
    }
    // 非阻塞加载套餐列表
    loadSubscriptionPlans();
  }, []);

  const loadSubscriptionPlans = useCallback(async () => {
    try {
      const res = await API.get('/api/subscription/plans');
      if (res.data?.success) {
        setSubscriptionPlans(normalizeSubscriptionPlans(res.data.data));
      }
    } catch {
      // 未登录或接口不可用，静默忽略
    }
  }, []);

  const loadUpstreamAccounts = useCallback(async () => {
    setAccountsLoading(true);
    try {
      const res = await API.get('/api/profit_board/upstream_accounts');
      if (!res.data.success)
        throw new Error(res.data.message || '加载上游账户失败');
      setOptions((prev) => ({
        ...prev,
        upstream_accounts: res.data.data || [],
      }));
    } finally {
      setAccountsLoading(false);
    }
  }, []);

  const loadCurrentConfig = useCallback(async () => {
    const res = await API.get('/api/profit_board/config/current');
    if (!res.data.success)
      throw new Error(res.data.message || '加载收益看板配置失败');
    return res.data.data?.config || null;
  }, []);

  const saveConfig = useCallback(
    async (validationErrors) => {
      if (validationErrors.length > 0) {
        showError(validationErrors[0]);
        return false;
      }
      setSaving(true);
      try {
        const res = await API.put('/api/profit_board/config', configPayload);
        if (!res.data.success) {
          showError(res.data.message);
          return false;
        }
        const savedConfig = res.data.data?.config;
        if (savedConfig) {
          applyLoadedConfig(savedConfig);
        }
        showSuccess('收益看板配置已保存');
        return true;
      } catch (error) {
        showError(error);
        return false;
      } finally {
        setSaving(false);
      }
    },
    [applyLoadedConfig, configPayload],
  );

  const groupRatioMap = useMemo(
    () => options.group_ratios || {},
    [options.group_ratios],
  );

  const resolveSharedSitePreview = useCallback(
    (sharedSiteConfig, modelName) => {
      const model = localModelMap.get(modelName);
      if (!model) return null;
      const currentSharedSite = sharedSiteConfig || {};
      if (
        currentSharedSite.group &&
        (model.enable_groups || []).length > 0 &&
        !(model.enable_groups || []).includes(currentSharedSite.group)
      )
        return null;
      if (model.quota_type === 1)
        return {
          input_price: clampNumber(model.model_price),
          output_price: 0,
          cache_read_price: 0,
          cache_creation_price: 0,
        };

      // 分组倍率：选了具体分组用该倍率，否则从 enable_groups 取最低
      let usedGroupRatio = 1;
      const selectedGroup = currentSharedSite.group || '';
      if (selectedGroup) {
        usedGroupRatio = groupRatioMap[selectedGroup] ?? 1;
      } else if (Array.isArray(model.enable_groups) && model.enable_groups.length > 0) {
        let minRatio = Infinity;
        for (const g of model.enable_groups) {
          const r = groupRatioMap[g];
          if (r !== undefined && r < minRatio) {
            minRatio = r;
          }
        }
        if (minRatio < Infinity) {
          usedGroupRatio = minRatio;
        }
      }

      let factor;
      if (currentSharedSite.use_recharge_price) {
        factor = rechargePriceFactor;
      } else if (currentSharedSite.plan_id > 0 && usdExchangeRate > 0) {
        const plan = subscriptionPlans.find(
          (p) => Number(p.id) === Number(currentSharedSite.plan_id),
        );
        if (plan) {
          const quotaPerUnit = getQuotaPerUnit();
          const effectiveRate = computePackageEffectiveRate(
            plan,
            quotaPerUnit,
            usdExchangeRate,
          );
          factor = effectiveRate != null ? effectiveRate : 1;
        } else {
          factor = 1;
        }
      } else {
        factor = 1;
      }
      const baseInput = clampNumber(model.model_ratio) * 2 * usedGroupRatio * factor;
      return {
        input_price: baseInput,
        output_price:
          clampNumber(model.model_ratio) *
          clampNumber(model.completion_ratio) *
          2 *
          usedGroupRatio *
          factor,
        cache_read_price: model.supports_cache_read
          ? baseInput * clampNumber(model.cache_ratio)
          : 0,
        cache_creation_price: model.supports_cache_creation
          ? baseInput * clampNumber(model.cache_creation_ratio)
          : 0,
      };
    },
    [groupRatioMap, localModelMap, rechargePriceFactor, subscriptionPlans, usdExchangeRate],
  );

  return {
    builderLoading,
    accountsLoading,
    saving,
    options,
    siteConfig,
    setSiteConfig,
    excludedUserIDs,
    setExcludedUserIDs,
    upstreamConfig,
    setUpstreamConfig,
    channelOptions,
    channelMap,
    channelModelMap,
    tagChannelMap,
    localModelMap,
    modelNameOptions,
    configPayload,
    walletModeEnabled,
    selectedAccount,
    loadBuilderOptions,
    loadUpstreamAccounts,
    loadCurrentConfig,
    applyLoadedConfig,
    saveConfig,
    resolveSharedSitePreview,
    getModelsByChannelIds,
    getModelsByTags,
    subscriptionPlans,
  };
};
