// Thin typed client over the filesd REST API.
// Every failure surfaces as ApiError so callers can distinguish "unavailable"
// (501/503/network) from ordinary validation errors and render an explicit
// state instead of fabricating data.

import type {
  AnalyticsData,
  AppLaunchResult,
  BulkRequest,
  BulkResponse,
  FavoritesResponse,
  FilesResponse,
  HealthResponse,
  RecentsResponse,
  StorageDevice,
  StorageListResponse,
  TransferAction,
  RenameResponse,
  TransferListResponse,
  UndoRequest,
  UsageInfo,
  ViewResponse,
} from './types';

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly unavailable: boolean;

  constructor(message: string, status: number, code: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.unavailable = status === 501 || status === 503 || status === 0;
  }
}

interface ErrorEnvelope {
  error?: { code?: string; message?: string };
}

async function readBody(res: Response): Promise<unknown> {
  const text = await res.text();
  if (!text) return null;
  try {
    return JSON.parse(text) as unknown;
  } catch {
    return text;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let res: Response;
  try {
    res = await fetch(path, {
      headers: {
        accept: 'application/json',
        ...(init?.body ? { 'content-type': 'application/json' } : {}),
      },
      ...init,
    });
  } catch (err) {
    throw new ApiError(err instanceof Error ? err.message : 'network error', 0, 'network_error');
  }

  const body = await readBody(res);
  if (!res.ok) {
    const envelope = body as ErrorEnvelope | null;
    const code = envelope?.error?.code ?? `http_${res.status}`;
    const message = envelope?.error?.message ?? `request failed with status ${res.status}`;
    throw new ApiError(message, res.status, code);
  }
  return body as T;
}

function query(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') search.set(key, String(value));
  }
  const qs = search.toString();
  return qs ? `?${qs}` : '';
}

export interface FilesQuery {
  storage_id: string;
  path?: string;
  q?: string;
  ext?: string;
  type?: 'file' | 'directory';
  min_size?: number;
  max_size?: number;
  sort?: 'name' | 'size' | 'modified' | 'created' | 'type';
  order?: 'asc' | 'desc';
  limit?: number;
  page?: number;
}

export const api = {
  health(): Promise<HealthResponse> {
    return request<HealthResponse>('/api/health');
  },

  async storage(): Promise<StorageDevice[]> {
    const body = await request<StorageListResponse>('/api/storage');
    return body.items ?? [];
  },

  files(params: FilesQuery): Promise<FilesResponse> {
    return request<FilesResponse>(
      `/api/files${query({
        storage_id: params.storage_id,
        path: params.path,
        q: params.q,
        ext: params.ext,
        type: params.type,
        min_size: params.min_size,
        max_size: params.max_size,
        sort: params.sort,
        order: params.order,
        limit: params.limit,
        page: params.page,
      })}`,
    );
  },

  usage(storageId: string): Promise<UsageInfo> {
    return request<UsageInfo>(`/api/storage/${encodeURIComponent(storageId)}/usage`);
  },

  createFolder(storageId: string, path: string): Promise<void> {
    return request<void>('/api/folders', {
      method: 'POST',
      body: JSON.stringify({ storage_id: storageId, path }),
    });
  },

  bulk(payload: BulkRequest): Promise<BulkResponse> {
    return request<BulkResponse>('/api/operations/bulk', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  // File identifiers are "storage:path" and must be escaped as one segment.
  rename(storageId: string, id: string, name: string): Promise<RenameResponse> {
    return request<RenameResponse>(`/api/files/${encodeURIComponent(`${storageId}:${id}`)}/rename`, {
      method: 'PUT',
      body: JSON.stringify({ name }),
    });
  },

  undo(payload: UndoRequest): Promise<void> {
    return request<void>('/api/operations/undo', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  },

  transfers(): Promise<TransferListResponse> {
    return request<TransferListResponse>('/api/transfers');
  },

  transferAction(id: string, action: TransferAction): Promise<void> {
    return request<void>(`/api/transfers/${encodeURIComponent(id)}/${action}`, { method: 'POST' });
  },

  favorites(): Promise<FavoritesResponse> {
    return request<FavoritesResponse>('/api/favorites');
  },

  addFavorite(storageId: string, uri: string): Promise<void> {
    return request<void>('/api/favorites', {
      method: 'POST',
      body: JSON.stringify({ storage_id: storageId, uri }),
    });
  },

  removeFavorite(storageId: string, uri: string): Promise<void> {
    return request<void>('/api/favorites', {
      method: 'DELETE',
      body: JSON.stringify({ storage_id: storageId, uri }),
    });
  },

  recents(kind?: string): Promise<RecentsResponse> {
    return request<RecentsResponse>(`/api/recents${query({ kind })}`);
  },

  view(name: string): Promise<ViewResponse> {
    return request<ViewResponse>(`/api/views/${encodeURIComponent(name)}`);
  },

  // Ask PJAppRegistry (through filesd) to open the entry. A registry that is not
  // configured answers status:"unavailable" rather than throwing.
  openInApp(storageId: string, path: string): Promise<AppLaunchResult> {
    return request<AppLaunchResult>(
      `/api/apps/open${query({ storage_id: storageId, path })}`,
    );
  },

  analytics(storageId: string): Promise<AnalyticsData> {
    return request<AnalyticsData>(`/api/analytics${query({ storage_id: storageId })}`);
  },
};
