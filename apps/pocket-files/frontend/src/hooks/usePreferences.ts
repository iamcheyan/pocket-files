// User preferences: theme, language, device profile and DPI.
// Everything is attribute-driven so emulators and small screens behave the
// same without user-agent sniffing. Preferences persist in localStorage.

import { useCallback, useEffect, useMemo, useState } from 'react';
import { LANGS, type Lang } from '../i18n';

export type ThemePreference = 'system' | 'light' | 'dark' | 'oled';
export type DeviceProfile = 'desktop' | 'handheld' | 'reader' | 'dap';
export type ProfilePreference = 'auto' | DeviceProfile;
export type DpiToken = 'low' | 'medium' | 'high';

export const THEMES: readonly ThemePreference[] = ['system', 'light', 'dark', 'oled'];
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

// Viewport buckets. Thresholds are intentional and emulator-friendly:
// handheld <= 480px wide, dap = short landscape screens (<= 520px tall),
// reader = medium portrait panels (e-ink tablets), desktop otherwise.
export function detectProfile(width: number, height: number): DeviceProfile {
  if (width <= 480) return 'handheld';
  if (height <= 520) return 'dap';
  if (width <= 900) return 'reader';
  return 'desktop';
}

export function detectDpi(): DpiToken {
  const ratio = window.devicePixelRatio || 1;
  if (ratio >= 1.75) return 'high';
  if (ratio >= 1.25) return 'medium';
  return 'low';
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
  const [dpi, setDpi] = useState<DpiToken>(detectDpi);

  useEffect(() => {
    const onResize = () => {
      setViewport({ width: window.innerWidth, height: window.innerHeight });
      setDpi(detectDpi());
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

  return { theme, setTheme, lang, setLang, profile, setProfile, resolvedProfile, dpi };
}
