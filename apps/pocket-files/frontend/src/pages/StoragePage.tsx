// Storage detail: usage from GET /api/storage/{id}/usage with an explicit
// unavailable state when the provider cannot report capacity.

import { HardDrive } from 'lucide-react';
import { api } from '../api';
import { ErrorState, LoadingState, MockBadge, UnavailableState } from '../components/ui';
import { formatBytes, percent } from '../format';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';
import type { StorageDevice } from '../types';

const KIND_LABEL: Record<string, string> = {
  internal: 'storageKindInternal',
  tf: 'storageKindTf',
  usb: 'storageKindUsb',
};

interface StoragePageProps {
  t: TFunction;
  device: StorageDevice | undefined;
  onBrowse: (storageId: string) => void;
}

export function StoragePage({ t, device, onBrowse }: StoragePageProps) {
  const storageId = device?.id ?? '';
  const usage = useApiResource(
    () => api.usage(storageId),
    [storageId],
    { enabled: storageId !== '' },
  );

  if (!device) {
    return <UnavailableState t={t} message={t('storageUnavailable')} />;
  }

  const total = usage.data?.total ?? device.total;
  const used = usage.data?.used ?? device.used;
  const free = usage.data?.free ?? Math.max(0, total - used);
  const ratio = percent(used, total);

  let body;
  if (usage.loading && !usage.data) {
    body = <LoadingState label={t('commonLoading')} />;
  } else if (usage.unavailable || usage.error) {
    // Capacity still comes from the device record; only the live usage call
    // failed, so show what we know and mark the detail unavailable.
    body = (
      <UnavailableState
        t={t}
        message={t('storageUnavailable')}
        onRetry={usage.reload}
      />
    );
  } else {
    body = (
      <div className="settings-card">
        <h2>{t('storageUsage')}</h2>
        <div className="usage-meter">
          <div className="meter tall">
            <span style={{ width: `${ratio}%` }} />
          </div>
          <div className="capacity">
            <span>
              {formatBytes(used)} {t('storageUsed')}
            </span>
            <span>{formatBytes(total)}</span>
          </div>
        </div>
        <div className="stat-grid">
          <div className="stat">
            <span className="stat-label">{t('storageUsed')}</span>
            <strong>{formatBytes(used)}</strong>
          </div>
          <div className="stat">
            <span className="stat-label">{t('storageFree')}</span>
            <strong>{formatBytes(free)}</strong>
          </div>
          <div className="stat">
            <span className="stat-label">{t('storageTotal')}</span>
            <strong>{formatBytes(total)}</strong>
          </div>
          <div className="stat">
            <span className="stat-label">{t('deviceUsed')}</span>
            <strong>{ratio.toFixed(1)}%</strong>
          </div>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="welcome compact">
        <div>
          <div className="eyebrow">
            {t((KIND_LABEL[device.kind] ?? 'deviceOther') as 'deviceOther')}
            {device.mock ? ` · ${t('commonMockStorage')}` : ''}
          </div>
          <h1>
            <HardDrive className="inline-icon" aria-hidden /> {device.name}
          </h1>
          <p>
            {device.read_only ? t('storageReadOnly') : t('deviceReady')}
            {device.mock ? ` · ${t('storageMockNote')}` : ''}
          </p>
        </div>
        <div className="welcome-actions">
          {device.mock && <MockBadge label={t('commonMock')} />}
          <button type="button" className="primary" onClick={() => onBrowse(device.id)} data-nav>
            {t('storageBrowse')}
          </button>
        </div>
      </div>
      {body}
    </>
  );
}
