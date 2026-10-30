// Small presentational primitives shared across pages. No page-specific logic.

import { useEffect, useRef, type ReactNode } from 'react';
import { CircleAlert, Loader, ServerCrash } from 'lucide-react';
import type { TFunction } from '../i18n';

export function PageState({ children }: { children: ReactNode }) {
  return <div className="page-state">{children}</div>;
}

export function LoadingState({ label }: { label: string }) {
  return (
    <PageState>
      <Loader className="spin" aria-hidden />
      <strong>{label}</strong>
    </PageState>
  );
}

export function UnavailableState({
  t,
  message,
  onRetry,
}: {
  t: TFunction;
  message: string;
  onRetry?: () => void;
}) {
  return (
    <PageState>
      <ServerCrash aria-hidden />
      <strong>{t('commonUnavailable')}</strong>
      <span>{message}</span>
      {onRetry && (
        <button type="button" className="soft" onClick={onRetry} data-nav>
          {t('commonRetry')}
        </button>
      )}
    </PageState>
  );
}

export function ErrorState({
  t,
  message,
  onRetry,
}: {
  t: TFunction;
  message: string;
  onRetry?: () => void;
}) {
  return (
    <PageState>
      <CircleAlert aria-hidden />
      <strong>{t('commonError')}</strong>
      <span>{message}</span>
      {onRetry && (
        <button type="button" className="soft" onClick={onRetry} data-nav>
          {t('commonRetry')}
        </button>
      )}
    </PageState>
  );
}

export function EmptyState({
  icon,
  title,
  hint,
}: {
  icon: ReactNode;
  title: string;
  hint?: string;
}) {
  return (
    <div className="empty-panel">
      {icon}
      <strong>{title}</strong>
      {hint && <span>{hint}</span>}
    </div>
  );
}

export function MockBadge({ label }: { label: string }) {
  return <span className="mock-badge">{label}</span>;
}

export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  body: string;
  confirmLabel: string;
  cancelLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  cancelLabel,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null);

  // Move focus into the dialog and wire Escape to cancel while it is open.
  useEffect(() => {
    if (!open) return;
    cancelRef.current?.focus();
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onCancel();
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      className="dialog-backdrop"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onCancel();
      }}
    >
      <div className="dialog" role="alertdialog" aria-modal="true" aria-labelledby="dialog-title">
        <h2 id="dialog-title">{title}</h2>
        <p>{body}</p>
        <div className="dialog-actions">
          <button
            type="button"
            ref={cancelRef}
            className="soft"
            onClick={onCancel}
            data-nav
            data-autofocus
          >
            {cancelLabel}
          </button>
          <button type="button" className="danger" onClick={onConfirm} data-nav>
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
