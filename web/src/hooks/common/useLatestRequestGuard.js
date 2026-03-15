import { useCallback, useMemo, useRef } from 'react';

export const useLatestRequestGuard = () => {
  const requestIdRef = useRef(0);

  const createRequestId = useCallback(() => {
    const nextId = requestIdRef.current + 1;
    requestIdRef.current = nextId;
    return nextId;
  }, []);

  const isLatestRequest = useCallback(
    (requestId) => requestId === requestIdRef.current,
    [],
  );

  return useMemo(
    () => ({
      createRequestId,
      isLatestRequest,
    }),
    [createRequestId, isLatestRequest],
  );
};

export default useLatestRequestGuard;
