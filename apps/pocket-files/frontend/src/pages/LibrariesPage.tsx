// Virtual libraries: Books/Music/Videos/Games/Downloads backed by /api/views/*.
// 501 renders an explicit "not implemented" state, never placeholder rows.

import { useState } from 'react';
import { BookOpen, Download, FileText, Film, Folder, Gamepad2, Music } from 'lucide-react';
import { api } from '../api';
import { EmptyState, LoadingState, UnavailableState } from '../components/ui';
import { formatBytes, formatDate } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';
import type { LibraryKey } from '../types';

interface LibraryDef {
  key: LibraryKey;
  labelKey:
    | 'librariesBooks'
    | 'librariesMusic'
    | 'librariesVideos'
    | 'librariesGames'
    | 'librariesDownloads';
  icon: typeof BookOpen;
}

const LIBRARIES: LibraryDef[] = [
  { key: 'books', labelKey: 'librariesBooks', icon: BookOpen },
  { key: 'music', labelKey: 'librariesMusic', icon: Music },
  { key: 'videos', labelKey: 'librariesVideos', icon: Film },
  { key: 'games', labelKey: 'librariesGames', icon: Gamepad2 },
  { key: 'downloads', labelKey: 'librariesDownloads', icon: Download },
];

interface LibrariesPageProps {
  t: TFunction;
  lang: string;
  initial: LibraryKey;
  onOpenEntry: (storageId: string, uri: string) => void;
}

export function LibrariesPage({ t, lang, initial, onOpenEntry }: LibrariesPageProps) {
  const [active, setActive] = useState<LibraryKey>(initial);
  const view = useApiResource(() => api.view(active), [active]);
  const items = view.data?.items ?? [];

  let body;
  if (view.loading && !view.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (view.unavailable) {
    body = (
      <UnavailableState
        t={t}
        message={view.error?.status === 501 ? t('librariesNotImplemented') : t('librariesUnavailable')}
        onRetry={view.reload}
      />
    );
  } else if (view.error) {
    body = <UnavailableState t={t} message={view.error.message} onRetry={view.reload} />;
  } else if (items.length === 0) {
    body = <EmptyState icon={<Folder />} title={t('commonEmpty')} hint={t('librariesUnavailable')} />;
  } else {
    body = (
      <div className="file-list">
        <div className="list-head">
          <span>{t('filesName')}</span>
          <span>{t('filesSize')}</span>
          <span>{t('filesModified')}</span>
          <span />
        </div>
        {items.map((entry) => (
          <div
            className="file-row"
            key={entry.id}
            tabIndex={0}
            data-nav
            onClick={() => onOpenEntry(entry.storage_id, entry.id)}
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
          <div className="eyebrow">{t('navLibraries')}</div>
          <h1>{t('librariesTitle')}</h1>
        </div>
      </div>
      <div className="library-tabs">
        {LIBRARIES.map(({ key, labelKey, icon: Icon }) => (
          <button
            type="button"
            key={key}
            className={'library-tab' + (active === key ? ' active' : '')}
            onClick={() => setActive(key)}
            data-nav
          >
            <Icon aria-hidden />
            <span>{t(labelKey)}</span>
          </button>
        ))}
      </div>
      {body}
    </>
  );
}
