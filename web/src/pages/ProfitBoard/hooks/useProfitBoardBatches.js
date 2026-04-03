import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { showError, showSuccess } from '../../../helpers';
import {
  createDefaultComboPricingConfig,
  createDefaultDraft,
  createDefaultPricingRule,
  normalizeBatchForState,
} from '../utils';

export const useProfitBoardBatches = ({
  restoredState,
  channelMap,
  tagChannelMap,
  siteConfig,
  upstreamConfig,
}) => {
  const { t } = useTranslation();
  const [batches, setBatches] = useState(restoredState.batches || []);
  const [draft, setDraft] = useState(
    restoredState.draft || createDefaultDraft(),
  );
  const [editingBatchId, setEditingBatchId] = useState(
    restoredState.editingBatchId || '',
  );
  const [comboConfigs, setComboConfigs] = useState(
    restoredState.comboConfigs || [],
  );

  const batchPayload = useMemo(
    () =>
      batches.map((batch) => ({
        id: batch.id,
        name: batch.name?.trim() || t('未命名组合'),
        scope_type: batch.scope_type,
        channel_ids: (batch.channel_ids || [])
          .map((item) => Number(item))
          .filter(Boolean),
        tags: batch.tags || [],
      })),
    [batches, t],
  );

  const duplicateBatchError = useMemo(() => {
    const ownerMap = new Map();
    for (const batch of batches) {
      const channelIds =
        batch.scope_type === 'tag'
          ? (batch.tags || []).flatMap((tag) => tagChannelMap.get(tag) || [])
          : batch.channel_ids || [];
      for (const channelId of Array.from(new Set(channelIds))) {
        const owner = ownerMap.get(channelId);
        if (owner && owner !== batch.name) {
          const channelName =
            channelMap.get(String(channelId))?.name || `#${channelId}`;
          return `${channelName} 同时出现在组合"${owner}"和"${batch.name}"中，请拆开后再统计`;
        }
        ownerMap.set(channelId, batch.name);
      }
    }
    return '';
  }, [batches, channelMap, tagChannelMap]);

  const validationErrors = useMemo(() => {
    const errors = [];
    if (!batches.length) errors.push(t('请至少添加一个组合'));
    if (duplicateBatchError) errors.push(duplicateBatchError);
    if (
      comboConfigs.some(
        (item) =>
          item.site_mode === 'shared_site_model' &&
          !(item.shared_site?.model_names || []).length,
      )
    ) {
      errors.push(t('启用了本站模型价格的组合必须至少选择一个模型'));
    }
    if (
      upstreamConfig.upstream_mode === 'wallet_observer' &&
      !Number(upstreamConfig.upstream_account_id || 0)
    ) {
      errors.push(t('钱包扣减模式必须绑定一个上游账户'));
    }
    return errors;
  }, [
    batches.length,
    comboConfigs,
    duplicateBatchError,
    upstreamConfig.upstream_account_id,
    upstreamConfig.upstream_mode,
    t,
  ]);

  // Sync comboConfigs when batches change
  useEffect(() => {
    setComboConfigs((prev) =>
      batches.map((batch) => {
        const existing = (prev || []).find(
          (item) => item.combo_id === batch.id,
        );
        const fallback = createDefaultComboPricingConfig(
          batch.id,
          siteConfig,
          siteConfig,
          upstreamConfig,
        );
        return {
          ...fallback,
          ...(existing || {}),
          combo_id: batch.id,
          site_rules: (existing?.site_rules || fallback.site_rules || []).map(
            (rule) => createDefaultPricingRule(rule),
          ),
          upstream_rules: (
            existing?.upstream_rules ||
            fallback.upstream_rules ||
            []
          ).map((rule) => createDefaultPricingRule(rule)),
        };
      }),
    );
  }, [batches, siteConfig, upstreamConfig]);

  const addOrUpdateBatch = useCallback(() => {
    const nextBatch = {
      id: editingBatchId || normalizeBatchForState({}, batches.length).id,
      name:
        draft.name?.trim() ||
        `组合 ${batches.length + (editingBatchId ? 0 : 1)}`,
      scope_type: draft.scope_type,
      channel_ids:
        draft.scope_type === 'channel' ? draft.channel_ids || [] : [],
      tags: draft.scope_type === 'tag' ? draft.tags || [] : [],
    };
    const selectedCount =
      nextBatch.scope_type === 'channel'
        ? nextBatch.channel_ids.length
        : nextBatch.tags.length;
    if (!selectedCount) return showError(t('请先选择渠道或标签'));
    setBatches((prev) =>
      editingBatchId
        ? prev.map((item) => (item.id === editingBatchId ? nextBatch : item))
        : [...prev, nextBatch],
    );
    setDraft(createDefaultDraft());
    setEditingBatchId('');
    showSuccess(editingBatchId ? t('组合已更新') : t('组合已添加'));
  }, [batches.length, draft, editingBatchId, t]);

  const editBatch = useCallback((batch) => {
    setEditingBatchId(batch.id);
    setDraft({
      id: batch.id,
      name: batch.name,
      scope_type: batch.scope_type,
      channel_ids: batch.channel_ids || [],
      tags: batch.tags || [],
    });
  }, []);

  const resetDraft = useCallback(() => {
    setDraft(createDefaultDraft());
    setEditingBatchId('');
  }, []);

  const removeBatch = useCallback(
    (batchId) => {
      setBatches((prev) => prev.filter((item) => item.id !== batchId));
      setComboConfigs((prev) =>
        prev.filter((item) => item.combo_id !== batchId),
      );
      if (editingBatchId === batchId) resetDraft();
    },
    [editingBatchId, resetDraft],
  );

  const updateComboConfig = useCallback(
    (comboId, updater) =>
      setComboConfigs((prev) =>
        prev.map((item) =>
          item.combo_id === comboId
            ? {
                ...item,
                ...(typeof updater === 'function' ? updater(item) : updater),
              }
            : item,
        ),
      ),
    [],
  );

  const addComboRule = useCallback(
    (comboId, field, initialRule = {}) =>
      updateComboConfig(comboId, (current) => ({
        [field]: [
          ...(current[field] || []),
          createDefaultPricingRule(initialRule),
        ],
      })),
    [updateComboConfig],
  );

  const updateComboRule = useCallback(
    (comboId, field, index, patch) =>
      updateComboConfig(comboId, (current) => ({
        [field]: (current[field] || []).map((item, itemIndex) =>
          itemIndex === index ? { ...item, ...patch } : item,
        ),
      })),
    [updateComboConfig],
  );

  const removeComboRule = useCallback(
    (comboId, field, index) =>
      updateComboConfig(comboId, (current) => ({
        [field]: (current[field] || []).filter(
          (_, itemIndex) => itemIndex !== index,
        ),
      })),
    [updateComboConfig],
  );

  const batchDigest = useCallback(
    (batch) =>
      batch.scope_type === 'channel'
        ? (batch.channel_ids || [])
            .map((id) => channelMap.get(String(id))?.name)
            .filter(Boolean)
            .slice(0, 3)
            .join('、')
        : (batch.tags || []).slice(0, 3).join('、'),
    [channelMap],
  );

  return {
    batches,
    setBatches,
    draft,
    setDraft,
    editingBatchId,
    comboConfigs,
    setComboConfigs,
    batchPayload,
    duplicateBatchError,
    validationErrors,
    addOrUpdateBatch,
    editBatch,
    resetDraft,
    removeBatch,
    updateComboConfig,
    addComboRule,
    updateComboRule,
    removeComboRule,
    batchDigest,
  };
};
