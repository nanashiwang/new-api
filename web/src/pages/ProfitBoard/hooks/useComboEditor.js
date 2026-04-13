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

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Modal } from '@douyinfe/semi-ui';
import { showError, showSuccess } from '../../../helpers';
import {
  buildBatchOverlapError,
  clampNumber,
  createBatchId,
  createBatchCreatedAt,
  createDefaultComboPricingConfig,
  createDefaultPricingRule,
  createSuggestedComboName,
  isLikelyAutoComboName,
  mergeComboDraftWithTemplate,
  pickDominantComboModes,
  pickRecommendedUpstreamAccountId,
} from '../utils';

const cloneComboDraft = (batch, comboConfig) => ({
  id: batch.id,
  name: batch.name || '',
  scope_type: batch.scope_type || 'channel',
  channel_ids: [...(batch.channel_ids || [])],
  tags: [...(batch.tags || [])],
  created_at: Number(batch.created_at || createBatchCreatedAt()),
  combo_id: comboConfig.combo_id,
  site_mode: comboConfig.site_mode,
  upstream_mode: comboConfig.upstream_mode,
  cost_source: 'manual_only',
  upstream_account_id: Number(comboConfig.upstream_account_id || 0),
  shared_site: { ...(comboConfig.shared_site || {}) },
  site_rules: (comboConfig.site_rules || []).map((rule) =>
    createDefaultPricingRule(rule),
  ),
  upstream_rules: (comboConfig.upstream_rules || []).map((rule) =>
    createDefaultPricingRule(rule),
  ),
  site_fixed_total_amount: clampNumber(comboConfig.site_fixed_total_amount),
  upstream_fixed_total_amount: clampNumber(
    comboConfig.upstream_fixed_total_amount,
  ),
  remote_observer: { ...(comboConfig.remote_observer || {}) },
});

export const useComboEditor = ({
  batches,
  upsertBatch,
  removeBatch,
  comboConfigs,
  setComboConfigs,
  availableAccountIds,
  availableAccounts,
  channelMap,
  tagChannelMap,
  siteConfig,
  upstreamConfig,
  builderOptionsReady,
  setBuilderOptionsReady,
  loadBuilderOptions,
  resolveComboConfig,
  onConfigChanged,
  syncAccount,
  t,
}) => {
  const [editorDraft, setEditorDraft] = useState(null);
  const [editorNameAuto, setEditorNameAuto] = useState(false);
  const [editingBatchId, setEditingBatchId] = useState('');
  const [editorVisible, setEditorVisible] = useState(false);
  const [editorValidationError, setEditorValidationError] = useState('');

  const duplicateBatchError = useMemo(
    () => buildBatchOverlapError(batches, channelMap, tagChannelMap),
    [batches, channelMap, tagChannelMap],
  );

  const buildDraftValidationError = useCallback(
    (draft) => {
      if (!draft) return '';
      const selectedCount =
        draft.scope_type === 'channel'
          ? (draft.channel_ids || []).length
          : (draft.tags || []).length;
      if (!selectedCount) return t('请先选择渠道或标签');
      if (
        (draft.site_mode === 'shared_site_model' || draft.site_mode === 'log_quota') &&
        !(draft.shared_site?.model_names || []).length
      ) {
        return t('启用了本站模型价格或智能模式的组合必须至少选择一个模型');
      }
      if (draft.upstream_mode === 'wallet_observer') {
        const accountId = Number(draft.upstream_account_id || 0);
        if (accountId <= 0 || !availableAccountIds.has(accountId)) {
          return t('钱包扣减模式必须绑定一个上游账户');
        }
      }
      const nextBatch = {
        id: draft.id,
        name: draft.name?.trim() || t('未命名组合'),
        scope_type: draft.scope_type,
        channel_ids:
          draft.scope_type === 'channel' ? draft.channel_ids || [] : [],
        tags: draft.scope_type === 'tag' ? draft.tags || [] : [],
      };
      return buildBatchOverlapError(
        [...batches.filter((item) => item.id !== draft.id), nextBatch],
        channelMap,
        tagChannelMap,
      );
    },
    [availableAccountIds, batches, channelMap, t, tagChannelMap],
  );

  useEffect(() => {
    if (!editorDraft) {
      setEditorValidationError('');
      return;
    }
    setEditorValidationError(buildDraftValidationError(editorDraft));
  }, [buildDraftValidationError, editorDraft]);

  const getEditorSuggestedName = useCallback(
    (draft) => {
      const fallbackName = editingBatchId
        ? draft?.name?.trim() || t('未命名组合')
        : `组合 ${batches.length + 1}`;
      return createSuggestedComboName(draft, channelMap, t, fallbackName);
    },
    [batches.length, channelMap, editingBatchId, t],
  );

  const setEditorDraftSmart = useCallback(
    (updater) => {
      setEditorDraft((prev) => {
        const nextValue =
          typeof updater === 'function' ? updater(prev) : updater;
        if (!nextValue) return nextValue;

        const nextDraft = { ...nextValue };
        if (nextDraft.upstream_mode === 'wallet_observer') {
          const accountId = Number(nextDraft.upstream_account_id || 0);
          if (
            (!accountId || !availableAccountIds.has(accountId)) &&
            availableAccounts.length === 1
          ) {
            nextDraft.upstream_account_id = Number(availableAccounts[0].id);
          }
        } else {
          nextDraft.upstream_account_id = 0;
        }

        if (editorNameAuto) {
          nextDraft.name = getEditorSuggestedName(nextDraft);
        }

        return nextDraft;
      });
    },
    [
      availableAccountIds,
      availableAccounts,
      editorNameAuto,
      getEditorSuggestedName,
    ],
  );

  const handleEditorNameChange = useCallback((value) => {
    setEditorNameAuto(false);
    setEditorDraft((prev) => (prev ? { ...prev, name: value } : prev));
  }, []);

  const handleRegenerateEditorName = useCallback(() => {
    setEditorNameAuto(true);
    setEditorDraft((prev) =>
      prev ? { ...prev, name: getEditorSuggestedName(prev) } : prev,
    );
  }, [getEditorSuggestedName]);

  const dominantComboModes = useMemo(
    () =>
      pickDominantComboModes(
        comboConfigs,
        siteConfig?.pricing_mode === 'site_model'
          ? 'shared_site_model'
          : 'manual',
        upstreamConfig?.upstream_mode || 'manual_rules',
      ),
    [comboConfigs, siteConfig?.pricing_mode, upstreamConfig?.upstream_mode],
  );

  const recommendedAccountId = useMemo(
    () =>
      pickRecommendedUpstreamAccountId(
        comboConfigs,
        availableAccountIds,
        editingBatchId,
      ),
    [availableAccountIds, comboConfigs, editingBatchId],
  );

  const recommendedAccount = useMemo(
    () =>
      availableAccounts.find(
        (item) => Number(item.id) === Number(recommendedAccountId || 0),
      ) || null,
    [availableAccounts, recommendedAccountId],
  );

  const copyTemplateOptions = useMemo(
    () =>
      batches
        .filter((batch) => batch.id !== editingBatchId)
        .map((batch) => ({
          label: batch.name || t('未命名组合'),
          value: batch.id,
        })),
    [batches, editingBatchId, t],
  );

  const closeEditor = useCallback(() => {
    setEditorVisible(false);
    setEditingBatchId('');
    setEditorDraft(null);
    setEditorNameAuto(false);
    setEditorValidationError('');
  }, []);

  const openCreateBatchModal = useCallback(async () => {
    try {
      if (!builderOptionsReady) {
        await loadBuilderOptions();
        setBuilderOptionsReady(true);
      }
      const batchId = createBatchId();
      const defaultBatch = {
        id: batchId,
        name: `组合 ${batches.length + 1}`,
        scope_type: 'channel',
        channel_ids: [],
        tags: [],
        created_at: createBatchCreatedAt(),
      };
      const defaultComboConfig = createDefaultComboPricingConfig(
        batchId,
        siteConfig,
        siteConfig,
        upstreamConfig,
      );
      const nextDraft = cloneComboDraft(defaultBatch, defaultComboConfig);
      setEditingBatchId('');
      setEditorNameAuto(true);
      setEditorDraft({
        ...nextDraft,
        name: createSuggestedComboName(
          nextDraft,
          channelMap,
          t,
          `组合 ${batches.length + 1}`,
        ),
      });
      setEditorVisible(true);
    } catch (error) {
      showError(error);
    }
  }, [
    batches.length,
    builderOptionsReady,
    channelMap,
    loadBuilderOptions,
    siteConfig,
    t,
    upstreamConfig,
  ]);

  const openEditBatchModal = useCallback(
    async (batch) => {
      try {
        if (!builderOptionsReady) {
          await loadBuilderOptions();
          setBuilderOptionsReady(true);
        }
        const nextDraft = cloneComboDraft(batch, resolveComboConfig(batch.id));
        const suggestedName = createSuggestedComboName(
          nextDraft,
          channelMap,
          t,
          batch.name || t('未命名组合'),
        );
        setEditingBatchId(batch.id);
        setEditorNameAuto(isLikelyAutoComboName(batch.name, suggestedName));
        setEditorDraft(nextDraft);
        setEditorVisible(true);
      } catch (error) {
        showError(error);
      }
    },
    [builderOptionsReady, channelMap, loadBuilderOptions, resolveComboConfig, t],
  );

  const handleApplyRecommendedModes = useCallback(() => {
    setEditorDraftSmart((prev) =>
      prev
        ? {
            ...prev,
            site_mode: dominantComboModes.site_mode,
            upstream_mode: dominantComboModes.upstream_mode,
          }
        : prev,
    );
  }, [
    dominantComboModes.site_mode,
    dominantComboModes.upstream_mode,
    setEditorDraftSmart,
  ]);

  const handleApplyTemplate = useCallback(
    (templateBatchId) => {
      const templateConfig = comboConfigs.find(
        (item) => item.combo_id === templateBatchId,
      );
      if (!templateConfig) return;
      setEditorDraftSmart((prev) =>
        mergeComboDraftWithTemplate(prev, templateConfig),
      );
    },
    [comboConfigs, setEditorDraftSmart],
  );

  const handleApplyRecommendedAccount = useCallback(() => {
    if (!recommendedAccountId) return;
    setEditorDraftSmart((prev) =>
      prev
        ? {
            ...prev,
            upstream_mode: 'wallet_observer',
            upstream_account_id: Number(recommendedAccountId),
          }
        : prev,
    );
  }, [recommendedAccountId, setEditorDraftSmart]);

  const handleSaveEditor = useCallback(async () => {
    const error = buildDraftValidationError(editorDraft);
    if (error) {
      setEditorValidationError(error);
      showError(error);
      return;
    }
    const nextBatch = {
      id: editorDraft.id,
      name:
        editorDraft.name?.trim() ||
        `组合 ${batches.length + (editingBatchId ? 0 : 1)}`,
      scope_type: editorDraft.scope_type,
      channel_ids:
        editorDraft.scope_type === 'channel'
          ? editorDraft.channel_ids || []
          : [],
      tags: editorDraft.scope_type === 'tag' ? editorDraft.tags || [] : [],
      created_at: Number(editorDraft.created_at || createBatchCreatedAt()),
    };
    const nextComboConfig = {
      combo_id: nextBatch.id,
      site_mode: editorDraft.site_mode,
      upstream_mode: editorDraft.upstream_mode,
      cost_source: 'manual_only',
      upstream_account_id: Number(editorDraft.upstream_account_id || 0),
      shared_site: { ...(editorDraft.shared_site || {}) },
      site_rules: (editorDraft.site_rules || []).map((rule) =>
        createDefaultPricingRule(rule),
      ),
      upstream_rules: (editorDraft.upstream_rules || []).map((rule) =>
        createDefaultPricingRule(rule),
      ),
      site_fixed_total_amount: clampNumber(editorDraft.site_fixed_total_amount),
      upstream_fixed_total_amount: clampNumber(
        editorDraft.upstream_fixed_total_amount,
      ),
      remote_observer: { ...(editorDraft.remote_observer || {}) },
    };

    upsertBatch(nextBatch);
    setComboConfigs((prev) => {
      const exists = prev.some((item) => item.combo_id === nextBatch.id);
      if (!exists) return [...prev, nextComboConfig];
      return prev.map((item) =>
        item.combo_id === nextBatch.id ? nextComboConfig : item,
      );
    });
    const walletAccountId =
      nextComboConfig.upstream_mode === 'wallet_observer'
        ? Number(nextComboConfig.upstream_account_id || 0)
        : 0;
    showSuccess(
      walletAccountId > 0
        ? editingBatchId
          ? '组合已更新，正在同步上游账户'
          : '组合已添加，正在同步上游账户'
        : editingBatchId
          ? '组合已更新'
          : '组合已添加',
    );
    closeEditor();
    onConfigChanged();

    if (walletAccountId > 0) {
      await syncAccount(walletAccountId, {
        forceRefresh: true,
        suppressReadyToast: true,
        suppressNeedsBaselineToast: true,
      });
    }
  }, [
    syncAccount,
    batches.length,
    buildDraftValidationError,
    closeEditor,
    editorDraft,
    editingBatchId,
    onConfigChanged,
    setComboConfigs,
    upsertBatch,
  ]);

  const handleRemoveBatch = useCallback(
    (batch) => {
      Modal.confirm({
        title: t('确认删除'),
        content: t(
          '删除后将同时移除组合"{{name}}"及其定价配置，并自动同步到服务器。',
          { name: batch.name },
        ),
        okText: t('确认删除'),
        cancelText: t('取消'),
        okButtonProps: {
          type: 'danger',
        },
        onOk: () => {
          removeBatch(batch.id);
          setComboConfigs((prev) =>
            prev.filter((item) => item.combo_id !== batch.id),
          );
          onConfigChanged();
          if (editingBatchId === batch.id) {
            closeEditor();
          }
          showSuccess(t('组合已删除'));
        },
      });
    },
    [closeEditor, editingBatchId, onConfigChanged, removeBatch, setComboConfigs, t],
  );

  const editorSmartSuggestions = useMemo(() => {
    if (!editorDraft) return null;

    const suggestedName = getEditorSuggestedName(editorDraft);
    const currentAccountId = Number(editorDraft.upstream_account_id || 0);
    const recommendedModeLabel = `${dominantComboModes.site_mode === 'shared_site_model' ? t('本站模型价格') : t('手动定价')} / ${dominantComboModes.upstream_mode === 'wallet_observer' ? t('钱包余额变化') : t('模型单价')}`;

    return {
      suggestedName,
      canApplySuggestedName:
        !!suggestedName &&
        (!editorNameAuto || (editorDraft.name?.trim() || '') !== suggestedName),
      copyTemplateOptions,
      shouldRecommendModes:
        editorDraft.site_mode !== dominantComboModes.site_mode ||
        editorDraft.upstream_mode !== dominantComboModes.upstream_mode,
      recommendedModeLabel,
      recommendedAccountName: recommendedAccount?.name || '',
      shouldRecommendAccount:
        !!recommendedAccount &&
        (editorDraft.upstream_mode !== 'wallet_observer' ||
          currentAccountId !== Number(recommendedAccount.id)),
    };
  }, [
    copyTemplateOptions,
    dominantComboModes.site_mode,
    dominantComboModes.upstream_mode,
    editorDraft,
    editorNameAuto,
    getEditorSuggestedName,
    recommendedAccount,
    t,
  ]);

  return {
    editorVisible,
    editingBatchId,
    editorDraft,
    editorValidationError,
    editorSmartSuggestions,
    duplicateBatchError,
    setEditorDraftSmart,
    openCreateBatchModal,
    openEditBatchModal,
    closeEditor,
    handleEditorNameChange,
    handleRegenerateEditorName,
    handleApplyRecommendedModes,
    handleApplyTemplate,
    handleApplyRecommendedAccount,
    handleSaveEditor,
    handleRemoveBatch,
  };
};
