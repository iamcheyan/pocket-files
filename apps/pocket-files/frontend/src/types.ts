// Shared data contracts for the Pocket Files UI.
// Field names mirror the filesd REST payloads (snake_case) so responses can be
// used directly without an adapter layer.

export interface StorageDevice {
  id: string;
  name: string;
  kind: string;
  status: string;
  mock: boolean;
  read_only: boolean;
  total: number;
  used: number;
}

export type EntryKind = 'file' | 'directory';

export interface FileEntry {
  id: string;
  storage_id: string;
  name: string;
  parent: string;
  kind: EntryKind | string;
  size: number;
  modified: string;
  created?: string;
  extension: string;
}

export interface UsageInfo {
  id: string;
  mock: boolean;
  total: number;
  used: number;
  free: number;
}

export interface StorageListResponse {
  items: StorageDevice[];
}

export interface FilesResponse {
  items: FileEntry[];
  page?: number;
  limit?: number;
  total?: number;
}

// Backend transfer states.
export type TransferStatus =
  | 'queued'
  | 'running'
  | 'paused'
  | 'failed'
  | 'completed'
  | 'cancelled'
  | 'conflict'
  | string;

export interface TransferItemState {
  src: string;
  dst: string;
  dst_storage?: string;
  state: string;
  error?: string;
}

export interface TransferJob {
  id: string;
  type?: string;
  storage_id?: string;
  status: TransferStatus;
  conflict_strategy?: string;
  paused?: boolean;
  priority?: number;
  bytes_done?: number;
  total_bytes?: number;
  speed?: number;
  eta?: number;
  items?: TransferItemState[];
}

export interface TransferListResponse {
  items: TransferJob[];
}

export type TransferAction = 'pause' | 'resume' | 'cancel' | 'retry';

export interface FavoriteItem {
  storage_id: string;
  uri: string;
}

export interface FavoritesResponse {
  items: FavoriteItem[];
}

export interface RecentItem {
  storage_id: string;
  uri: string;
  kind: string;
  accessed_at: string;
}

export interface RecentsResponse {
  items: RecentItem[];
}

export interface ViewResponse {
  view: string;
  items: FileEntry[];
}

export interface AnalyticsLargestFile {
  id?: string;
  name?: string;
  size?: number;
  extension?: string;
  storage_id?: string;
}

export interface AnalyticsData {
  storage_id: string;
  mock: boolean;
  total: number;
  used: number;
  free: number;
  category_counts: Record<string, number>;
  largest_files: AnalyticsLargestFile[];
  largest_directories: Array<{ storage_id: string; uri: string; size: number }>;
  cache_timestamp: string;
  backgrounded: boolean;
  progress: number;
}

export type BulkOperation = 'copy' | 'move' | 'delete';

export interface BulkOperationResult {
  status: string;
  action: string;
  storage_id: string;
  source: string;
  destination?: string;
  trash_path?: string;
  error?: string;
}

export interface BulkRequest {
  action: BulkOperation;
  storage_id: string;
  sources: string[];
  destination?: string;
  /** Set to copy across devices; cross-device copies are queued as transfers. */
  destination_storage?: string;
  confirm?: boolean;
}

export interface RenameResponse {
  status: string;
  storage_id: string;
  source: string;
  destination: string;
  name: string;
}

export interface UndoRequest {
  storage_id: string;
  source: string;
  trash_path: string;
}

export interface BulkResponse {
  status: string;
  action: string;
  storage_id: string;
  results: BulkOperationResult[];
}

export interface PinnedLocation {
  id: number;
  name: string;
  storage_id: string;
  uri: string;
}

export interface PinnedResponse {
  items: PinnedLocation[];
}

export interface DownloadsResponse {
  status: string;
  source: string;
  reason?: string;
  message?: string;
  items?: Array<Record<string, unknown>>;
}

export interface AppLaunchResult {
  status: string;
  app?: string;
  target?: string;
  reason?: string;
  message?: string;
}

export interface HealthResponse {
  status: string;
  mode: string;
}

export type SortField = 'name' | 'size' | 'modified' | 'type';
export type SortDirection = 'asc' | 'desc';

export type FileTypeFilter = 'all' | 'folders' | 'files';

export type PageId =
  | 'overview'
  | 'files'
  | 'storage'
  | 'transfers'
  | 'search'
  | 'favorites'
  | 'recents'
  | 'libraries'
  | 'analytics'
  | 'settings';

export type LibraryKey = 'books' | 'music' | 'videos' | 'games' | 'downloads';
