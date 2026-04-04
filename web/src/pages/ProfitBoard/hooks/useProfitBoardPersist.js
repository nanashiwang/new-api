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
  REPORT_CACHE_KEY,
  STORAGE_KEY,
  normalizeCachedReportBundle,
  normalizeRestoredState,
  safeParse,
} from '../utils';

const DEBOUNCE_MS = 1000;

export const useProfitBoardPersist = () => {
  const cachedBundle = useMemo(
    () =>
      normalizeCachedReportBundle(
        safeParse(localStorage.getItem(REPORT_CACHE_KEY), null),
      ),
    [],
  );
  const restoredState = useMemo(
    () =>
      normalizeRestoredState(safeParse(localStorage.getItem(STORAGE_KEY), {})),
    [],
  );

  const saveTimeoutRef = useRef(null);
  const lastHashRef = useRef('');

  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
    };
  }, []);

  const persistState = useCallback((snapshot) => {
    if (saveTimeoutRef.current) clearTimeout(saveTimeoutRef.current);
    saveTimeoutRef.current = setTimeout(() => {
      const hash = JSON.stringify(snapshot);
      if (hash === lastHashRef.current) return;
      lastHashRef.current = hash;
      localStorage.setItem(STORAGE_KEY, hash);
    }, DEBOUNCE_MS);
  }, []);

  const persistReportCache = useCallback((report, queryKey) => {
    localStorage.setItem(
      REPORT_CACHE_KEY,
      JSON.stringify({
        report,
        queryKey,
        activityWatermark: report?.meta?.activity_watermark || '',
      }),
    );
  }, []);

  return { restoredState, cachedBundle, persistState, persistReportCache };
};
