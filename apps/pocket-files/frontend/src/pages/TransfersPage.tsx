// Transfers: polls GET /api/transfers every 2s and exposes pause/resume/cancel/
// retry actions via POST /api/transfers/{id}/{action}. Empty list is a valid
// state; failure is explicit.

import { ArrowLeftRight, Pause, Play, RefreshCw, X } from 'lucide-react';
import { api } from '../api';
import { EmptyState, LoadingState, UnavailableState } from '../components/ui';
import { formatBytes } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';
import type { TransferAction, TransferJob } from '../types';

const STATUS_LABEL: Record<string, string> = {
  queued: 'transfersStatusQueued',
  running: 'transfersStatusActive',
  paused: 'transfersStatusPaused',
  completed: 'transfersStatusCompleted',
  failed: 'transfersStatusFailed',
  cancelled: 'transfersStatusCanceled',
  conflict: 'transfersStatusFailed',
};

const STATUS_ACTIONS: Record<string, TransferAction[]> = {
  queued: ['cancel'],
  running: ['pause', 'cancel'],
  paused: ['resume', 'cancel'],
  failed: ['retry', 'cancel'],
  conflict: ['resume', 'cancel'],
  completed: [],
  cancelled: ['retry'],
};

const ACTION_ICON = {
  pause: Pause,
  resume: Play,
  cancel: X,
  retry: RefreshCw,
} as const;

const ACTION_LABEL = {
  pause: 'transfersPause',
  resume: 'transfersResume',
  cancel: 'transfersCancel',
  retry: 'transfersRetry',
} as const;

interface TransfersPageProps {
  t: TFunction;
}

export function TransfersPage({ t }: TransfersPageProps) {
  const transfers = useApiResource(() => api.transfers(), [], { intervalMs: 2000 });

  const runAction = async (job: TransferJob, action: TransferAction) => {
    try {
      await api.transferAction(job.id, action);
    } finally {
      // Polling is authoritative; avoid optimistic mutation.
      transfers.reload();
    }
  };

  const items = transfers.data?.items ?? [];

  let body;
  if (transfers.loading && !transfers.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (transfers.unavailable) {
    body = (
      <UnavailableState t={t} message={t('transfersUnavailable')} onRetry={transfers.reload} />
    );
  } else if (items.length === 0) {
    body = (
      <EmptyState icon={<ArrowLeftRight />} title={t('transfersEmpty')} hint={t('transfersNone')} />
    );
  } else {
    body = (
      <div className="file-list">
        <div className="list-head transfers-head">
          <span>{t('filesName')}</span>
          <span>{t('transfersStatus')}</span>
          <span>{t('filesSize')}</span>
          <span />
        </div>
        {items.map((job) => {
          const status = String(job.status);
          const total = job.total_bytes ?? 0;
          const done = job.bytes_done ?? 0;
          const progress = total > 0 ? Math.min(100, (done / total) * 100) : 0;
          const actions = STATUS_ACTIONS[status] ?? [];
          const label =
            job.items?.[0]?.dst ??
            job.items?.[0]?.src ??
            job.id;
          return (
            <div className="file-row transfers-row" key={job.id}>
              <span className="file-name ellipsis">{label}</span>
              <span className={'status-pill status-' + status}>
                {t((STATUS_LABEL[status] ?? 'transfersStatusQueued') as 'transfersStatusQueued')}
              </span>
              <span className="progress-cell">
                <span className="meter">
                  <span style={{ width: `${progress}%` }} />
                </span>
                <em>
                  {formatBytes(done)} / {formatBytes(total)}
                  {status === 'running' && (job.speed ?? 0) > 0
                    ? ` · ${formatBytes(job.speed ?? 0)}/s`
                    : ''}
                </em>
              </span>
              <span className="row-actions">
                {actions.map((action) => {
                  const Icon = ACTION_ICON[action];
                  return (
                    <button
                      type="button"
                      key={action}
                      className="icon-btn"
                      title={t(ACTION_LABEL[action])}
                      aria-label={t(ACTION_LABEL[action])}
                      onClick={() => void runAction(job, action)}
                      data-nav
                    >
                      <Icon aria-hidden />
                    </button>
                  );
                })}
              </span>
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <>
      <div className="welcome compact">
        <div>
          <div className="eyebrow">{t('navTransfers')}</div>
          <h1>{t('transfersTitle')}</h1>
        </div>
      </div>
      {body}
    </>
  );
}
