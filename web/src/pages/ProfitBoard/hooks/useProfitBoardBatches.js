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
import { showError, showSuccess } from '../../../helpers';
import { createDefaultDraft, normalizeBatchForState } from '../utils';

export const useProfitBoardBatches = ({ restoredState }) => {
  const [batches, setBatches] = useState(restoredState.batches || []);
  const [draft, setDraft] = useState(
    restoredState.draft || createDefaultDraft(),
  );
  const [editingBatchId, setEditingBatchId] = useState(
    restoredState.editingBatchId || '',
  );

  const batchPayload = useMemo(
    () =>
      batches.map((batch) => ({
        id: batch.id,
        name: batch.name?.trim() || '未命名组合',
        scope_type: batch.scope_type,
        channel_ids: (batch.channel_ids || [])
          .map((item) => Number(item))
          .filter(Boolean),
        tags: batch.tags || [],
      })),
    [batches],
  );

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
    if (!selectedCount) return showError('请先选择渠道或标签');
    setBatches((prev) =>
      editingBatchId
        ? prev.map((item) => (item.id === editingBatchId ? nextBatch : item))
        : [...prev, nextBatch],
    );
    setDraft(createDefaultDraft());
    setEditingBatchId('');
    showSuccess(editingBatchId ? '组合已更新' : '组合已添加');
  }, [batches.length, draft, editingBatchId]);

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
      if (editingBatchId === batchId) resetDraft();
    },
    [editingBatchId, resetDraft],
  );

  return {
    batches,
    setBatches,
    draft,
    setDraft,
    editingBatchId,
    batchPayload,
    addOrUpdateBatch,
    editBatch,
    resetDraft,
    removeBatch,
  };
};
