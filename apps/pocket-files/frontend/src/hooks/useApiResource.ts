// Generic async resource loader. Surfaces loading, error and the API's
// "unavailable" flag so pages never substitute fabricated data. Supports
// optional polling (transfers) with abort-on-unmount semantics.

import { useCallback, useEffect, useRef, useState } from 'react';
import { ApiError } from '../api';

export interface ResourceState<T> {
  data: T | null;
  loading: boolean;
  error: ApiError | null;
  unavailable: boolean;
  reload: () => void;
}

export interface ResourceOptions {
  /** When > 0, re-fetch on this cadence. */
  intervalMs?: number;
  /** Skip fetching entirely (e.g. no storage selected yet). */
  enabled?: boolean;
}

export function useApiResource<T>(
  load: () => Promise<T>,
  deps: readonly unknown[],
  options: ResourceOptions = {},
): ResourceState<T> {
  const { intervalMs = 0, enabled = true } = options;
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(enabled);
  const [error, setError] = useState<ApiError | null>(null);
  const [nonce, setNonce] = useState(0);
  const loadRef = useRef(load);
  loadRef.current = load;

  const reload = useCallback(() => setNonce((n) => n + 1), []);

  useEffect(() => {
    if (!enabled) {
      setData(null);
      setLoading(false);
      setError(null);
      return;
    }

    let active = true;
    let timer = 0;

    const run = async (initial: boolean) => {
      if (initial) setLoading(true);
      try {
        const next = await loadRef.current();
        if (!active) return;
        setData(next);
        setError(null);
      } catch (err) {
        if (!active) return;
        const apiError =
          err instanceof ApiError ? err : new ApiError(String(err), 0, 'network_error');
        setError(apiError);
        if (initial) setData(null);
      } finally {
        if (active) {
          setLoading(false);
          if (intervalMs > 0) timer = window.setTimeout(() => void run(false), intervalMs);
        }
      }
    };

    void run(true);

    return () => {
      active = false;
      window.clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, enabled, intervalMs, nonce]);

  return {
    data,
    loading,
    error,
    unavailable: error?.unavailable === true,
    reload,
  };
}
