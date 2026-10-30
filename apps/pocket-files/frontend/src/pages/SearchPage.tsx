// Search page: reuses /api/files with q + server-side type/min_size/max_size
// filters (filesd supports all of them), so results match the browser exactly.

import { useState } from 'react';
import { FileText, Folder, Search } from 'lucide-react';
import { api } from '../api';
import { EmptyState, LoadingState, UnavailableState } from '../components/ui';
import { formatBytes, formatDate, parseSizeToBytes } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';
import type { FileTypeFilter } from '../types';

interface SearchPageProps {
  t: TFunction;
  lang: string;
  storageId: string;
  initialQuery: string;
  onOpen: (storageId: string, path: string) => void;
}

export function SearchPage({ t, lang, storageId, initialQuery, onOpen }: SearchPageProps) {
  const [term, setTerm] = useState(initialQuery);
  const [submitted, setSubmitted] = useState(initialQuery);
  const [typeFilter, setTypeFilter] = useState<FileTypeFilter>('all');
  const [minSize, setMinSize] = useState('');
  const [maxSize, setMaxSize] = useState('');

  const results = useApiResource(
    () =>
      submitted
        ? api.files({
            storage_id: storageId,
            q: submitted,
            type: typeFilter === 'all' ? undefined : typeFilter === 'folders' ? 'directory' : 'file',
            min_size: parseSizeToBytes(minSize) ?? undefined,
            max_size: parseSizeToBytes(maxSize) ?? undefined,
            limit: 200,
          })
        : Promise.resolve({ items: [] }),
    [storageId, submitted, typeFilter, minSize, maxSize],
  );

  const rows = results.data?.items ?? [];

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitted(term.trim());
  };

  let body;
  if (results.loading) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (results.unavailable) {
    body = <UnavailableState t={t} message={t('searchUnavailable')} onRetry={results.reload} />;
  } else if (!submitted) {
    body = <EmptyState icon={<Search />} title={t('searchTitle')} hint={t('searchPlaceholder')} />;
  } else if (rows.length === 0) {
    body = (
      <EmptyState icon={<Search />} title={t('searchEmpty')} hint={t('commonResults', { n: 0 })} />
    );
  } else {
    body = (
      <div className="file-list">
        <div className="list-head">
          <span>{t('filesName')}</span>
          <span>{t('filesSize')}</span>
          <span>{t('filesModified')}</span>
          <span />
        </div>
        {rows.map((entry) => (
          <div
            className="file-row"
            key={entry.id}
            tabIndex={0}
            data-nav
            onClick={() => entry.kind === 'directory' && onOpen(entry.storage_id, entry.id)}
          >
            <span className="file-name">
              {entry.kind === 'directory' ? <Folder aria-hidden /> : <FileText aria-hidden />}
              {entry.name}
            </span>
            <span>{entry.kind === 'directory' ? '—' : formatBytes(entry.size)}</span>
            <span>{formatDate(entry.modified, lang)}</span>
            <span />
          </div>
        ))}
      </div>
    );
  }

  return (
    <>
      <div className="welcome compact">
        <div>
          <div className="eyebrow">{t('navOverview')}</div>
          <h1>{t('searchTitle')}</h1>
          <p>{t('commonResults', { n: rows.length })}</p>
        </div>
      </div>

      <form className="filter-bar" onSubmit={submit}>
        <label className="search grow">
          <Search aria-hidden />
          <input
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder={t('searchPlaceholder')}
            data-nav
          />
        </label>
        <label className="select-control">
          <span>{t('searchType')}</span>
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
        <label className="select-control">
          <span>{t('searchMinSize')}</span>
          <input
            value={minSize}
            onChange={(event) => setMinSize(event.target.value)}
            placeholder={t('searchMinSize')}
            data-nav
          />
        </label>
        <label className="select-control">
          <span>{t('searchMaxSize')}</span>
          <input
            value={maxSize}
            onChange={(event) => setMaxSize(event.target.value)}
            placeholder={t('searchMaxSizeHint')}
            data-nav
          />
        </label>
        <button type="submit" className="primary" data-nav>
          {t('searchSubmit')}
        </button>
      </form>

      {body}
    </>
  );
}
