// FilePreview renders a verified preview served by filesd. The UI never learns a
// filesystem path: it asks for /api/files/{id}/preview, and the backend validates
// the type, size cap and sandbox before streaming bytes.
//
// Images, video, audio and PDF use native elements with the API URL. Text-like
// formats (txt/md/csv/log/json/yaml) are fetched as text and rendered in a
// scrollable panel so no HTML injection is possible. EPUB and other formats have
// no in-browser renderer here: they show their metadata plus an explicit note
// instead of pretending to render.

import { useEffect, useState } from 'react';
import { ExternalLink, X } from 'lucide-react';
import { ApiError, api } from '../api';
import type { TFunction } from '../i18n';
import { formatBytes, formatDate } from '../format';
import type { FileEntry } from '../types';

interface FilePreviewProps {
  t: TFunction;
  storageId: string;
  entry: FileEntry;
  onClose: () => void;
}

const IMAGE_EXTS = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp'];
const VIDEO_EXTS = ['mp4', 'mkv', 'webm', 'avi', 'mov', 'm4v'];
const AUDIO_EXTS = ['mp3', 'flac', 'wav', 'ogg', 'm4a', 'aac', 'opus'];
const TEXT_EXTS = ['txt', 'md', 'markdown', 'csv', 'log', 'json', 'yaml', 'yml'];
const BOOK_EXTS = ['epub', 'mobi', 'azw', 'azw3', 'cbz', 'cbr'];

export function FilePreview({ t, storageId, entry, onClose }: FilePreviewProps) {
  const extension = (entry.extension || '').toLowerCase();
  const previewUrl = `/api/files/${encodeURIComponent(`${storageId}:${entry.id}`)}/preview`;
  const kind = IMAGE_EXTS.includes(extension)
    ? 'image'
    : VIDEO_EXTS.includes(extension)
      ? 'video'
      : AUDIO_EXTS.includes(extension)
        ? 'audio'
        : extension === 'pdf'
          ? 'pdf'
          : TEXT_EXTS.includes(extension)
            ? 'text'
            : BOOK_EXTS.includes(extension)
              ? 'book'
              : 'none';

  const [text, setText] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [launch, setLaunch] = useState<{ state: 'idle' | 'pending' | 'unavailable' | 'opened'; message?: string }>({
    state: 'idle',
  });

  useEffect(() => {
    if (kind !== 'text') return;
    let cancelled = false;
    fetch(previewUrl)
      .then(async (response) => {
        if (!response.ok) throw new Error(`preview ${response.status}`);
        return response.text();
      })
      .then((body) => {
        if (!cancelled) setText(body);
      })
      .catch(() => {
        if (!cancelled) setError(t('previewUnavailable'));
      });
    return () => {
      cancelled = true;
    };
  }, [kind, previewUrl, t]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  return (
    <div className="dialog-backdrop" onClick={onClose} role="presentation">
      <div
        className="dialog preview-dialog"
        role="dialog"
        aria-modal="true"
        aria-label={t('filesPreview')}
        onClick={(event) => event.stopPropagation()}
      >
        <header className="preview-head">
          <div>
            <strong>{entry.name}</strong>
            <small>
              {formatBytes(entry.size)} · {formatDate(entry.modified, 'en')}
              {entry.extension ? ` · ${entry.extension}` : ''}
            </small>
          </div>
          <button type="button" className="icon-btn" onClick={onClose} aria-label={t('commonClose')} data-nav>
            <X aria-hidden />
          </button>
        </header>

        <div className="preview-actions">
          <button
            type="button"
            className="soft"
            disabled={launch.state === 'pending'}
            onClick={async () => {
              setLaunch({ state: 'pending' });
              try {
                const result = await api.openInApp(storageId, entry.id);
                setLaunch(
                  result.status === 'ok'
                    ? { state: 'opened', message: result.app }
                    : { state: 'unavailable', message: result.message ?? t('previewAppUnavailable') },
                );
              } catch (err) {
                setLaunch({
                  state: 'unavailable',
                  message: err instanceof ApiError ? err.message : t('previewAppUnavailable'),
                });
              }
            }}
            data-nav
          >
            <ExternalLink aria-hidden /> {t('previewOpenInApp')}
          </button>
          {launch.state === 'unavailable' && (
            <span className="preview-unavailable">{t('previewAppUnavailable')}</span>
          )}
          {launch.state === 'opened' && <span className="preview-opened">{launch.message}</span>}
        </div>

        <div className="preview-body">
          {kind === 'image' && <img src={previewUrl} alt={entry.name} />}
          {kind === 'video' && <video src={previewUrl} controls />}
          {kind === 'audio' && <audio src={previewUrl} controls />}
          {kind === 'pdf' && <iframe src={previewUrl} title={entry.name} />}
          {kind === 'text' && !error && text !== null && <pre className="preview-text">{text}</pre>}
          {kind === 'text' && !error && text === null && <p className="muted">{t('commonLoading')}</p>}
          {error && <p className="muted">{error}</p>}
          {kind === 'book' && (
            <div className="preview-note">
              <strong>{t('previewBookTitle')}</strong>
              <p>{t('previewBookBody')}</p>
            </div>
          )}
          {kind === 'none' && <p className="muted">{t('previewUnsupported')}</p>}
        </div>
      </div>
    </div>
  );
}
