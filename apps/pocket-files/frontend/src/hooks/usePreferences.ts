// User preferences: theme, language, device profile and DPI.
// Everything is attribute-driven so emulators and small screens behave the
// same without user-agent sniffing. Preferences persist in localStorage.

import { useCallback, useEffect, useMemo, useState } from 'react';
import { LANGS, type Lang } from '../i18n';

export type ThemePreference = 'system' | 'light' | 'dark' | 'oled';
export type DeviceProfile = 'desktop' | 'handheld' | 'reader' | 'dap';
export type ProfilePreference = 'auto' | DeviceProfile;
export type DpiToken = '1x' | '1.5x' | '2x' | '3x';

export const THEMES: readonly ThemePreference[] = ['system', 'light', 'dark', 'oled'];
export const DPI_TOKENS: readonly DpiToken[] = ['1x', '1.5x', '2x', '3x'];
export type DpiPreference = 'auto' | DpiToken;
export const DPI_PREFERENCES: readonly DpiPreference[] = ['auto', ...DPI_TOKENS];

export const PROFILES: readonly ProfilePreference[] = [
  'auto',
  'desktop',
  'handheld',
  'reader',
  'dap',
];

const THEME_KEY = 'pocket-files-theme';
const LANG_KEY = 'pocket-files-lang';
const PROFILE_KEY = 'pocket-files-profile';
const DPI_KEY = 'pocket-files-dpi';

function readStored<T extends string>(key: string, allowed: readonly T[], fallback: T): T {
  try {
    const value = window.localStorage.getItem(key);
    if (value && (allowed as readonly string[]).includes(value)) return value as T;
  } catch {
    // localStorage may be blocked; fall through to default.
  }
  return fallback;
}

function detectLang(): Lang {
  try {
    const nav = window.navigator.language.toLowerCase();
    if (nav.startsWith('zh')) return 'zh';
    if (nav.startsWith('ja')) return 'ja';
  } catch {
    // ignore
  }
  return 'en';
}

// Viewport buckets, matched to the reference devices:
//   handheld — RG34XX-class panels (<= 480px wide)
//   dap      — very short landscape screens (<= 360px tall), e.g. M21
//   reader   — medium portrait panels, e-ink tablets (<= 900px wide)
//   desktop  — everything else
// 640x480 and 720x720 therefore land on reader/desktop instead of being
// mistaken for a DAP.
export function detectProfile(width: number, height: number): DeviceProfile {
  if (width <= 480) return 'handheld';
  if (height <= 360) return 'dap';
  if (width <= 900 || height <= 560) return 'reader';
  return 'desktop';
}

// DPI tokens map to the four density scales Pocket devices ship with. The bucket
// follows the reported device pixel ratio so 1x/1.5x/2x/3x panels each get their
// own typography and row density.
export function detectDpi(): DpiToken {
  const ratio = window.devicePixelRatio || 1;
  if (ratio >= 2.75) return '3x';
  if (ratio >= 1.75) return '2x';
  if (ratio >= 1.25) return '1.5x';
  return '1x';
}

const THEME_COLORS: Record<Exclude<ThemePreference, 'system'>, string> = {
  light: '#f7f8fa',
  dark: '#101114',
  oled: '#000000',
};

export interface Preferences {
  theme: ThemePreference;
  setTheme: (theme: ThemePreference) => void;
  lang: Lang;
  setLang: (lang: Lang) => void;
  profile: ProfilePreference;
  setProfile: (profile: ProfilePreference) => void;
  resolvedProfile: DeviceProfile;
  dpi: DpiToken;
  /** `auto` follows devicePixelRatio; an explicit token pins the density. */
  dpiPreference: DpiPreference;
  setDpiPreference: (value: DpiPreference) => void;
}

export function usePreferences(): Preferences {
  const [theme, setThemeState] = useState<ThemePreference>(() =>
    readStored(THEME_KEY, THEMES, 'system'),
  );
  const [lang, setLangState] = useState<Lang>(() => readStored(LANG_KEY, LANGS, detectLang()));
  const [profile, setProfileState] = useState<ProfilePreference>(() =>
    readStored(PROFILE_KEY, PROFILES, 'auto'),
  );
  const [viewport, setViewport] = useState(() => ({
    width: window.innerWidth,
    height: window.innerHeight,
  }));
  const [dpiPreference, setDpiPreferenceState] = useState<DpiPreference>(() =>
    readStored<DpiPreference>(DPI_KEY, DPI_PREFERENCES, 'auto'),
  );
  const [detectedDpi, setDetectedDpi] = useState<DpiToken>(detectDpi);
  const dpi: DpiToken = dpiPreference === 'auto' ? detectedDpi : dpiPreference;

  useEffect(() => {
    const onResize = () => {
      setViewport({ width: window.innerWidth, height: window.innerHeight });
      setDetectedDpi(detectDpi());
    };
    window.addEventListener('resize', onResize);
    window.addEventListener('orientationchange', onResize);
    return () => {
      window.removeEventListener('resize', onResize);
      window.removeEventListener('orientationchange', onResize);
    };
  }, []);

  const resolvedProfile = useMemo(
    () => (profile === 'auto' ? detectProfile(viewport.width, viewport.height) : profile),
    [profile, viewport.width, viewport.height],
  );

  useEffect(() => {
    const root = document.documentElement;
    root.dataset.theme = theme;
    root.dataset.profile = resolvedProfile;
    root.dataset.dpi = dpi;
    root.dataset.profilePreference = profile;
    root.dataset.lang = lang;
    // Real BCP-47 lang for screen readers and CJK font selection.
    root.lang = lang === 'zh' ? 'zh-CN' : lang === 'ja' ? 'ja-JP' : 'en';

    const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]');
    if (meta) {
      const isSystemDark =
        theme === 'system' &&
        window.matchMedia?.('(prefers-color-scheme: dark)').matches === true;
      meta.content =
        theme === 'system'
          ? isSystemDark
            ? THEME_COLORS.dark
            : THEME_COLORS.light
          : THEME_COLORS[theme];
    }
  }, [theme, resolvedProfile, dpi, profile, lang]);

  const setDpiPreference = useCallback((next: DpiPreference) => {
    setDpiPreferenceState(next);
    try {
      window.localStorage.setItem(DPI_KEY, next);
    } catch {
      // Storage may be blocked; keep the in-memory value.
    }
  }, []);

  const setTheme = useCallback((next: ThemePreference) => {
    setThemeState(next);
    try {
      window.localStorage.setItem(THEME_KEY, next);
    } catch {
      // ignore persistence failure
    }
  }, []);

  const setLang = useCallback((next: Lang) => {
    setLangState(next);
    try {
      window.localStorage.setItem(LANG_KEY, next);
    } catch {
      // ignore persistence failure
    }
  }, []);

  const setProfile = useCallback((next: ProfilePreference) => {
    setProfileState(next);
    try {
      window.localStorage.setItem(PROFILE_KEY, next);
    } catch {
      // ignore persistence failure
    }
  }, []);

  return {
    theme,
    setTheme,
    lang,
    setLang,
    profile,
    setProfile,
    resolvedProfile,
    dpi,
    dpiPreference,
    setDpiPreference,
  };
}
