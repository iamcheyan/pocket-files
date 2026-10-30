// Files browser: server-side sort, type filter and pagination via
// /api/files?sort&order&type&page&limit; multi-select with bulk copy/move/delete
// through POST /api/operations/bulk (delete sends confirm=true). Unavailable
// endpoints render an explicit state instead of fabricated rows.

import { useEffect, useMemo, useState } from 'react';
import {
  ChevronLeft,
  ChevronRight,
  Copy,
  FileText,
  Folder,
  FolderPlus,
  Move,
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

const PAGE_LIMIT = 50;

interface FilesPageProps {
  t: TFunction;
  lang: string;
  storageId: string;
  storageName: string;
  path: string;
  onPathChange: (path: string) => void;
}

export function FilesPage({ t, lang, storageId, storageName, path, onPathChange }: FilesPageProps) {
  const [sortField, setSortField] = useState<SortField>('name');
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
  const [typeFilter, setTypeFilter] = useState<FileTypeFilter>('all');
  const [page, setPage] = useState(0);
  const [selected, setSelected] = useState<string[]>([]);
  const [destination, setDestination] = useState('');
  const [pendingDelete, setPendingDelete] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [confirmError, setConfirmError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

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
    setBusy(true);
    setNotice(null);
    setConfirmError(null);
    try {
      const response = await api.bulk({
        action: operation,
        storage_id: storageId,
        sources: selected,
        destination: operation === 'delete' ? undefined : target,
        confirm: confirmed ? true : undefined,
      });
      const failures = response.results.filter((result) => result.status === 'error');
      setSelected([]);
      setPendingDelete(false);
      setNotice(
        failures.length === 0
          ? t('filesOperationDone')
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

  const openEntry = (entry: FileEntry) => {
    if (entry.kind === 'directory') onPathChange(joinPath(path, entry.name));
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
                data-nav
                onClick={(event) => {
                  if (event.detail === 2) openEntry(entry);
                }}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') openEntry(entry);
                }}
              >
                <span className="file-name">
                  {entry.kind === 'directory' ? <Folder aria-hidden /> : <FileText aria-hidden />}
                  {entry.name}
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
