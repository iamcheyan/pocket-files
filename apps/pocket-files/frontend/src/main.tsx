// Application shell: sidebar navigation, topbar, page routing and shared state.
// Navigation is a plain discriminated union held in React state — no router
// dependency (registry is offline) and no browser history assumptions.

import { useCallback, useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  ArrowLeftRight,
  BarChart3,
  Clock,
  Folder,
  Grid2X2,
  HardDrive,
  LayoutGrid,
  Menu,
  MoreHorizontal,
  PanelLeftClose,
  Search,
  Settings,
  Star,
} from 'lucide-react';
import './style.css';
import { api } from './api';
import { translate, type TFunction } from './i18n';
import { useDevices } from './pages/OverviewPage';
import { OverviewPage } from './pages/OverviewPage';
import { FilesPage } from './pages/FilesPage';
import { StoragePage } from './pages/StoragePage';
import { TransfersPage } from './pages/TransfersPage';
import { SearchPage } from './pages/SearchPage';
import { FavoritesPage } from './pages/FavoritesPage';
import { RecentsPage } from './pages/RecentsPage';
import { LibrariesPage } from './pages/LibrariesPage';
import { AnalyticsPage } from './pages/AnalyticsPage';
import { SettingsPage } from './pages/SettingsPage';
import { usePreferences } from './hooks/usePreferences';
import { useKeyboardNav } from './hooks/useKeyboardNav';
import type { LibraryKey, PageId } from './types';

interface NavTarget {
  page: PageId;
  storageId: string;
  path: string;
  library: LibraryKey;
  query: string;
}

const INITIAL_NAV: NavTarget = {
  page: 'overview',
  storageId: 'internal',
  path: '',
  library: 'books',
  query: '',
};

const LIBRARY_LABEL: Record<LibraryKey, 'librariesBooks' | 'librariesMusic' | 'librariesVideos' | 'librariesGames' | 'librariesDownloads'> = {
  books: 'librariesBooks',
  music: 'librariesMusic',
  videos: 'librariesVideos',
  games: 'librariesGames',
  downloads: 'librariesDownloads',
};

function App() {
  const prefs = usePreferences();
  const t = useCallback<TFunction>((key, params) => translate(prefs.lang, key, params), [prefs.lang]);

  const [nav, setNav] = useState<NavTarget>(INITIAL_NAV);
  const [query, setQuery] = useState('');
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [health, setHealth] = useState<string | null>(null);

  const devices = useDevices();
  const list = devices.data ?? [];
  const currentDevice = list.find((device) => device.id === nav.storageId);

  useEffect(() => {
    api
      .health()
      .then((res) => setHealth(res.mode))
      .catch(() => setHealth(null));
  }, []);

  const navigate = useCallback((patch: Partial<NavTarget>) => {
    setNav((current) => ({ ...current, ...patch }));
    setSidebarOpen(false);
  }, []);

  const goFiles = useCallback(
    (storageId: string, path: string) => navigate({ page: 'files', storageId, path }),
    [navigate],
  );

  const goStorage = useCallback(
    (storageId: string) => navigate({ page: 'storage', storageId, path: '' }),
    [navigate],
  );

  const openLibrary = useCallback(
    (library: LibraryKey) => navigate({ page: 'libraries', library }),
    [navigate],
  );

  // Escape = "back": folder up inside Files, otherwise overview.
  const handleBack = useCallback(() => {
    if (nav.page === 'files' && nav.path !== '') {
      const trimmed = nav.path.replace(/\/+$/, '');
      const idx = trimmed.lastIndexOf('/');
      navigate({ path: idx <= 0 ? '' : trimmed.slice(0, idx) });
      return;
    }
    if (nav.page !== 'overview') navigate({ page: 'overview' });
  }, [nav.page, nav.path, navigate]);

  useKeyboardNav({ onBack: handleBack });

  // Cmd/Ctrl+K focuses the global search field.
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        document.querySelector<HTMLInputElement>('.search input')?.focus();
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);

  const submitSearch = (event: React.FormEvent) => {
    event.preventDefault();
    navigate({ page: 'search', query: query.trim() });
  };

  const storageNav = useMemo(
    () =>
      list.map((device) => ({
        id: device.id,
        name: device.name,
        mock: device.mock,
      })),
    [list],
  );

  const navItems: Array<{ id: PageId; label: string; icon: typeof Folder }> = [
    { id: 'overview', label: t('navOverview'), icon: LayoutGrid },
    { id: 'files', label: t('navFiles'), icon: Folder },
    { id: 'transfers', label: t('navTransfers'), icon: ArrowLeftRight },
    { id: 'favorites', label: t('navFavorites'), icon: Star },
    { id: 'recents', label: t('navRecents'), icon: Clock },
    { id: 'libraries', label: t('navLibraries'), icon: Grid2X2 },
    { id: 'analytics', label: t('navAnalytics'), icon: BarChart3 },
    { id: 'settings', label: t('navSettings'), icon: Settings },
  ];

  const breadcrumb =
    nav.page === 'files'
      ? [currentDevice?.name ?? t('navFiles'), ...(nav.path ? [nav.path] : [])]
      : [labelFor(nav, t)];

  return (
    <main className="app" data-nav-scope="app">
      <aside className={'sidebar' + (sidebarOpen ? ' open' : '')}>
        <div className="brand">
          <div className="brand-icon">
            <Grid2X2 aria-hidden />
          </div>
          <div>
            <strong>{t('appName')}</strong>
            <small>{t('appTagline')}</small>
          </div>
          <button
            type="button"
            className="ghost collapse"
            onClick={() => setSidebarOpen((open) => !open)}
            aria-label={sidebarOpen ? t('sidebarCollapse') : t('sidebarExpand')}
            data-nav
          >
            <PanelLeftClose aria-hidden />
          </button>
        </div>

        <div className="nav-label">{t('navStorage')}</div>
        {storageNav.map((device) => (
          <button
            type="button"
            key={device.id}
            className={'nav-item' + (nav.page === 'files' || nav.page === 'storage' ? '' : '') + (nav.storageId === device.id ? ' current' : '')}
            onClick={() => goFiles(device.id, '')}
            onDoubleClick={() => goStorage(device.id)}
            data-nav
          >
            <HardDrive aria-hidden />
            <span>{device.name}</span>
            {device.mock && <i aria-hidden />}
          </button>
        ))}
        {storageNav.length === 0 && (
          <button type="button" className="nav-item" onClick={() => navigate({ page: 'files' })} data-nav>
            <HardDrive aria-hidden />
            <span>{t('navFiles')}</span>
          </button>
        )}

        <div className="nav-label library-label">{t('navLibraries')}</div>
        {(['books', 'music', 'videos', 'games', 'downloads'] as LibraryKey[]).map((key) => (
          <button
            type="button"
            key={key}
            className={'nav-item' + (nav.page === 'libraries' && nav.library === key ? ' active' : '')}
            onClick={() => openLibrary(key)}
            data-nav
          >
            <Folder aria-hidden />
            <span>{t(LIBRARY_LABEL[key])}</span>
          </button>
        ))}

        <div className="nav-label library-label">{t('navOverview')}</div>
        {navItems.map(({ id, label, icon: Icon }) => (
          <button
            type="button"
            key={id}
            className={'nav-item' + (nav.page === id ? ' active' : '')}
            onClick={() => navigate({ page: id })}
            data-nav
          >
            <Icon aria-hidden />
            <span>{label}</span>
          </button>
        ))}

        <div className="sidebar-bottom">
          <span className="avatar">P</span>
          <div>
            <strong>{t('accountName')}</strong>
            <small>{t('accountProfile')}</small>
          </div>
          <MoreHorizontal aria-hidden />
        </div>
      </aside>

      <section className="content">
        <header className="topbar">
          <div className="crumb">
            <button
              type="button"
              className="ghost menu-toggle"
              onClick={() => setSidebarOpen((open) => !open)}
              aria-label={sidebarOpen ? t('sidebarCollapse') : t('sidebarExpand')}
              data-nav
            >
              <Menu aria-hidden />
            </button>
            {breadcrumb.map((part, index) => (
              <span key={`${part}-${index}`} className={index === breadcrumb.length - 1 ? 'crumb-current' : ''}>
                {index > 0 && <span className="crumb-sep">/</span>}
                {part}
              </span>
            ))}
          </div>
          <div className="top-actions">
            <form className="search" onSubmit={submitSearch}>
              <Search aria-hidden />
              <input
                placeholder={t('commonSearch')}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                aria-label={t('commonSearch')}
                data-nav
              />
              <kbd>⌘K</kbd>
            </form>
            <button
              type="button"
              className="round"
              onClick={() => prefs.setTheme(prefs.theme === 'oled' ? 'system' : 'oled')}
              aria-label={t('settingsTheme')}
              data-nav
            >
              ◐
            </button>
          </div>
        </header>

        <div className="page-body">
          {nav.page === 'overview' && (
            <OverviewPage
              t={t}
              devices={list}
              devicesLoading={devices.loading}
              onOpenStorage={goStorage}
              onOpenLibrary={openLibrary}
              onBrowseFiles={goFiles}
            />
          )}
          {nav.page === 'files' && (
            <FilesPage
              t={t}
              lang={prefs.lang}
              storageId={nav.storageId}
              devices={(devices.data ?? []).map((device) => ({ id: device.id, name: device.name, status: device.status }))}
              storageName={currentDevice?.name ?? ''}
              path={nav.path}
              onPathChange={(path) => navigate({ path })}
            />
          )}
          {nav.page === 'storage' && (
            <StoragePage
              t={t}
              device={currentDevice}
              onBrowse={(storageId) => goFiles(storageId, '')}
            />
          )}
          {nav.page === 'transfers' && <TransfersPage t={t} />}
          {nav.page === 'search' && (
            <SearchPage
              t={t}
              lang={prefs.lang}
              storageId={nav.storageId}
              initialQuery={nav.query}
              onOpen={goFiles}
            />
          )}
          {nav.page === 'favorites' && (
            <FavoritesPage
              t={t}
              onOpen={(storageId, uri) => goFiles(storageId, uri.includes('/') ? uri.slice(0, uri.lastIndexOf('/')) : '')}
            />
          )}
          {nav.page === 'recents' && (
            <RecentsPage t={t} lang={prefs.lang} onOpen={goFiles} />
          )}
          {nav.page === 'libraries' && (
            <LibrariesPage
              t={t}
              lang={prefs.lang}
              initial={nav.library}
              onOpenEntry={(storageId, uri) =>
                goFiles(storageId, uri.includes('/') ? uri.slice(0, uri.lastIndexOf('/')) : '')
              }
            />
          )}
          {nav.page === 'analytics' && (
            <AnalyticsPage t={t} storageId={nav.storageId} storageName={currentDevice?.name ?? ''} />
          )}
          {nav.page === 'settings' && (
            <SettingsPage
              t={t}
              theme={prefs.theme}
              onThemeChange={prefs.setTheme}
              lang={prefs.lang}
              onLangChange={prefs.setLang}
              profile={prefs.profile}
              onProfileChange={prefs.setProfile}
              resolvedProfile={prefs.resolvedProfile}
              dpi={prefs.dpi}
            />
          )}
        </div>
      </section>
    </main>
  );
}

function labelFor(nav: NavTarget, t: TFunction): string {
  switch (nav.page) {
    case 'overview':
      return t('navOverview');
    case 'files':
      return t('navFiles');
    case 'storage':
      return t('storageTitle');
    case 'transfers':
      return t('transfersTitle');
    case 'search':
      return t('searchTitle');
    case 'favorites':
      return t('favoritesTitle');
    case 'recents':
      return t('recentsTitle');
    case 'libraries':
      return t(LIBRARY_LABEL[nav.library]);
    case 'analytics':
      return t('analyticsTitle');
    case 'settings':
      return t('settingsTitle');
    default:
      return t('appName');
  }
}

createRoot(document.getElementById('root')!).render(<App />);
