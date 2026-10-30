// Display helpers shared by every page. Pure functions only.

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), UNITS.length - 1);
  const value = bytes / 1024 ** exponent;
  const digits = exponent === 0 || value >= 100 ? 0 : 1;
  return `${value.toFixed(digits)} ${UNITS[exponent]}`;
}

export function formatDate(iso: string, lang: string): string {
  if (!iso) return '—';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleDateString(lang === 'zh' ? 'zh-CN' : lang === 'ja' ? 'ja-JP' : 'en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

export function percent(used: number, total: number): number {
  if (!Number.isFinite(total) || total <= 0) return 0;
  return Math.max(0, Math.min(100, (used / total) * 100));
}

export function joinPath(base: string, name: string): string {
  if (!base) return name;
  return `${base.replace(/\/+$/, '')}/${name}`;
}

export function parentPath(path: string): string {
  const clean = path.replace(/\/+$/, '');
  if (!clean) return '';
  const idx = clean.lastIndexOf('/');
  return idx <= 0 ? '' : clean.slice(0, idx);
}

export function parseSizeToBytes(value: string): number | null {
  const trimmed = value.trim();
  if (!trimmed) return null;
  const match = /^([\d.]+)\s*(b|kb|mb|gb|tb)?$/i.exec(trimmed);
  if (!match) return null;
  const amount = Number.parseFloat(match[1]);
  if (!Number.isFinite(amount)) return null;
  const unit = (match[2] ?? 'b').toLowerCase();
  const exponent = UNITS.findIndex((u) => u.toLowerCase() === unit);
  const power = exponent < 0 ? 0 : exponent;
  return Math.round(amount * 1024 ** power);
}
