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
import {
  clampNumber,
  createDefaultComboPricingConfig,
  createDefaultPricingRule,
} from '../utils';

export const useProfitBoardConfig = ({
  batchPayload,
  comboConfigs,
  setComboConfigs,
  restoredState,
}) => {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [options, setOptions] = useState({
    channels: [],
    tags: [],
    groups: [],
    local_models: [],
    site_models: [],
    upstream_accounts: [],
  });
  const [siteConfig, setSiteConfig] = useState(restoredState.siteConfig || {});
  const [upstreamConfig, setUpstreamConfig] = useState(
    restoredState.upstreamConfig || {},
  );

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

  const configLookupKey = useMemo(
    () => JSON.stringify(batchPayload),
    [batchPayload],
  );

  const configPayload = useMemo(
    () => ({
      batches: batchPayload,
      shared_site: {
        model_names: siteConfig.model_names || [],
        group: siteConfig.group || '',
        use_recharge_price: !!siteConfig.use_recharge_price,
      },
      combo_configs: comboConfigs,
      upstream: { ...upstreamConfig, fixed_amount: 0 },
      site: { ...siteConfig, fixed_amount: 0 },
    }),
    [batchPayload, comboConfigs, siteConfig, upstreamConfig],
  );

  const walletModeEnabled = upstreamConfig.upstream_mode === 'wallet_observer';

  const selectedAccount = useMemo(
    () =>
      (options.upstream_accounts || []).find(
        (item) => item.id === Number(upstreamConfig.upstream_account_id || 0),
      ) || null,
    [options.upstream_accounts, upstreamConfig.upstream_account_id],
  );

  const loadOptions = useCallback(async () => {
    const res = await API.get('/api/profit_board/options');
    if (!res.data.success) throw new Error(res.data.message || '加载选项失败');
    setOptions(
      res.data.data || {
        channels: [],
        tags: [],
        groups: [],
        local_models: [],
        site_models: [],
        upstream_accounts: [],
      },
    );
  }, []);

  const loadConfig = useCallback(async () => {
    if (!configLookupKey || configLookupKey === '[]') return;
    const res = await API.post('/api/profit_board/config/lookup', {
      batches: batchPayload,
    });
    if (!res.data.success) throw new Error(res.data.message || '加载配置失败');
    const config = res.data.data?.config;
    if (!config) return null;
    setSiteConfig((prev) => ({
      ...prev,
      ...(config.shared_site || {}),
      model_names: config.shared_site?.model_names || [],
    }));
    setUpstreamConfig((prev) => ({ ...prev, ...(config.upstream || {}) }));
    setComboConfigs(
      (config.combo_configs || []).map((item) => ({
        ...createDefaultComboPricingConfig(
          item.combo_id || '',
          item.shared_site || config.shared_site,
          config.site,
          config.upstream,
        ),
        ...item,
        site_rules: (item.site_rules || []).map((rule) =>
          createDefaultPricingRule(rule),
        ),
        upstream_rules: (item.upstream_rules || []).map((rule) =>
          createDefaultPricingRule(rule),
        ),
      })),
    );
    return true;
  }, [batchPayload, configLookupKey, setComboConfigs]);

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
          setSiteConfig((prev) => ({
            ...prev,
            ...(savedConfig.shared_site || {}),
            model_names: savedConfig.shared_site?.model_names || [],
          }));
          setUpstreamConfig((prev) => ({
            ...prev,
            ...(savedConfig.upstream || {}),
          }));
          setComboConfigs(
            (savedConfig.combo_configs || []).map((item) => ({
              ...createDefaultComboPricingConfig(
                item.combo_id || '',
                item.shared_site || savedConfig.shared_site,
                savedConfig.site,
                savedConfig.upstream,
              ),
              ...item,
              site_rules: (item.site_rules || []).map((rule) =>
                createDefaultPricingRule(rule),
              ),
              upstream_rules: (item.upstream_rules || []).map((rule) =>
                createDefaultPricingRule(rule),
              ),
            })),
          );
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
    [configPayload, setComboConfigs],
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
      const factor = currentSharedSite.use_recharge_price
        ? clampNumber(model.model_price || 1)
        : 1;
      const baseInput = clampNumber(model.model_ratio) * 2 * factor;
      return {
        input_price: baseInput,
        output_price:
          clampNumber(model.model_ratio) *
          clampNumber(model.completion_ratio) *
          2 *
          factor,
        cache_read_price: model.supports_cache_read
          ? baseInput * clampNumber(model.cache_ratio)
          : 0,
        cache_creation_price: model.supports_cache_creation
          ? baseInput * clampNumber(model.cache_creation_ratio)
          : 0,
      };
    },
    [localModelMap],
  );

  return {
    loading,
    setLoading,
    saving,
    options,
    siteConfig,
    setSiteConfig,
    upstreamConfig,
    setUpstreamConfig,
    channelOptions,
    channelMap,
    tagChannelMap,
    localModelMap,
    modelNameOptions,
    configLookupKey,
    configPayload,
    walletModeEnabled,
    selectedAccount,
    loadOptions,
    loadConfig,
    saveConfig,
    resolveSharedSitePreview,
  };
};
