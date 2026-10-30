// Mouse-free navigation for keyboard, D-pad and gamepads.
//
// Arrow keys move DOM focus spatially (nearest control in the pressed
// direction) across the whole page, so list rows, toolbar buttons and sidebar
// links all participate without per-component wiring. Enter/Space activate the
// focused control, Escape invokes the page's "back" action. A requestAnimation-
// Frame loop polls the Gamepad API and maps D-pad/left-stick to arrows, A to
// activate and B to back. Missing gamepads are ignored silently.

import { useEffect } from 'react';

type Direction = 'up' | 'down' | 'left' | 'right';

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
  '[role="button"]',
].join(',');

const TEXT_ENTRY_TAGS = ['INPUT', 'TEXTAREA', 'SELECT'];

function isTextEntry(el: Element | null): boolean {
  if (!el) return false;
  if (el instanceof HTMLElement && el.isContentEditable) return true;
  return TEXT_ENTRY_TAGS.includes(el.tagName);
}

function focusableIn(root: ParentNode): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter((el) => {
    if (el.getAttribute('aria-hidden') === 'true') return false;
    if (el.tabIndex < 0 && !el.hasAttribute('data-nav')) return false;
    const rects = el.getClientRects();
    return rects.length > 0 && el.offsetParent !== null;
  });
}

function centerOf(el: HTMLElement): { x: number; y: number } {
  const rect = el.getBoundingClientRect();
  return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
}

// Spatial navigation: pick the closest focusable whose center lies in the
// pressed direction, penalising drift along the perpendicular axis.
function nextInDirection(current: HTMLElement, direction: Direction): HTMLElement | null {
  const from = centerOf(current);
  const candidates = focusableIn(document).filter((el) => el !== current);

  let best: HTMLElement | null = null;
  let bestScore = Number.POSITIVE_INFINITY;

  for (const el of candidates) {
    const to = centerOf(el);
    const dx = to.x - from.x;
    const dy = to.y - from.y;

    let primary: number;
    let perpendicular: number;
    switch (direction) {
      case 'up':
        if (dy >= -4) continue;
        primary = -dy;
        perpendicular = Math.abs(dx);
        break;
      case 'down':
        if (dy <= 4) continue;
        primary = dy;
        perpendicular = Math.abs(dx);
        break;
      case 'left':
        if (dx >= -4) continue;
        primary = -dx;
        perpendicular = Math.abs(dy);
        break;
      case 'right':
        if (dx <= 4) continue;
        primary = dx;
        perpendicular = Math.abs(dy);
        break;
    }

    const score = primary + perpendicular * 2;
    if (score < bestScore) {
      bestScore = score;
      best = el;
    }
  }
  return best;
}

function moveFocus(direction: Direction): boolean {
  const active = document.activeElement;
  if (!(active instanceof HTMLElement) || active === document.body) {
    const first = focusableIn(document.querySelector('[data-nav-scope="main"]') ?? document)[0];
    if (first) {
      first.focus();
      return true;
    }
    return false;
  }
  const target = nextInDirection(active, direction);
  if (!target) return false;
  target.focus();
  return true;
}

function activateFocused(): boolean {
  const active = document.activeElement;
  if (!(active instanceof HTMLElement)) return false;
  // Native controls already handle Enter/Space (and Space toggles checkboxes),
  // so clicking them here would double-fire.
  if (active.tagName === 'BUTTON' || active.tagName === 'A' || active.tagName === 'INPUT') {
    return false;
  }
  active.click();
  return true;
}

export interface KeyboardNavOptions {
  /** Escape / gamepad B. Called only when no text field holds focus. */
  onBack?: () => void;
  /** Disable the whole manager (e.g. while a blocking dialog owns keys). */
  enabled?: boolean;
}

export function useKeyboardNav({ onBack, enabled = true }: KeyboardNavOptions): void {
  useEffect(() => {
    if (!enabled) return;

    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target instanceof Element ? event.target : null;

      // A modal owns its own keys; ConfirmDialog subscribes to Escape itself.
      if (document.querySelector('[role="alertdialog"]')) return;

      if (event.key === 'Escape') {
        // Let the focused text field blur first, then treat the next Escape as
        // a back action.
        if (isTextEntry(target)) {
          (target as HTMLElement).blur();
          return;
        }
        if (onBack) {
          event.preventDefault();
          onBack();
        }
        return;
      }

      // Arrow keys must keep their caret/selection meaning inside text fields.
      if (isTextEntry(target)) return;

      const arrowMap: Partial<Record<string, Direction>> = {
        ArrowUp: 'up',
        ArrowDown: 'down',
        ArrowLeft: 'left',
        ArrowRight: 'right',
      };
      const direction = arrowMap[event.key];
      if (direction) {
        if (event.metaKey || event.ctrlKey || event.altKey) return;
        if (moveFocus(direction)) event.preventDefault();
        return;
      }

      // Enter and Space activate controls. Native buttons/links already do this
      // on their own; everything else (rows, cards, custom controls) is clicked
      // explicitly so Enter/Space and gamepad A behave identically.
      if (event.key === 'Enter' || event.key === ' ' || event.key === 'Spacebar') {
        if (activateFocused()) {
          event.preventDefault();
        }
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onBack, enabled]);

  useEffect(() => {
    if (!enabled) return;
    if (typeof navigator === 'undefined' || typeof navigator.getGamepads !== 'function') return;

    let frame = 0;
    let lastMove = 0;
    let previous: Record<number, boolean[]> = {};
    let previousAxes: Record<number, { x: number; y: number }> = {};

    const pulse = (now: number) => {
      const pads = navigator.getGamepads();
      for (const pad of pads) {
        if (!pad) continue;
        const buttons = pad.buttons.map((b) => b.pressed);
        const prevButtons = previous[pad.index] ?? [];
        const pressed = (index: number) => buttons[index] === true && prevButtons[index] !== true;

        const dpad: Array<[number, Direction]> = [
          [12, 'up'],
          [13, 'down'],
          [14, 'left'],
          [15, 'right'],
        ];
        for (const [index, direction] of dpad) {
          if (pressed(index)) moveFocus(direction);
        }

        // Analog stick: 0 = left stick X, 1 = left stick Y.
        const axes = { x: pad.axes[0] ?? 0, y: pad.axes[1] ?? 0 };
        const prevAxes = previousAxes[pad.index] ?? { x: 0, y: 0 };
        if (now - lastMove > 180) {
          if (axes.y < -0.55 && prevAxes.y >= -0.55) {
            moveFocus('up');
            lastMove = now;
          } else if (axes.y > 0.55 && prevAxes.y <= 0.55) {
            moveFocus('down');
            lastMove = now;
          } else if (axes.x < -0.55 && prevAxes.x >= -0.55) {
            moveFocus('left');
            lastMove = now;
          } else if (axes.x > 0.55 && prevAxes.x <= 0.55) {
            moveFocus('right');
            lastMove = now;
          }
        }
        previousAxes[pad.index] = axes;

        if (pressed(0)) activateFocused();
        if (pressed(1) && onBack) onBack();

        previous[pad.index] = buttons;
      }
      frame = window.requestAnimationFrame(pulse);
    };

    frame = window.requestAnimationFrame(pulse);
    return () => {
      window.cancelAnimationFrame(frame);
      previous = {};
      previousAxes = {};
    };
  }, [enabled, onBack]);
}
