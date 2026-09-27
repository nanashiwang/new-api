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

import { useEffect, useRef, useState } from 'react';
import { API, showError, showInfo, showSuccess } from '../../helpers';
import { normalizeModelList, selectPendingModels } from './upstreamUpdateUtils';

export const useChannelUpstreamUpdates = ({ t, refresh }) => {
  const [showUpstreamUpdateModal, setShowUpstreamUpdateModal] = useState(false);
  const [upstreamUpdateChannel, setUpstreamUpdateChannel] = useState(null);
  const [upstreamUpdateAddModels, setUpstreamUpdateAddModels] = useState([]);
  const [upstreamUpdateRemoveModels, setUpstreamUpdateRemoveModels] = useState(
    [],
  );
  const [upstreamUpdatePreferredTab, setUpstreamUpdatePreferredTab] =
    useState('add');
  const [upstreamApplyLoading, setUpstreamApplyLoading] = useState(false);
  const [upstreamPreviewLoading, setUpstreamPreviewLoading] = useState(false);
  const [upstreamPreviewError, setUpstreamPreviewError] = useState('');
  const [detectAllUpstreamUpdatesLoading, setDetectAllUpstreamUpdatesLoading] =
    useState(false);
  const [applyAllUpstreamUpdatesLoading, setApplyAllUpstreamUpdatesLoading] =
    useState(false);

  const applyUpstreamUpdatesInFlightRef = useRef(false);
  const previewRequestRef = useRef(0);
  const detectAllUpstreamUpdatesInFlightRef = useRef(false);
  const applyAllUpstreamUpdatesInFlightRef = useRef(false);
  useEffect(
    () => () => {
      previewRequestRef.current++;
    },
    [],
  );

  const refreshAfterSuccess = async () => {
    try {
      await refresh();
    } catch {
      showInfo(t('操作已成功，但列表刷新失败，请手动刷新'));
    }
  };

  const openUpstreamUpdateModal = (
    record,
    pendingAddModels = [],
    pendingRemoveModels = [],
    preferredTab = 'add',
  ) => {
    const normalizedAddModels = normalizeModelList(pendingAddModels);
    const normalizedRemoveModels = normalizeModelList(pendingRemoveModels);
    if (!record?.id || applyUpstreamUpdatesInFlightRef.current) return;
    previewRequestRef.current++;
    setUpstreamPreviewLoading(false);
    setUpstreamPreviewError('');
    setUpstreamUpdateChannel(record);
    setUpstreamUpdateAddModels(normalizedAddModels);
    setUpstreamUpdateRemoveModels(normalizedRemoveModels);
    const normalizedPreferredTab = preferredTab === 'remove' ? 'remove' : 'add';
    setUpstreamUpdatePreferredTab(normalizedPreferredTab);
    setShowUpstreamUpdateModal(true);
  };

  const resetUpstreamUpdateModal = () => {
    previewRequestRef.current++;
    setUpstreamPreviewLoading(false);
    setUpstreamPreviewError('');
    setShowUpstreamUpdateModal(false);
    setUpstreamUpdateChannel(null);
    setUpstreamUpdateAddModels([]);
    setUpstreamUpdateRemoveModels([]);
    setUpstreamUpdatePreferredTab('add');
  };
  const closeUpstreamUpdateModal = () => {
    if (!applyUpstreamUpdatesInFlightRef.current) resetUpstreamUpdateModal();
  };

  const applyUpstreamUpdates = async ({
    addModels: selectedAddModels = [],
    removeModels: selectedRemoveModels = [],
  } = {}) => {
    if (applyUpstreamUpdatesInFlightRef.current) {
      showInfo(t('正在处理，请稍候'));
      return;
    }
    if (upstreamPreviewLoading || upstreamPreviewError) return;
    if (!upstreamUpdateChannel?.id) {
      closeUpstreamUpdateModal();
      return;
    }
    const normalizedSelectedAddModels = selectPendingModels(
      selectedAddModels,
      upstreamUpdateAddModels,
    );
    const normalizedSelectedRemoveModels = selectPendingModels(
      selectedRemoveModels,
      upstreamUpdateRemoveModels,
    );
    if (
      !normalizedSelectedAddModels.length &&
      !normalizedSelectedRemoveModels.length
    )
      return;
    applyUpstreamUpdatesInFlightRef.current = true;
    setUpstreamApplyLoading(true);
    const requestId = previewRequestRef.current;

    try {
      const res = await API.post(
        '/api/channel/upstream_updates/apply',
        {
          id: upstreamUpdateChannel.id,
          add_models: normalizedSelectedAddModels,
          ignore_models: [],
          remove_models: normalizedSelectedRemoveModels,
        },
        { skipErrorHandler: true },
      );
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('操作失败'));
        return;
      }

      const addedCount = data?.added_models?.length || 0;
      const removedCount = data?.removed_models?.length || 0;
      showSuccess(
        t(
          '已处理上游模型更新：加入 {{added}} 个，删除 {{removed}} 个；未勾选项保留待处理',
          {
            added: addedCount,
            removed: removedCount,
          },
        ),
      );
      if (requestId === previewRequestRef.current) {
        resetUpstreamUpdateModal();
      }
      await refreshAfterSuccess();
    } catch (error) {
      showError(
        error?.response?.data?.message || error?.message || t('操作失败'),
      );
    } finally {
      applyUpstreamUpdatesInFlightRef.current = false;
      setUpstreamApplyLoading(false);
    }
  };

  const applyAllUpstreamUpdates = async () => {
    if (applyAllUpstreamUpdatesInFlightRef.current) {
      showInfo(t('正在批量处理，请稍候'));
      return;
    }
    applyAllUpstreamUpdatesInFlightRef.current = true;
    setApplyAllUpstreamUpdatesLoading(true);
    try {
      const res = await API.post(
        '/api/channel/upstream_updates/apply_all',
        {},
        { skipErrorHandler: true },
      );
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('批量处理失败'));
        return;
      }

      const channelCount = data?.processed_channels || 0;
      const addedCount = data?.added_models || 0;
      const removedCount = data?.removed_models || 0;
      const failedCount = (data?.failed_channel_ids || []).length;
      showSuccess(
        t(
          '已批量处理上游模型更新：渠道 {{channels}} 个，加入 {{added}} 个，删除 {{removed}} 个，失败 {{fails}} 个',
          {
            channels: channelCount,
            added: addedCount,
            removed: removedCount,
            fails: failedCount,
          },
        ),
      );
      await refreshAfterSuccess();
    } catch (error) {
      showError(
        error?.response?.data?.message || error?.message || t('批量处理失败'),
      );
    } finally {
      applyAllUpstreamUpdatesInFlightRef.current = false;
      setApplyAllUpstreamUpdatesLoading(false);
    }
  };

  const detectChannelUpstreamUpdates = async (channel) => {
    if (!channel?.id || applyUpstreamUpdatesInFlightRef.current) return;
    openUpstreamUpdateModal(channel);
    const requestId = previewRequestRef.current;
    setUpstreamPreviewLoading(true);
    try {
      const res = await API.post(
        '/api/channel/upstream_updates/detect',
        {
          id: channel.id,
        },
        { skipErrorHandler: true },
      );
      if (requestId !== previewRequestRef.current) return;
      const { success, message, data } = res.data || {};
      if (!success) {
        setUpstreamPreviewError(message || t('检测失败'));
        return;
      }
      setUpstreamUpdateAddModels(normalizeModelList(data?.add_models));
      setUpstreamUpdateRemoveModels(normalizeModelList(data?.remove_models));
      await refreshAfterSuccess();
    } catch (error) {
      if (requestId !== previewRequestRef.current) return;
      setUpstreamPreviewError(
        error?.response?.data?.message || error?.message || t('检测失败'),
      );
    } finally {
      if (requestId === previewRequestRef.current)
        setUpstreamPreviewLoading(false);
    }
  };

  const detectAllUpstreamUpdates = async () => {
    if (detectAllUpstreamUpdatesInFlightRef.current) {
      showInfo(t('正在批量检测，请稍候'));
      return;
    }
    detectAllUpstreamUpdatesInFlightRef.current = true;
    setDetectAllUpstreamUpdatesLoading(true);
    try {
      const res = await API.post(
        '/api/channel/upstream_updates/detect_all',
        {},
        { skipErrorHandler: true },
      );
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('批量检测失败'));
        return;
      }

      const channelCount = data?.processed_channels || 0;
      const addCount = data?.detected_add_models || 0;
      const removeCount = data?.detected_remove_models || 0;
      const failedCount = (data?.failed_channel_ids || []).length;
      showSuccess(
        t(
          '批量检测完成：渠道 {{channels}} 个，新增 {{add}} 个，删除 {{remove}} 个，失败 {{fails}} 个',
          {
            channels: channelCount,
            add: addCount,
            remove: removeCount,
            fails: failedCount,
          },
        ),
      );
      await refreshAfterSuccess();
    } catch (error) {
      showError(
        error?.response?.data?.message || error?.message || t('批量检测失败'),
      );
    } finally {
      detectAllUpstreamUpdatesInFlightRef.current = false;
      setDetectAllUpstreamUpdatesLoading(false);
    }
  };

  return {
    showUpstreamUpdateModal,
    upstreamUpdateChannel,
    upstreamUpdateAddModels,
    upstreamUpdateRemoveModels,
    upstreamUpdatePreferredTab,
    upstreamApplyLoading,
    upstreamPreviewLoading,
    upstreamPreviewError,
    detectAllUpstreamUpdatesLoading,
    applyAllUpstreamUpdatesLoading,
    openUpstreamUpdateModal,
    closeUpstreamUpdateModal,
    applyUpstreamUpdates,
    applyAllUpstreamUpdates,
    detectChannelUpstreamUpdates,
    detectAllUpstreamUpdates,
  };
};
