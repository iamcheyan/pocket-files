package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Backend struct {
	p  storageBackend
	db *sql.DB
	tc *transferController
	sc *scanCache
}

func newBackend(p storageBackend, db *sql.DB) *Backend {
	return &Backend{p: p, db: db, tc: newTransferController(p, db), sc: newScanCache()}
}

func (b *Backend) getDevice(id string) (Device, bool) {
	for _, d := range b.p.allDevices() {
		if d.ID == id {
			return d, true
		}
	}
	return Device{}, false
}

func (b *Backend) getFriendlyName(id string) string {
	var name string
	e := b.db.QueryRow(`SELECT friendly_name FROM storage_devices WHERE id=?`, id).Scan(&name)
	if e == nil && name != "" {
		return name
	}
	if d, ok := b.getDevice(id); ok {
		return d.Name
	}
	return id
}

func (b *Backend) updateFriendlyName(id, name string) error {
	if _, ok := b.getDevice(id); !ok {
		return fmt.Errorf("unknown storage")
	}
	if name == "" {
		return fmt.Errorf("friendly_name is required")
	}
	_, e := b.db.Exec(`INSERT INTO storage_devices(id,friendly_name,kind,status,mock) VALUES(?,?,?,?,1) ON CONFLICT(id) DO UPDATE SET friendly_name=excluded.friendly_name`, id, name, "", 0)
	return e
}

func (b *Backend) storageUsage(id string) (total, used, free int64, err error) {
	d, ok := b.getDevice(id)
	if !ok {
		return 0, 0, 0, fmt.Errorf("unknown storage")
	}
	switch d.Status {
	case "not_implemented":
		return 0, 0, 0, ErrNotImplemented
	}
	if d.Mock {
		// Mock volumes declare their own nominal capacity; reporting statfs here
		// would attribute the development host's disk size to a fake device.
		sum := directoryUsage(d.Root)
		return d.Total, sum, d.Total - sum, nil
	}
	if d.Root == "" {
		return 0, 0, 0, fmt.Errorf("storage has no accessible root")
	}
	var st syscall.Statfs_t
	if e := syscall.Statfs(d.Root, &st); e != nil {
		return 0, 0, 0, fmt.Errorf("statfs unavailable")
	}
	bsize := int64(st.Bsize)
	total = bsize * int64(st.Blocks)
	free = bsize * int64(st.Bfree)
	used = total - free
	return total, used, free, nil
}

// ---- favorites ----
func (b *Backend) listFavorites(storageID string) []map[string]any {
	q := `SELECT storage_id,uri FROM favorites`
	args := []any{}
	if storageID != "" {
		q += ` WHERE storage_id=?`
		args = append(args, storageID)
	}
	q += ` ORDER BY id`
	rows, e := b.db.Query(q, args...)
	if e != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var sid, uri string
		if e := rows.Scan(&sid, &uri); e != nil {
			continue
		}
		out = append(out, map[string]any{"storage_id": sid, "uri": uri})
	}
	return out
}

func (b *Backend) addFavorite(storageID, uri string) error {
	if storageID == "" || uri == "" {
		return fmt.Errorf("storage_id and uri are required")
	}
	_, e := b.db.Exec(`INSERT INTO favorites(storage_id,uri) VALUES(?,?) ON CONFLICT(storage_id,uri) DO NOTHING`, storageID, uri)
	return e
}

func (b *Backend) removeFavorite(storageID, uri string) error {
	if storageID == "" || uri == "" {
		return fmt.Errorf("storage_id and uri are required")
	}
	_, e := b.db.Exec(`DELETE FROM favorites WHERE storage_id=? AND uri=?`, storageID, uri)
	return e
}

// ---- tags ----
func (b *Backend) listTags(storageID, uri string) []map[string]any {
	q := `SELECT storage_id,uri,tag FROM file_tags`
	cl := []string{}
	args := []any{}
	if storageID != "" {
		cl = append(cl, "storage_id=?")
		args = append(args, storageID)
	}
	if uri != "" {
		cl = append(cl, "uri=?")
		args = append(args, uri)
	}
	if len(cl) > 0 {
		q += ` WHERE ` + strings.Join(cl, " AND ")
	}
	q += ` ORDER BY id`
	rows, e := b.db.Query(q, args...)
	if e != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var sid, uri2, tag string
		if e := rows.Scan(&sid, &uri2, &tag); e != nil {
			continue
		}
		out = append(out, map[string]any{"storage_id": sid, "uri": uri2, "tag": tag})
	}
	return out
}

func (b *Backend) upsertTag(storageID, uri, tag string) error {
	if storageID == "" || uri == "" || tag == "" {
		return fmt.Errorf("storage_id, uri and tag are required")
	}
	_, e := b.db.Exec(`INSERT INTO file_tags(storage_id,uri,tag) VALUES(?,?,?)`, storageID, uri, tag)
	return e
}

func (b *Backend) deleteTag(storageID, uri, tag string) error {
	if tag == "" {
		_, e := b.db.Exec(`DELETE FROM file_tags WHERE storage_id=? AND uri=?`, storageID, uri)
		return e
	}
	_, e := b.db.Exec(`DELETE FROM file_tags WHERE storage_id=? AND uri=? AND tag=?`, storageID, uri, tag)
	return e
}

// ---- recents ----
func (b *Backend) listRecents(kind string, limit int) []map[string]any {
	q := `SELECT storage_id,uri,kind,accessed_at FROM recent_files`
	args := []any{}
	if kind != "" {
		q += ` WHERE kind=?`
		args = append(args, kind)
	}
	q += ` ORDER BY accessed_at DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, e := b.db.Query(q, args...)
	if e != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var sid, uri, kind, at string
		if e := rows.Scan(&sid, &uri, &kind, &at); e != nil {
			continue
		}
		out = append(out, map[string]any{"storage_id": sid, "uri": uri, "kind": kind, "accessed_at": at})
	}
	return out
}

func (b *Backend) recordRecent(storageID, uri, kind string) error {
	if storageID == "" || uri == "" {
		return fmt.Errorf("storage_id and uri are required")
	}
	if kind == "" {
		kind = "file"
	}
	at := time.Now().UTC().Format(time.RFC3339)
	_, e := b.db.Exec(`INSERT INTO recent_files(storage_id,uri,kind,accessed_at) VALUES(?,?,?,?)`, storageID, uri, kind, at)
	return e
}

func (b *Backend) clearRecents() error {
	_, e := b.db.Exec(`DELETE FROM recent_files`)
	return e
}

// ---- settings ----
func (b *Backend) getSettings() map[string]any {
	out := map[string]any{"mode": "mock", "privacy_mode": false}
	rows, e := b.db.Query(`SELECT key,value FROM settings`)
	if e != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if e := rows.Scan(&k, &v); e != nil {
			continue
		}
		var val any
		if e := json.Unmarshal([]byte(v), &val); e != nil {
			val = v
		}
		out[k] = val
	}
	return out
}

func (b *Backend) putSetting(key, value string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	_, e := b.db.Exec(`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return e
}

// ---- entry helper ----
func (b *Backend) getEntry(storageID, rel string) (Entry, error) {
	return b.p.Stat(context.Background(), storageID, rel)
}

func (b *Backend) recordRecentForEntry(e Entry) {
	_ = b.recordRecent(e.StorageID, e.ID, e.Kind)
}

// ---- file filters (search extensions) ----
func (b *Backend) applyFileFilters(items []Entry, q url.Values) []Entry {
	if ext := q.Get("ext"); ext != "" {
		ext = strings.TrimPrefix(strings.ToLower(ext), ".")
		filtered := items[:0]
		for _, x := range items {
			if strings.EqualFold(x.Extension, ext) {
				filtered = append(filtered, x)
			}
		}
		items = filtered
	}
	if typ := q.Get("type"); typ != "" {
		filtered := items[:0]
		for _, x := range items {
			if x.Kind == typ {
				filtered = append(filtered, x)
			}
		}
		items = filtered
	}
	if ms := parseSize(q.Get("min_size")); ms > 0 {
		filtered := items[:0]
		for _, x := range items {
			if x.Size >= ms {
				filtered = append(filtered, x)
			}
		}
		items = filtered
	}
	if xs := parseSize(q.Get("max_size")); xs > 0 {
		filtered := items[:0]
		for _, x := range items {
			if x.Size <= xs {
				filtered = append(filtered, x)
			}
		}
		items = filtered
	}
	if since := parseDate(q.Get("modified_since")); !since.IsZero() {
		filtered := items[:0]
		for _, x := range items {
			if !x.Modified.Before(since) {
				filtered = append(filtered, x)
			}
		}
		items = filtered
	}
	if until := parseDateUpper(q.Get("modified_before")); !until.IsZero() {
		filtered := items[:0]
		for _, x := range items {
			if !x.Modified.After(until) {
				filtered = append(filtered, x)
			}
		}
		items = filtered
	}
	if sortBy := q.Get("sort"); sortBy != "" {
		items = stableSortEntries(items, sortBy, q.Get("order"))
	}
	return items
}

// parseDateUpper is parseDate for upper bounds. A date-only value expands to the
// end of that day so "modified_before=2026-10-30" includes files changed then.
func parseDateUpper(value string) time.Time {
	parsed := parseDate(value)
	if parsed.IsZero() {
		return parsed
	}
	if len(strings.TrimSpace(value)) == len("2006-01-02") {
		return parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return parsed
}

// parseDate accepts RFC3339 timestamps and plain YYYY-MM-DD dates.
func parseDate(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func parseSize(s string) int64 {
	if s == "" {
		return 0
	}
	s = strings.TrimSpace(s)
	mult := float64(1)
	lower := strings.ToLower(s)
	switch {
	case strings.HasSuffix(lower, "kb"):
		mult = float64(1 << 10)
		lower = lower[:len(lower)-2]
	case strings.HasSuffix(lower, "mb"):
		mult = float64(1 << 20)
		lower = lower[:len(lower)-2]
	case strings.HasSuffix(lower, "gb"):
		mult = float64(1 << 30)
		lower = lower[:len(lower)-2]
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(lower), 64)
	if err != nil {
		return 0
	}
	return int64(v * mult)
}

func stableSortEntries(items []Entry, by, order string) []Entry {
	desc := strings.EqualFold(order, "desc")
	stable := make([]Entry, len(items))
	copy(stable, items)
	sort.SliceStable(stable, func(i, j int) bool {
		cmp := 0
		switch by {
		case "size":
			cmp = compareInt64(stable[i].Size, stable[j].Size)
		case "modified":
			cmp = compareTime(stable[i].Modified, stable[j].Modified)
		case "created":
			cmp = compareTime(stable[i].Created, stable[j].Created)
		case "type":
			cmp = strings.Compare(stable[i].Extension, stable[j].Extension)
		default:
			cmp = strings.Compare(stable[i].Name, stable[j].Name)
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
	return stable
}

func compareInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func compareTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	}
	return 0
}

// ---- views ----
func (b *Backend) getView(ctx context.Context, name string) ([]map[string]any, error) {
	var out []map[string]any
	var err error
	switch name {
	case "books":
		out = b.viewByExts(ctx, bookExts)
	case "music":
		out = b.viewByExts(ctx, musicExts)
	case "videos":
		out = b.viewByExts(ctx, videoExts)
	case "games":
		out = b.viewByExts(ctx, gameExts)
	case "downloads":
		out = b.viewPath(ctx, "Downloads")
	case "recent":
		out = b.viewRecents()
	case "favorites":
		out = b.viewFavorites()
	default:
		err = fmt.Errorf("unknown view %q", name)
	}
	return out, err
}

func (b *Backend) viewByExts(ctx context.Context, exts map[string]bool) []map[string]any {
	out := []map[string]any{}
	var walk func(id, rel string)
	walk = func(id, rel string) {
		items, e := b.p.Browse(ctx, id, rel)
		if e != nil {
			return
		}
		for _, en := range items {
			if en.Kind == "directory" {
				walk(id, en.ID)
				continue
			}
			if exts[strings.ToLower(en.Extension)] {
				out = append(out, entryItem(en))
			}
		}
	}
	for _, d := range b.p.allDevices() {
		walk(d.ID, "")
	}
	return out
}
func (b *Backend) viewPath(ctx context.Context, rel string) []map[string]any {
	out := []map[string]any{}
	for _, d := range b.p.allDevices() {
		items, e := b.p.Browse(ctx, d.ID, rel)
		if e != nil {
			continue
		}
		for _, en := range items {
			out = append(out, entryItem(en))
		}
	}
	return out
}

func (b *Backend) viewRecents() []map[string]any {
	recs := b.listRecents("", 0)
	out := make([]map[string]any, 0, len(recs))
	for _, r := range recs {
		sid, uri := r["storage_id"].(string), r["uri"].(string)
		e, err := b.getEntry(sid, uri)
		if err != nil {
			continue
		}
		out = append(out, entryItem(e))
	}
	return out
}

func (b *Backend) viewFavorites() []map[string]any {
	favs := b.listFavorites("")
	out := make([]map[string]any, 0, len(favs))
	for _, f := range favs {
		sid, uri := f["storage_id"].(string), f["uri"].(string)
		e, err := b.getEntry(sid, uri)
		if err != nil {
			continue
		}
		out = append(out, entryItem(e))
	}
	return out
}

func entryItem(e Entry) map[string]any {
	return map[string]any{
		"id": e.ID, "storage_id": e.StorageID, "name": e.Name, "parent": e.Parent, "kind": e.Kind,
		"size": e.Size, "modified": e.Modified, "created": e.Created, "extension": e.Extension,
	}
}

var (
	bookExts  = map[string]bool{"epub": true, "pdf": true, "mobi": true, "azw": true, "azw3": true, "cbz": true, "cbr": true, "txt": true}
	musicExts = map[string]bool{"mp3": true, "flac": true, "wav": true, "ogg": true, "m4a": true, "aac": true, "opus": true}
	videoExts = map[string]bool{"mp4": true, "mkv": true, "webm": true, "avi": true, "mov": true, "m4v": true}
	gameExts  = map[string]bool{"elf": true, "iso": true, "rom": true, "gba": true, "nes": true, "sfc": true, "nro": true}
)

// ---- analytics ----
type analyticsResult struct {
	StorageID          string           `json:"storage_id"`
	Mock               bool             `json:"mock"`
	Total              int64            `json:"total"`
	Used               int64            `json:"used"`
	Free               int64            `json:"free"`
	CategoryCounts     map[string]int   `json:"category_counts"`
	LargestFiles       []map[string]any `json:"largest_files"`
	LargestDirectories []map[string]any `json:"largest_directories"`
	CacheTimestamp     string           `json:"cache_timestamp"`
	Backgrounded       bool             `json:"backgrounded"`
	Progress           float64          `json:"progress"`
}

func (b *Backend) computeAnalytics(storageID string) analyticsResult {
	d, ok := b.getDevice(storageID)
	res := analyticsResult{StorageID: storageID, Mock: d.Mock, CategoryCounts: map[string]int{}}
	if ok {
		if total, used, free, err := b.storageUsage(storageID); err == nil {
			res.Total, res.Used, res.Free = total, used, free
		}
	}
	res.LargestFiles = []map[string]any{}
	res.LargestDirectories = []map[string]any{}
	files, cats, dirs := b.scanTree(storageID)
	res.CategoryCounts = cats
	sort.Slice(files, func(i, j int) bool { return files[i].Size > files[j].Size })
	for i, en := range files {
		if i >= 20 {
			break
		}
		res.LargestFiles = append(res.LargestFiles, entryItem(en))
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].size > dirs[j].size })
	for i, d2 := range dirs {
		if i >= 20 {
			break
		}
		res.LargestDirectories = append(res.LargestDirectories, map[string]any{"storage_id": d2.storage, "uri": d2.uri, "size": d2.size})
	}
	res.CacheTimestamp = time.Now().UTC().Format(time.RFC3339)
	return res
}

type dirSize struct {
	storage, uri string
	size         int64
}

func (b *Backend) scanTree(storageID string) ([]Entry, map[string]int, []dirSize) {
	cats := map[string]int{}
	dirSizes := map[string]int64{}
	files := map[string]Entry{}
	var walk func(id, rel string) int64
	walk = func(id, rel string) int64 {
		items, err := b.p.Browse(context.Background(), id, rel)
		if err != nil {
			return 0
		}
		var total int64
		for _, item := range items {
			if item.Kind == "directory" {
				total += walk(id, item.ID)
				continue
			}
			files[id+":"+item.ID] = item
			total += item.Size
			cats[categoryOf(item.Extension)]++
		}
		if rel != "" {
			dirSizes[id+"\x00"+rel] += total
		}
		return total
	}
	if storageID == "" {
		for _, d := range b.p.allDevices() {
			walk(d.ID, "")
		}
	} else {
		walk(storageID, "")
	}
	fileList := make([]Entry, 0, len(files))
	for _, entry := range files {
		fileList = append(fileList, entry)
	}
	dirs := make([]dirSize, 0, len(dirSizes))
	for key, size := range dirSizes {
		if size == 0 {
			continue
		}
		parts := strings.SplitN(key, "\x00", 2)
		dirs = append(dirs, dirSize{storage: parts[0], uri: parts[1], size: size})
	}
	return fileList, cats, dirs
}

// categoryOf classifies a file for analytics and library views. Book formats are
// checked before Documents so reader content is not hidden under Documents.
func categoryOf(ext string) string {
	ext = strings.ToLower(ext)
	switch {
	case ext == "pdf" || ext == "epub" || ext == "mobi" || ext == "azw" || ext == "azw3" || ext == "cbz" || ext == "cbr":
		return "Books"
	case musicExts[ext]:
		return "Music"
	case videoExts[ext]:
		return "Videos"
	case gameExts[ext]:
		return "Games"
	case ext == "md" || ext == "doc" || ext == "docx" || ext == "odt" || ext == "txt":
		return "Documents"
	}
	return "Other"
}

func (b *Backend) getAnalytics(storageID string) analyticsResult {
	if res, ok := b.sc.get(storageID); ok {
		return res
	}
	res := b.computeAnalytics(storageID)
	b.sc.set(storageID, res)
	return res
}

// ---- operations helpers ----
func (b *Backend) trashEntry(id, rel string) (string, error) {
	return b.p.Trash(context.Background(), id, rel)
}

func (b *Backend) restoreEntry(id, src, trashRel string) error {
	return b.p.Restore(context.Background(), id, src, trashRel)
}

func (b *Backend) copyFile(id, src, dst string) error {
	return b.p.Copy(context.Background(), id, src, dst)
}
func (b *Backend) moveFile(id, src, dst string) error {
	return b.p.Move(context.Background(), id, src, dst)
}
func (b *Backend) deleteFile(id, rel string) error { return b.p.Delete(context.Background(), id, rel) }

func (b *Backend) splitID(id string) (storageID, rel string, err error) {
	sp := strings.SplitN(id, ":", 2)
	if len(sp) != 2 || sp[0] == "" || sp[1] == "" {
		return "", "", fmt.Errorf("invalid id %q (expected storage:rel)", id)
	}
	return sp[0], sp[1], nil
}

func resolveRelativeParent(rel string) (string, string) {
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." || dir == "" {
		dir = ""
	}
	return dir, filepath.Base(rel)
}
func (b *Backend) recoverTransfers() { b.tc.recover() }

// providerFor resolves a provider through the router when one is configured.
func (b *Backend) providerFor(id string) (Provider, error) {
	if router, ok := b.p.(*providerRouter); ok {
		return router.providerFor(id)
	}
	if mock, ok := b.p.(*MockProvider); ok {
		return mock, nil
	}
	return nil, ErrNotImplemented
}
