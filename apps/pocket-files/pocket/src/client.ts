// Files client: storage volumes and directory entries over filesd.
// Destructive actions are confirmed by the daemon; this port shows the report.
import { createPortClient } from "./client-kit";
import type { AppConfig, Metric, Section, SettingRow } from "./model";

const sections: Section[] = [
  {
    id: "volumes", title: "VOLUMES", jp: "ストレージ", items: [
      { id: "internal", title: "Internal", subtitle: "58.2 GB free of 128 GB", meta: "MOCK", badge: "INT", progress: 0.55, body: "Provider-backed internal volume; paths are sandboxed.", lines: [{ label: "used", value: "69.8 GB" }, { label: "readonly", value: "no" }, { label: "source", value: "mock" }] },
      { id: "tf-a", title: "TF-A", subtitle: "119 GB free of 128 GB", meta: "MOCK", badge: "TF", progress: 0.07, body: "Removable microSD volume." },
      { id: "usb", title: "USB", subtitle: "27 GB free of 32 GB", meta: "MOCK", badge: "USB", progress: 0.16, body: "USB mass storage volume." },
    ],
  },
  {
    id: "files", title: "RECENT FILES", jp: "最近", items: [
      { id: "f-doc", title: "Documents", subtitle: "14 items · 320 MB", meta: "internal", badge: "DIR", body: "Text, PDF and markdown documents." },
      { id: "f-books", title: "Books", subtitle: "38 items · 1.8 GB", meta: "internal", badge: "DIR", body: "EPUB and PDF library." },
      { id: "f-videos", title: "Videos", subtitle: "6 items · 12.4 GB", meta: "internal", badge: "DIR", body: "Offline video archive." },
    ],
  },
];

const metrics: Metric[] = [
  { label: "VOLUMES", value: "4" },
  { label: "JOBS", value: "1" },
  { label: "MODE", value: "mock" },
];

const settings: SettingRow[] = [
  { id: "sort", label: "Sort", jp: "並び順", value: "NAME", options: ["NAME", "SIZE", "MODIFIED"] },
  { id: "view", label: "View", jp: "表示", value: "LIST", options: ["LIST", "GRID"] },
  { id: "trash", label: "Delete to trash", jp: "ゴミ箱", value: "ON", options: ["ON", "OFF"] },
];

export const APP: AppConfig = {
  title: "FILES",
  jp: "ファイル",
  kind: "detail",
  page: "FILES",
  tabs: ["ファイル", "転送", "設定"],
  actions: ["OPEN", "COPY", "DELETE"],
  settingsTitle: "BROWSER",
  settingsJp: "表示設定",
  deviceGap: "filesd bridge missing.",
  sourceNote: "/api/storage · /api/files",
};

export const client = createPortClient({
  resource: "files",
  fixture: { sections, metrics, settings },
  endpoints: {
    list: "/api/storage",
    get: (id) => `/api/files?storage_id=${encodeURIComponent(id)}`,
    action: (name) => `/api/operations/${encodeURIComponent(name.toLowerCase())}`,
  },
  act(name, payload, bundle) {
    const item = bundle.sections.flatMap((section) => section.items).find((entry) => entry.id === payload?.id);
    if (!item) return { ok: false, message: "no entry selected" };
    if (name === "OPEN") { item.state = "open"; return { ok: true, message: `${item.title} opened` }; }
    if (name === "COPY") { item.badge = "CP"; return { ok: true, message: `${item.title} queued as transfer` }; }
    if (name === "DELETE") {
      if (item.id === "internal" || item.id === "tf-a" || item.id === "usb") return { ok: false, message: "volume delete refused" };
      item.badge = "TRASH";
      return { ok: true, message: `${item.title} moved to .Trash` };
    }
    return { ok: false, message: `${name} unavailable` };
  },
});
