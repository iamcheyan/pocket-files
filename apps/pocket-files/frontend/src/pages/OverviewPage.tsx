// Overview: device cards from GET /api/storage plus quick-access shortcuts.

import { ChevronRight, HardDrive, LayoutGrid, MoreHorizontal, Plus } from 'lucide-react';
import { api } from '../api';
import { useApiResource } from '../hooks/useApiResource';
import type { TFunction } from '../i18n';
import { formatBytes, percent } from '../format';
import type { StorageDevice } from '../types';
import { EmptyState, LoadingState } from '../components/ui';
import type { LibraryKey } from '../types';

const KIND_LABEL: Record<string, string> = {
  internal: 'deviceSystem',
  tf: 'deviceRemovableSd',
  usb: 'deviceRemovableUsb',
};

interface OverviewProps {
  t: TFunction;
  devices: StorageDevice[];
  devicesLoading: boolean;
  onOpenStorage: (id: string) => void;
  onOpenLibrary: (key: LibraryKey) => void;
  onBrowseFiles: (storageId: string, path: string) => void;
}

export function OverviewPage({
  t,
  devices,
  devicesLoading,
  onOpenStorage,
  onOpenLibrary,
  onBrowseFiles,
}: OverviewProps) {
  if (devicesLoading && devices.length === 0) {
    return <LoadingState label={t('commonLoading')} />;
  }

  const quickAccess: Array<{ key: LibraryKey; label: string }> = [
    { key: 'books', label: t('librariesBooks') },
    { key: 'music', label: t('librariesMusic') },
    { key: 'videos', label: t('librariesVideos') },
    { key: 'games', label: t('librariesGames') },
    { key: 'downloads', label: t('librariesDownloads') },
  ];

  return (
    <>
      <div className="welcome">
        <div>
          <div className="eyebrow">{t('overviewEyebrow')}</div>
          <h1>{t('overviewTitle')}</h1>
          <p>{t('overviewSubtitle')}</p>
        </div>
        <button
          type="button"
          className="primary"
          onClick={() => onBrowseFiles(devices[0]?.id ?? 'internal', '')}
          data-nav
        >
          <Plus aria-hidden /> {t('overviewBrowse')}
        </button>
      </div>

      {devices.length === 0 ? (
        <EmptyState icon={<HardDrive />} title={t('commonEmpty')} hint={t('storageUnavailable')} />
      ) : (
        <div className="device-grid">
          {devices.map((device, index) => {
            const usage = percent(device.used, device.total);
            return (
              <div className="device-card" key={device.id} data-nav>
                <div className="card-head">
                  <div className={'drive-icon drive-' + (index % 4)}>
                    <HardDrive aria-hidden />
                  </div>
                  {device.mock && <span className="mock-badge">{t('commonMock')}</span>}
                  <MoreHorizontal aria-hidden />
                </div>
                <h2>{device.name}</h2>
                <p>{t((KIND_LABEL[device.kind] ?? 'deviceOther') as 'deviceOther')}</p>
                <div className="meter">
                  <span style={{ width: `${usage}%` }} />
                </div>
                <div className="capacity">
                  <span>
                    {formatBytes(device.used)} {t('deviceUsed')}
                  </span>
                  <span>{formatBytes(device.total)}</span>
                </div>
                <div className="card-foot">
                  <span className="dot" /> {t('deviceReady')}
                  <span className="card-open">
                    <button
                      type="button"
                      className="link-btn"
                      onClick={() => onOpenStorage(device.id)}
                      data-nav
                    >
                      {t('deviceOpen')} <ChevronRight aria-hidden />
                    </button>
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      )}

      <div className="section-heading">
        <div>
          <h2>{t('overviewQuickAccess')}</h2>
          <p>{t('overviewQuickAccessSub')}</p>
        </div>
        <button type="button" className="text-btn" data-nav>
          {t('overviewEditLocations')}
        </button>
      </div>
      <div className="quick-grid">
        {quickAccess.map(({ key, label }) => (
          <button
            type="button"
            key={key}
            onClick={() => onOpenLibrary(key)}
            className="quick-card"
            data-nav
          >
            <LayoutGrid aria-hidden />
            <span>{label}</span>
            <ChevronRight aria-hidden />
          </button>
        ))}
      </div>

      <div className="notice">
        <span className="notice-icon">i</span>
        <div>
          <strong>{t('overviewNoticeTitle')}</strong>
          <p>{t('overviewNoticeBody')}</p>
        </div>
        <span className="mock-badge">{t('commonMockData')}</span>
      </div>
    </>
  );
}

export function useDevices() {
  return useApiResource(() => api.storage(), []);
}
