// Settings: theme, language, device profile. All choices persist through
// usePreferences (localStorage) and drive data-* attributes on <html>.

import type { TFunction } from '../i18n';
import { langNames, LANGS, type Lang } from '../i18n';
import {
  DPI_PREFERENCES,
  PROFILES,
  THEMES,
  type DpiPreference,
  type ProfilePreference,
  type ThemePreference,
} from '../hooks/usePreferences';

const THEME_LABEL: Record<ThemePreference, 'settingsThemeSystem' | 'settingsThemeLight' | 'settingsThemeDark' | 'settingsThemeOled'> = {
  system: 'settingsThemeSystem',
  light: 'settingsThemeLight',
  dark: 'settingsThemeDark',
  oled: 'settingsThemeOled',
};

const PROFILE_LABEL: Record<
  ProfilePreference,
  | 'settingsProfileAuto'
  | 'settingsProfileDesktop'
  | 'settingsProfileHandheld'
  | 'settingsProfileReader'
  | 'settingsProfileDap'
> = {
  auto: 'settingsProfileAuto',
  desktop: 'settingsProfileDesktop',
  handheld: 'settingsProfileHandheld',
  reader: 'settingsProfileReader',
  dap: 'settingsProfileDap',
};

interface SettingsPageProps {
  t: TFunction;
  theme: ThemePreference;
  onThemeChange: (theme: ThemePreference) => void;
  lang: Lang;
  onLangChange: (lang: Lang) => void;
  profile: ProfilePreference;
  onProfileChange: (profile: ProfilePreference) => void;
  resolvedProfile: string;
  dpi: string;
  dpiPreference: DpiPreference;
  onDpiChange: (value: DpiPreference) => void;
}

export function SettingsPage({
  t,
  theme,
  onThemeChange,
  lang,
  onLangChange,
  profile,
  onProfileChange,
  resolvedProfile,
  dpi,
  dpiPreference,
  onDpiChange,
}: SettingsPageProps) {
  return (
    <>
      <div className="welcome">
        <div>
          <div className="eyebrow">{t('settingsPreferences')}</div>
          <h1>{t('settingsTitle')}</h1>
          <p>{t('settingsSubtitle')}</p>
        </div>
      </div>
      <div className="settings-card">
        <h2>{t('settingsAppearance')}</h2>
        <p>{t('settingsTheme')}</p>
        <div className="choices">
          {THEMES.map((value) => (
            <button
              type="button"
              key={value}
              className={theme === value ? 'choice selected' : 'choice'}
              onClick={() => onThemeChange(value)}
              aria-pressed={theme === value}
              data-nav
            >
              {t(THEME_LABEL[value])}
            </button>
          ))}
        </div>

        <h2>{t('settingsLanguage')}</h2>
        <div className="choices">
          {LANGS.map((code) => (
            <button
              type="button"
              key={code}
              className={lang === code ? 'choice selected' : 'choice'}
              onClick={() => onLangChange(code)}
              aria-pressed={lang === code}
              data-nav
            >
              {langNames[code]}
            </button>
          ))}
        </div>

        <h2>{t('settingsProfile')}</h2>
        <div className="choices">
          {PROFILES.map((value) => (
            <button
              type="button"
              key={value}
              className={profile === value ? 'choice selected' : 'choice'}
              onClick={() => onProfileChange(value)}
              aria-pressed={profile === value}
              data-nav
            >
              {t(PROFILE_LABEL[value])}
            </button>
          ))}
        </div>
        <h2>{t('settingsDpi')}</h2>
        <p>{t('settingsDpiHint')}</p>
        <div className="choices">
          {DPI_PREFERENCES.map((value) => (
            <button
              type="button"
              key={value}
              className={dpiPreference === value ? 'choice selected' : 'choice'}
              onClick={() => onDpiChange(value)}
              aria-pressed={dpiPreference === value}
              data-nav
            >
              {value === 'auto' ? t('settingsDpiAuto') : value}
            </button>
          ))}
        </div>
        <p className="muted">
          {t('settingsProfileDetail', {
            profile: resolvedProfile,
            label: t('settingsProfile'),
            dpi,
          })}
        </p>
      </div>
    </>
  );
}
