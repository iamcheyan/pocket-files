// Storage analytics: GET /api/analytics?storage_id=. Renders the aggregates the
// backend actually returns (category_counts, largest files/directories, usage).
// Unavailable state instead of invented statistics.

import { BarChart3 } from 'lucide-react';
import { api } from '../api';
import { EmptyState, LoadingState, UnavailableState } from '../components/ui';
import { formatBytes } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';

interface AnalyticsPageProps {
  t: TFunction;
  storageId: string;
  storageName: string;
}

export function AnalyticsPage({ t, storageId, storageName }: AnalyticsPageProps) {
  const analytics = useApiResource(() => api.analytics(storageId), [storageId]);
  const data = analytics.data;

  let body;
  if (analytics.loading && !analytics.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (analytics.unavailable) {
    body = <UnavailableState t={t} message={t('analyticsUnavailable')} onRetry={analytics.reload} />;
  } else if (!data) {
    body = <EmptyState icon={<BarChart3 />} title={t('commonEmpty')} hint={t('analyticsUnavailable')} />;
  } else {
    const categories = Object.entries(data.category_counts ?? {});
    const maxCategory = Math.max(1, ...categories.map(([, count]) => count));
    const largest = data.largest_files ?? [];
    body = (
      <>
        <div className="stat-grid wide">
          <div className="stat">
            <span className="stat-label">{t('storageUsed')}</span>
            <strong>{formatBytes(data.used)}</strong>
          </div>
          <div className="stat">
            <span className="stat-label">{t('storageFree')}</span>
            <strong>{formatBytes(data.free)}</strong>
          </div>
          <div className="stat">
            <span className="stat-label">{t('storageTotal')}</span>
            <strong>{formatBytes(data.total)}</strong>
          </div>
        </div>

        <div className="settings-card">
          <h2>{t('analyticsByKind')}</h2>
          {categories.length === 0 ? (
            <p className="muted">{t('commonEmpty')}</p>
          ) : (
            <div className="bar-list">
              {categories.map(([kind, count]) => (
                <div className="bar-row" key={kind}>
                  <span>{kind}</span>
                  <span className="meter">
                    <span style={{ width: `${(count / maxCategory) * 100}%` }} />
                  </span>
                  <em>{count}</em>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="settings-card">
          <h2>{t('analyticsLargest')}</h2>
          {largest.length === 0 ? (
            <p className="muted">{t('commonEmpty')}</p>
          ) : (
            <div className="bar-list">
              {largest.slice(0, 10).map((file, index) => (
                <div className="bar-row" key={`${file.name ?? 'file'}-${index}`}>
                  <span className="ellipsis">{file.name ?? file.id ?? '—'}</span>
                  <span />
                  <em>{formatBytes(file.size ?? 0)}</em>
                </div>
              ))}
            </div>
          )}
        </div>
      </>
    );
  }

  return (
    <>
      <div className="welcome compact">
        <div>
          <div className="eyebrow">{storageName}</div>
          <h1>{t('analyticsTitle')}</h1>
          <p>{storageName}</p>
        </div>
      </div>
      {body}
    </>
  );
}
