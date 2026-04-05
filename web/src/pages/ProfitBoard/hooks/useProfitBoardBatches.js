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

export const useProfitBoardBatches = ({ restoredState }) => {
  const [batches, setBatches] = useState(restoredState.batches || []);

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

  const upsertBatch = useCallback((batch) => {
    setBatches((prev) => {
      const exists = prev.some((item) => item.id === batch.id);
      if (!exists) {
        return [...prev, batch];
      }
      return prev.map((item) => (item.id === batch.id ? batch : item));
    });
  }, []);

  const removeBatch = useCallback((batchId) => {
    setBatches((prev) => prev.filter((item) => item.id !== batchId));
  }, []);

  return {
    batches,
    setBatches,
    batchPayload,
    upsertBatch,
    removeBatch,
  };
};
