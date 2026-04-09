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
import { useCallback, useEffect, useMemo, useRef } from 'react';
import {
  PROFIT_BOARD_CACHE_VERSION,
  REPORT_CACHE_KEY,
  STORAGE_KEY,
  normalizeCachedReportBundle,
  normalizeRestoredState,
} from '../utils';

const DEBOUNCE_MS = 1000;

const readVersionedCache = (storageKey) => {
  const raw = localStorage.getItem(storageKey);
  if (raw === null) {
    return { exists: false, valid: true, data: null };
  }

  try {
    const parsed = JSON.parse(raw);
    if (
      !parsed ||
      typeof parsed !== 'object' ||
      Array.isArray(parsed) ||
      parsed.version !== PROFIT_BOARD_CACHE_VERSION ||
      !Object.prototype.hasOwnProperty.call(parsed, 'data')
    ) {
      return { exists: true, valid: false, data: null };
    }
    return { exists: true, valid: true, data: parsed.data };
  } catch (error) {
    return { exists: true, valid: false, data: null };
  }
};

export const useProfitBoardPersist = () => {
  const cacheState = useMemo(() => {
    const restoredStateEntry = readVersionedCache(STORAGE_KEY);
    const reportCacheEntry = readVersionedCache(REPORT_CACHE_KEY);
    const shouldResetCache =
      (restoredStateEntry.exists && !restoredStateEntry.valid) ||
      (reportCacheEntry.exists && !reportCacheEntry.valid);

    if (shouldResetCache) {
      return {
        shouldResetCache,
        restoredStateData: {},
        reportCacheData: null,
      };
    }

    return {
      shouldResetCache,
      restoredStateData: restoredStateEntry.data || {},
      reportCacheData: reportCacheEntry.data,
    };
  }, []);

  const cachedBundle = useMemo(
    () => normalizeCachedReportBundle(cacheState.reportCacheData),
    [cacheState.reportCacheData],
  );
  const restoredState = useMemo(
    () => normalizeRestoredState(cacheState.restoredStateData),
    [cacheState.restoredStateData],
  );

  const saveTimeoutRef = useRef(null);
  const lastHashRef = useRef('');

  useEffect(() => {
    if (cacheState.shouldResetCache) {
      localStorage.removeItem(STORAGE_KEY);
      localStorage.removeItem(REPORT_CACHE_KEY);
    }
    return () => {
      if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
    };
  }, [cacheState.shouldResetCache]);

  const persistState = useCallback((snapshot) => {
    if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
    saveTimeoutRef.current = setTimeout(() => {
      const hash = JSON.stringify(snapshot);
      if (hash === lastHashRef.current) return;
      lastHashRef.current = hash;
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({
          version: PROFIT_BOARD_CACHE_VERSION,
          data: snapshot,
        }),
      );
    }, DEBOUNCE_MS);
  }, []);

  const persistReportCache = useCallback((report, queryKey) => {
    localStorage.setItem(
      REPORT_CACHE_KEY,
      JSON.stringify({
        version: PROFIT_BOARD_CACHE_VERSION,
        data: {
          report,
          queryKey,
          activityWatermark: report?.meta?.activity_watermark || '',
        },
      }),
    );
  }, []);

  return { restoredState, cachedBundle, persistState, persistReportCache };
};
