// Recents: GET /api/recents?kind= with an all/files/folders switch. Items carry
// storage_id/uri/kind/accessed_at only, so names derive from the URI tail.

import { useState } from 'react';
import { Clock, FileText, Folder } from 'lucide-react';
import { api } from '../api';
import { EmptyState, LoadingState, UnavailableState } from '../components/ui';
import { formatDate } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';

type RecentsKind = 'all' | 'file' | 'directory';

interface RecentsPageProps {
  t: TFunction;
  lang: string;
  onOpen: (storageId: string, uri: string) => void;
}

export function RecentsPage({ t, lang, onOpen }: RecentsPageProps) {
  const [kind, setKind] = useState<RecentsKind>('all');
  const recents = useApiResource(
    () => api.recents(kind === 'all' ? undefined : kind),
    [kind],
  );
  const items = recents.data?.items ?? [];

  const labelFor = (uri: string) => uri.split('/').filter(Boolean).pop() ?? uri;
  const parentFor = (uri: string) => {
    const trimmed = uri.replace(/\/+$/, '');
    const index = trimmed.lastIndexOf('/');
    return index > 0 ? trimmed.slice(0, index) : '';
  };

  let body;
  if (recents.loading && !recents.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (recents.unavailable) {
    body = <UnavailableState t={t} message={t('recentsUnavailable')} onRetry={recents.reload} />;
  } else if (items.length === 0) {
    body = <EmptyState icon={<Clock />} title={t('recentsEmpty')} hint={t('recentsNone')} />;
  } else {
    body = (
      <div className="file-list">
        <div className="list-head">
          <span>{t('filesName')}</span>
          <span>{t('filesType')}</span>
          <span>{t('filesModified')}</span>
          <span />
        </div>
        {items.map((item, index) => (
          <div
            className="file-row"
            key={`${item.storage_id}:${item.uri}:${index}`}
            tabIndex={0}
            data-nav
            onClick={() => onOpen(item.storage_id, parentFor(item.uri))}
          >
            <span className="file-name">
              {item.kind === 'directory' ? <Folder aria-hidden /> : <FileText aria-hidden />}
              {labelFor(item.uri)}
            </span>
            <span>
              {item.kind === 'directory' ? t('filesFilterFolders') : t('filesFilterFiles')}
            </span>
            <span>{formatDate(item.accessed_at, lang)}</span>
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
          <div className="eyebrow">{t('navLibraries')}</div>
          <h1>{t('recentsTitle')}</h1>
          <p>{t('recentsEmpty')}</p>
        </div>
      </div>
      <div className="filter-bar">
        <label className="select-control">
          <span>{t('recentsKind')}</span>
          <select
            value={kind}
            onChange={(event) => setKind(event.target.value as RecentsKind)}
            data-nav
          >
            <option value="all">{t('recentsKindAll')}</option>
            <option value="file">{t('recentsKindFile')}</option>
            <option value="directory">{t('recentsKindFolder')}</option>
          </select>
        </label>
      </div>
      {body}
    </>
  );
}
