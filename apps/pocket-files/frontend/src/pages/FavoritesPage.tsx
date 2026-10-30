// Favorites: GET /api/favorites, DELETE with {storage_id,uri} body.
// The backend stores URIs only, so labels derive from the URI tail.

import { Star, X } from 'lucide-react';
import { api } from '../api';
import { EmptyState, LoadingState, UnavailableState } from '../components/ui';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';

interface FavoritesPageProps {
  t: TFunction;
  onOpen: (storageId: string, uri: string) => void;
}

interface FavoriteRow {
  storage_id: string;
  uri: string;
}

export function FavoritesPage({ t, onOpen }: FavoritesPageProps) {
  const favorites = useApiResource(() => api.favorites(), []);
  const items: FavoriteRow[] = favorites.data?.items ?? [];

  const remove = async (row: FavoriteRow) => {
    try {
      await api.removeFavorite(row.storage_id, row.uri);
    } finally {
      favorites.reload();
    }
  };

  const labelFor = (uri: string) => uri.split('/').filter(Boolean).pop() ?? uri;
  const parentFor = (uri: string) => {
    const trimmed = uri.replace(/\/+$/, '');
    const index = trimmed.lastIndexOf('/');
    return index > 0 ? trimmed.slice(0, index) : '';
  };

  let body;
  if (favorites.loading && !favorites.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (favorites.unavailable) {
    body = (
      <UnavailableState t={t} message={t('favoritesUnavailable')} onRetry={favorites.reload} />
    );
  } else if (items.length === 0) {
    body = <EmptyState icon={<Star />} title={t('favoritesEmpty')} hint={t('favoritesNone')} />;
  } else {
    body = (
      <div className="file-list">
        <div className="list-head">
          <span>{t('filesName')}</span>
          <span>{t('navStorage')}</span>
          <span>{t('filesType')}</span>
          <span />
        </div>
        {items.map((row) => (
          <div className="file-row" key={`${row.storage_id}:${row.uri}`} tabIndex={0} data-nav>
            <span className="file-name">
              <Star aria-hidden />
              <button
                type="button"
                className="link-btn"
                onClick={() => onOpen(row.storage_id, parentFor(row.uri))}
                data-nav
              >
                {labelFor(row.uri)}
              </button>
            </span>
            <span>{row.storage_id}</span>
            <span className="ellipsis">{row.uri}</span>
            <span className="row-actions">
              <button
                type="button"
                className="icon-btn"
                title={t('favoritesRemove')}
                aria-label={t('favoritesRemove')}
                onClick={() => void remove(row)}
                data-nav
              >
                <X aria-hidden />
              </button>
            </span>
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
          <h1>{t('favoritesTitle')}</h1>
          <p>{t('favoritesEmpty')}</p>
        </div>
      </div>
      {body}
    </>
  );
}
