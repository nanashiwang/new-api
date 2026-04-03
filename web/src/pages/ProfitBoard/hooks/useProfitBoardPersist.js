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
