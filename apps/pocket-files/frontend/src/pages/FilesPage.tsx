// Files browser: server-side sort, type filter and pagination via
// /api/files?sort&order&type&page&limit; multi-select with bulk copy/move/delete
// through POST /api/operations/bulk (delete sends confirm=true). Unavailable
// endpoints render an explicit state instead of fabricated rows.

import { useEffect, useMemo, useState } from 'react';
import {
  Archive,
  BookOpen,
  ChevronLeft,
  ChevronRight,
  Copy,
  Eye,
  FileText,
  Film,
  Folder,
  FolderPlus,
  Image as ImageIcon,
  Move,
  Music,
  Pencil,
  Pin,
  SquareCheck,
  Star,
  Trash,
} from 'lucide-react';
import { api, ApiError } from '../api';
import { ConfirmDialog, ErrorState, LoadingState, UnavailableState } from '../components/ui';
import { formatBytes, formatDate, joinPath, parentPath } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';
import type { BulkOperation, FileEntry, FileTypeFilter, SortDirection, SortField } from '../types';
import { FilePreview } from '../components/FilePreview';

const PAGE_LIMIT = 50;

interface FilesPageProps {
  t: TFunction;
  lang: string;
  storageId: string;
  storageName: string;
  devices: { id: string; name: string; status: string }[];
  path: string;
  onPathChange: (path: string) => void;
}

export function FilesPage({
  t,
  lang,
  storageId,
  storageName,
  devices,
  path,
  onPathChange,
}: FilesPageProps) {
  const [sortField, setSortField] = useState<SortField>('name');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
  const [typeFilter, setTypeFilter] = useState<FileTypeFilter>('all');
  const [page, setPage] = useState(0);
  const [selected, setSelected] = useState<string[]>([]);
  const [destination, setDestination] = useState('');
  // Cross-device destinations go through the persistent transfer queue.
  const [destinationStorage, setDestinationStorage] = useState('');
  const [pendingDelete, setPendingDelete] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  // Last destructive action, kept so the user can undo a delete immediately.
  const [undoBatch, setUndoBatch] = useState<{
    storageId: string;
    entries: { source: string; trashPath: string }[];
  } | null>(null);
  const [renaming, setRenaming] = useState<FileEntry | null>(null);
  const [previewing, setPreviewing] = useState<FileEntry | null>(null);

  useEffect(() => {
    setPage(0);
    setSelected([]);
    setNotice(null);
  }, [storageId, path, typeFilter, sortField, sortDirection]);

  const files = useApiResource(
    () =>
      api.files({
        storage_id: storageId,
        path,
        type: typeFilter === 'all' ? undefined : typeFilter === 'folders' ? 'directory' : 'file',
        sort: sortField,
        order: sortDirection,
        limit: PAGE_LIMIT,
        page,
      }),
    [storageId, path, typeFilter, sortField, sortDirection, page],
  );

  const rows = useMemo(() => files.data?.items ?? [], [files.data]);
  // Only usable devices that are not the current one may be a copy destination;
  // not_implemented providers must never be offered as a target.
  const otherDevices = useMemo(
    () =>
      devices.filter(
        (device) => device.id !== storageId && device.status !== 'not_implemented',
      ),
    [devices, storageId],
  );
  const total = files.data?.total ?? rows.length;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_LIMIT));
  const isRoot = path === '';
  const allSelected = rows.length > 0 && rows.every((entry) => selected.includes(entry.id));

  const toggleSelect = (id: string) => {
    setSelected((current) =>
      current.includes(id) ? current.filter((x) => x !== id) : [...current, id],
    );
  };

  const toggleAll = () => {
    setSelected((current) => {
      if (allSelected) return current.filter((id) => !rows.some((entry) => entry.id === id));
      const additions = rows.map((entry) => entry.id).filter((id) => !current.includes(id));
      return [...current, ...additions];
    });
  };

  // filesd treats `destination` as a folder prefix, so a bare folder like
  // "Documents" must be sent as "Documents/" to avoid colliding with the
  // file name during copy/move.
  const normalizedDestination = () => {
    const raw = destination.trim().replace(/^\/+/, '');
    if (!raw) return '';
    return raw.endsWith('/') ? raw : `${raw}/`;
  };

  const runBulk = async (operation: BulkOperation, confirmed = false) => {
    if (selected.length === 0) return;
    const target = normalizedDestination();
    if ((operation === 'copy' || operation === 'move') && !target) {
      setNotice(t('filesDestination'));
      return;
    }
    const crossDevice = operation === 'copy' && destinationStorage !== '' && destinationStorage !== storageId;
    setBusy(true);
    setNotice(null);
    setConfirmError(null);
    try {
      const response = await api.bulk({
        action: operation,
        storage_id: storageId,
        sources: selected,
        destination: operation === 'delete' ? undefined : target,
        destination_storage: crossDevice ? destinationStorage : undefined,
        confirm: confirmed ? true : undefined,
      });
      const failures = response.results.filter((result) => result.status === 'error');
      if (operation === 'delete') {
        // The delete response carries the journal path for every trashed entry;
        // undo uses exactly those paths and never invents one.
        const trampled = response.results
          .filter((result) => result.status === 'ok' && result.trash_path)
          .map((result) => ({ source: result.source, trashPath: result.trash_path ?? '' }));
        setUndoBatch(trampled.length > 0 ? { storageId, entries: trampled } : null);
      } else {
        setUndoBatch(null);
      }
      setSelected([]);
      setPendingDelete(false);
      const queued = response.results.some((result) => result.status === 'queued');
      setNotice(
        failures.length === 0
          ? queued
            ? t('filesQueuedTransfer')
            : t('filesOperationDone')
          : `${t('filesOperationFailed')}: ${failures[0].error ?? ''}`,
      );
      files.reload();
    } catch (err) {
      const message = err instanceof ApiError ? err.message : t('filesOperationFailed');
      setConfirmError(`${t('filesOperationFailed')}: ${message}`);
      files.reload();
    } finally {
      setBusy(false);
    }
  };

  // Undo restores each trashed entry from the journal returned by the delete
  // call; the UI never guesses a trash path.
  const undoDelete = async () => {
    if (!undoBatch) return;
    setBusy(true);
    try {
      for (const entry of undoBatch.entries) {
        await api.undo({
          storage_id: undoBatch.storageId,
          source: entry.source,
          trash_path: entry.trashPath,
        });
      }
      setUndoBatch(null);
      setNotice(t('filesUndone'));
      files.reload();
    } catch (err) {
      const message = err instanceof ApiError ? err.message : t('filesOperationFailed');
      setConfirmError(`${t('filesOperationFailed')}: ${message}`);
    } finally {
      setBusy(false);
    }
  };

  const commitRename = async (name: string) => {
    if (!renaming) return;
    const trimmed = name.trim();
    if (!trimmed || trimmed === renaming.name) {
      setRenaming(null);
      return;
    }
    setBusy(true);
    try {
      await api.rename(storageId, renaming.id, trimmed);
      setRenaming(null);
      setNotice(t('filesOperationDone'));
      files.reload();
    } catch (err) {
      const message = err instanceof ApiError ? err.message : t('filesOperationFailed');
      setConfirmError(`${t('filesOperationFailed')}: ${message}`);
    } finally {
      setBusy(false);
    }
  };

  // Pin the current folder so it appears in the sidebar. The API stores a
  // provider-relative URI and rejects paths that escape the storage root.
  const pinCurrentFolder = async () => {
    const label = path === '' ? storageName : path.split('/').pop() ?? path;
    try {
      await api.addLocation({ name: label, storage_id: storageId, uri: path });
      setNotice(t('filesPinned'));
    } catch (err) {
      const message = err instanceof ApiError ? err.message : t('filesOperationFailed');
      setNotice(`${t('filesOperationFailed')}: ${message}`);
    }
  };

  const createFolder = async () => {
    const name = window.prompt(t('filesNewFolder'));
    if (!name) return;
    try {
      await api.createFolder(storageId, joinPath(path, name));
      files.reload();
    } catch {
      setNotice(t('filesOperationFailed'));
    }
  };

  const addFavorite = async (entry: FileEntry) => {
    try {
      await api.addFavorite(storageId, entry.id);
      setNotice(t('favoritesAdd'));
    } catch {
      setNotice(t('filesOperationFailed'));
    }
  };

  // Directories navigate; files open the verified preview surface instead of
  // exposing any filesystem path to the browser.
  const openEntry = (entry: FileEntry) => {
    if (entry.kind === 'directory') {
      onPathChange(joinPath(path, entry.name));
      return;
    }
    setPreviewing(entry);
  };

  const iconFor = (entry: FileEntry) => {
    if (entry.kind === 'directory') return <Folder aria-hidden />;
    const ext = (entry.extension || '').toLowerCase();
    if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg'].includes(ext)) return <ImageIcon aria-hidden />;
    if (['mp4', 'mkv', 'webm', 'avi', 'mov', 'm4v'].includes(ext)) return <Film aria-hidden />;
    if (['mp3', 'flac', 'wav', 'ogg', 'm4a', 'aac', 'opus'].includes(ext)) return <Music aria-hidden />;
    if (['epub', 'mobi', 'azw', 'azw3', 'cbz', 'cbr'].includes(ext)) return <BookOpen aria-hidden />;
    if (['zip', 'tar', 'gz', 'xz', '7z', 'rar'].includes(ext)) return <Archive aria-hidden />;
    return <FileText aria-hidden />;
  };

  let body;
  if (files.loading && !files.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (files.unavailable) {
    body = <UnavailableState t={t} message={t('filesUnavailable')} onRetry={files.reload} />;
  } else if (files.error) {
    body = <ErrorState t={t} message={files.error.message} onRetry={files.reload} />;
  } else {
    body = (
      <>
        <div className="file-toolbar">
          <button
            type="button"
            className="soft"
            onClick={() => onPathChange(parentPath(path))}
            disabled={isRoot}
            data-nav
          >
            <ChevronLeft aria-hidden /> {t('filesBack')}
          </button>

          <label className="select-control">
            <span>{t('filesSort')}</span>
            <select
              value={`${sortField}:${sortDirection}`}
              onChange={(event) => {
                const [field, direction] = event.target.value.split(':') as [
                  SortField,
                  SortDirection,
                ];
                setSortField(field);
                setSortDirection(direction);
              }}
              data-nav
            >
              <option value="name:asc">{t('filesSortName')} ↑</option>
              <option value="name:desc">{t('filesSortName')} ↓</option>
              <option value="size:asc">{t('filesSortSize')} ↑</option>
              <option value="size:desc">{t('filesSortSize')} ↓</option>
              <option value="modified:asc">{t('filesSortModified')} ↑</option>
              <option value="modified:desc">{t('filesSortModified')} ↓</option>
              <option value="type:asc">{t('filesSortType')} ↑</option>
              <option value="type:desc">{t('filesSortType')} ↓</option>
            </select>
          </label>

          <label className="select-control">
            <span>{t('filesFilterType')}</span>
            <select
              value={typeFilter}
              onChange={(event) => setTypeFilter(event.target.value as FileTypeFilter)}
              data-nav
            >
              <option value="all">{t('filesFilterAll')}</option>
              <option value="folders">{t('filesFilterFolders')}</option>
              <option value="files">{t('filesFilterFiles')}</option>
            </select>
          </label>

          <span className="toolbar-count">{t('commonItems', { n: total })}</span>
          <button type="button" className="soft" onClick={() => void createFolder()} data-nav>
            <FolderPlus aria-hidden /> {t('filesNewFolder')}
          </button>
          <button type="button" className="soft" onClick={() => void pinCurrentFolder()} data-nav>
            <Pin aria-hidden /> {t('filesPin')}
          </button>
        </div>

        {selected.length > 0 && (
          <div className="bulk-bar">
            <strong>{t('commonSelected', { n: selected.length })}</strong>
            <label className="bulk-destination">
              <span>{t('filesDestination')}</span>
              <input
                value={destination}
                onChange={(event) => setDestination(event.target.value)}
                placeholder={path || '/'}
                data-nav
              />
              <select
                value={destinationStorage}
                onChange={(event) => setDestinationStorage(event.target.value)}
                aria-label={t('filesDestinationStorage')}
                data-nav
              >
                <option value="">{storageName}</option>
                {otherDevices.map((device) => (
                  <option key={device.id} value={device.id}>
                    {device.name}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              className="soft"
              onClick={() => void runBulk('copy')}
              disabled={busy}
              data-nav
            >
              <Copy aria-hidden /> {t('filesBulkCopy')}
            </button>
            <button
              type="button"
              className="soft"
              onClick={() => void runBulk('move')}
              disabled={busy}
              data-nav
            >
              <Move aria-hidden /> {t('filesBulkMove')}
            </button>
            <button
              type="button"
              className="soft danger-soft"
              onClick={() => setPendingDelete(true)}
              disabled={busy}
              data-nav
            >
              <Trash aria-hidden /> {t('filesBulkDelete')}
            </button>
            <button type="button" className="ghost" onClick={() => setSelected([])} data-nav>
              {t('filesDone')}
            </button>
          </div>
        )}

        {notice && <div className="inline-notice">{notice}</div>}
        {undoBatch && (
          <div className="inline-notice undo-notice">
            <span>{t('filesUndoHint')}</span>
            <button type="button" className="soft" onClick={() => void undoDelete()} disabled={busy} data-nav>
              {t('filesUndo')}
            </button>
          </div>
        )}

        <div className="file-list" role="table" aria-label={t('filesTitle')}>
          <div className="list-head" role="row">
            <span>{t('filesName')}</span>
            <span>{t('filesSize')}</span>
            <span>{t('filesModified')}</span>
            <span>{t('filesType')}</span>
            <label className="checkbox">
              <input
                type="checkbox"
                checked={allSelected}
                onChange={toggleAll}
                aria-label={t('filesSelect')}
              />
              <SquareCheck aria-hidden />
            </label>
          </div>
          {rows.map((entry) => {
            const checked = selected.includes(entry.id);
            return (
              <div
                key={entry.id}
                className={'file-row' + (checked ? ' selected' : '')}
                role="row"
                tabIndex={0}
                aria-selected={checked}
                data-nav
                onClick={(event) => {
                  if (event.detail === 2) openEntry(entry);
                }}
                onKeyDown={(event) => {
                  // Enter opens; Space toggles selection so the list is fully
                  // operable from a keyboard or D-pad alone.
                  if (event.key === 'Enter') openEntry(entry);
                  if (event.key === ' ' || event.key === 'Spacebar') {
                    event.preventDefault();
                    toggleSelect(entry.id);
                  }
                }}
              >
                <span className="file-name">
                  {iconFor(entry)}
                  {entry.name}
                  {entry.kind !== 'directory' && (
                    <button
                      type="button"
                      className="icon-btn tiny"
                      title={t('filesPreview')}
                      aria-label={`${t('filesPreview')} ${entry.name}`}
                      onClick={(event) => {
                        event.stopPropagation();
                        setPreviewing(entry);
                      }}
                      data-nav
                    >
                      <Eye aria-hidden />
                    </button>
                  )}
                  <button
                    type="button"
                    className="icon-btn tiny"
                    title={t('favoritesAdd')}
                    aria-label={t('favoritesAdd')}
                    onClick={(event) => {
                      event.stopPropagation();
                      void addFavorite(entry);
                    }}
                    data-nav
                  >
                    <Star aria-hidden />
                  </button>
                  {renaming?.id === entry.id ? (
                    <input
                      className="rename-input"
                      defaultValue={entry.name}
                      autoFocus
                      aria-label={t('filesRenamePrompt')}
                      onClick={(event) => event.stopPropagation()}
                      onBlur={(event) => void commitRename(event.target.value)}
                      onKeyDown={(event) => {
                        event.stopPropagation();
                        if (event.key === 'Enter') void commitRename(event.currentTarget.value);
                        if (event.key === 'Escape') setRenaming(null);
                      }}
                    />
                  ) : (
                    <button
                      type="button"
                      className="icon-btn tiny"
                      title={t('filesRename')}
                      aria-label={t('filesRename')}
                      onClick={(event) => {
                        event.stopPropagation();
                        setRenaming(entry);
                      }}
                      data-nav
                    >
                      <Pencil aria-hidden />
                    </button>
                  )}
                </span>
                <span>{entry.kind === 'directory' ? '—' : formatBytes(entry.size)}</span>
                <span>{formatDate(entry.modified, lang)}</span>
                <span>
                  {entry.kind === 'directory' ? t('filesFilterFolders') : entry.extension || t('filesFilterFiles')}
                </span>
                <label className="checkbox" onClick={(event) => event.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={checked}
                    onChange={() => toggleSelect(entry.id)}
                    aria-label={`${t('filesSelect')} ${entry.name}`}
                  />
                  <SquareCheck aria-hidden />
                </label>
              </div>
            );
          })}
          {rows.length === 0 && <div className="empty">{t('commonEmptyFolder')}</div>}
        </div>

        {previewing && (
          <FilePreview
            t={t}
            storageId={storageId}
            entry={previewing}
            onClose={() => setPreviewing(null)}
          />
        )}

        <div className="pagination">
          <button
            type="button"
            className="soft"
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0}
            data-nav
          >
            <ChevronLeft aria-hidden /> {t('filesPrev')}
          </button>
          <span>
            {t('filesPage')} {page + 1} / {pageCount}
          </span>
          <button
            type="button"
            className="soft"
            onClick={() => setPage((p) => Math.min(pageCount - 1, p + 1))}
            disabled={page + 1 >= pageCount}
            data-nav
          >
            {t('filesNext')} <ChevronRight aria-hidden />
          </button>
        </div>
      </>
    );
  }

  return (
    <>
      <div className="welcome compact">
        <div>
          <div className="eyebrow">{storageName}</div>
          <h1>{path || storageName || t('filesTitle')}</h1>
          <p>{path ? storageName : t('filesTitle')}</p>
        </div>
      </div>
      {confirmError && <div className="inline-notice">{confirmError}</div>}
      {body}
      <ConfirmDialog
        open={pendingDelete}
        title={t('filesConfirmDeleteTitle')}
        body={t('filesConfirmDeleteBody', { n: selected.length })}
        confirmLabel={t('filesConfirmDelete')}
        cancelLabel={t('commonCancel')}
        onConfirm={() => void runBulk('delete', true)}
        onCancel={() => setPendingDelete(false)}
      />
    </>
  );
}
