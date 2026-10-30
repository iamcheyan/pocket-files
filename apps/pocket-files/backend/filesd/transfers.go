package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	transferQueued    = "queued"
	transferRunning   = "running"
	transferPaused    = "paused"
	transferFailed    = "failed"
	transferCompleted = "completed"
	transferCancelled = "cancelled"
	transferConflict  = "conflict"

	conflictSkip     = "skip"
	conflictReplace  = "replace"
	conflictKeepBoth = "keep_both"
	conflictAsk      = "ask"
)

type transferItem struct {
	Src string
	Dst string
}

type transferJob struct {
	ID         string
	Type       string
	StorageID  string
	Status     string
	Conflict   string
	Items      []transferItem
	Paused     bool
	Cancelled  bool
	Priority   int
	progress   []itemProgress
	totalBytes int64
	bytesDone  int64
	speed      int64
	eta        int64
	persistFn  func(*transferJob)
	mu         sync.Mutex
}

type itemProgress struct {
	state string
	err   string
}

type transferController struct {
	p    *MockProvider
	db   *sql.DB
	mu   sync.Mutex
	jobs map[string]*transferJob
	stop map[string]chan struct{}
	dbMu sync.Mutex
}

func newTransferController(p *MockProvider, db *sql.DB) *transferController {
	return &transferController{p: p, db: db, jobs: map[string]*transferJob{}, stop: map[string]chan struct{}{}}
}

func (tc *transferController) load(jobID string) (*transferJob, error) {
	var typ, storageID, status, conflict, extra, createdAt, updatedAt string
	e := tc.db.QueryRow(`SELECT type,storage_id,status,conflict_strategy,extra,created_at,updated_at FROM transfer_jobs WHERE id=?`, jobID).Scan(&typ, &storageID, &status, &conflict, &extra, &createdAt, &updatedAt)
	if e != nil {
		return nil, e
	}
	j := &transferJob{ID: jobID, Type: typ, StorageID: storageID, Status: status, Conflict: conflict}
	if extra != "" {
		var ex struct {
			Paused     bool  `json:"paused"`
			Priority   int   `json:"priority"`
			BytesDone  int64 `json:"bytes_done"`
			TotalBytes int64 `json:"total_bytes"`
			Speed      int64 `json:"speed"`
			Eta        int64 `json:"eta"`
		}
		if json.Unmarshal([]byte(extra), &ex) == nil {
			j.Paused = ex.Paused
			j.Priority = ex.Priority
			j.bytesDone = ex.BytesDone
			j.totalBytes = ex.TotalBytes
			j.speed = ex.Speed
			j.eta = ex.Eta
		}
	}
	rows, e := tc.db.Query(`SELECT src,dst,state,error FROM transfer_items WHERE job_id=? ORDER BY item_index`, jobID)
	if e != nil {
		return j, nil
	}
	defer rows.Close()
	for rows.Next() {
		var src, dst, state, errx string
		if e := rows.Scan(&src, &dst, &state, &errx); e != nil {
			continue
		}
		j.Items = append(j.Items, transferItem{Src: src, Dst: dst})
		j.progress = append(j.progress, itemProgress{state: state, err: errx})
	}
	return j, nil
}

func (tc *transferController) persist(j *transferJob) {
	tc.dbMu.Lock()
	defer tc.dbMu.Unlock()
	j.mu.Lock()
	status := j.Status
	conflict := j.Conflict
	paused := j.Paused
	priority := j.Priority
	bytesDone := j.bytesDone
	totalBytes := j.totalBytes
	speed := j.speed
	eta := j.eta
	items := make([]transferItem, len(j.Items))
	copy(items, j.Items)
	prog := make([]itemProgress, len(j.progress))
	copy(prog, j.progress)
	j.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	extraBytes, _ := json.Marshal(map[string]any{
		"paused": paused, "priority": priority, "bytes_done": bytesDone,
		"total_bytes": totalBytes, "speed": speed, "eta": eta,
	})
	_, e := tc.db.Exec(`INSERT INTO transfer_jobs(id,type,storage_id,status,conflict_strategy,extra,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)
  ON CONFLICT(id) DO UPDATE SET status=excluded.status,conflict_strategy=excluded.conflict_strategy,extra=excluded.extra,updated_at=excluded.updated_at`,
		j.ID, j.Type, j.StorageID, status, conflict, string(extraBytes), now, now)
	if e != nil {
		return
	}
	tc.db.Exec(`DELETE FROM transfer_items WHERE job_id=?`, j.ID)
	for i, it := range items {
		st := "queued"
		errx := ""
		if i < len(prog) {
			st = prog[i].state
			errx = prog[i].err
		}
		tc.db.Exec(`INSERT INTO transfer_items(job_id,item_index,src,dst,state,error) VALUES(?,?,?,?,?,?)`, j.ID, i, it.Src, it.Dst, st, errx)
	}
}

func (tc *transferController) recover() {
	rows, e := tc.db.Query(`SELECT id FROM transfer_jobs WHERE status=?`, transferRunning)
	if e != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		j, e := tc.load(id)
		if e != nil {
			continue
		}
		j.mu.Lock()
		for i := range j.progress {
			if j.progress[i].state == "running" {
				j.progress[i].state = "failed"
				j.progress[i].err = "interrupted"
			}
		}
		j.Status = transferFailed
		j.mu.Unlock()
		tc.persist(j)
	}
}

func (tc *transferController) get(id string) (*transferJob, error) {
	tc.mu.Lock()
	j, ok := tc.jobs[id]
	tc.mu.Unlock()
	if ok {
		return j, nil
	}
	j, e := tc.load(id)
	if e != nil {
		return nil, e
	}
	j.persistFn = tc.persist
	tc.mu.Lock()
	tc.jobs[id] = j
	tc.mu.Unlock()
	return j, nil
}

func (tc *transferController) list() []*transferJob {
	rows, e := tc.db.Query(`SELECT id FROM transfer_jobs ORDER BY created_at`)
	if e != nil {
		return nil
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	out := make([]*transferJob, 0, len(ids))
	for _, id := range ids {
		if j, e := tc.get(id); e == nil {
			out = append(out, j)
		}
	}
	return out
}

func (tc *transferController) active(id string) bool {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	_, ok := tc.stop[id]
	return ok
}

func (tc *transferController) create(id, typ, storageID, conflict string, items []transferItem, priority int) error {
	if conflict == "" {
		conflict = conflictSkip
	}
	switch conflict {
	case conflictSkip, conflictReplace, conflictKeepBoth, conflictAsk:
	default:
		return fmt.Errorf("invalid conflict_strategy %q", conflict)
	}
	switch typ {
	case "copy", "move", "delete":
	default:
		return fmt.Errorf("invalid transfer type %q", typ)
	}
	j := &transferJob{ID: id, Type: typ, StorageID: storageID, Status: transferQueued, Conflict: conflict, Items: items, Priority: priority, persistFn: tc.persist}
	if j.progress == nil {
		j.progress = make([]itemProgress, len(items))
	}
	for i := range items {
		j.progress[i].state = transferQueued
		if typ != "delete" {
			if e, err := tc.p.Stat(context.Background(), storageID, items[i].Src); err == nil {
				j.totalBytes += e.Size
			}
		}
	}
	tc.mu.Lock()
	tc.jobs[id] = j
	tc.mu.Unlock()
	tc.persist(j)
	return nil
}

func (tc *transferController) start(j *transferJob) {
	tc.mu.Lock()
	if _, ok := tc.stop[j.ID]; ok {
		tc.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	tc.stop[j.ID] = stop
	tc.mu.Unlock()
	tc.persist(j)
	go tc.run(j, stop)
}

func (tc *transferController) run(j *transferJob, stop chan struct{}) {
	defer func() {
		tc.mu.Lock()
		delete(tc.stop, j.ID)
		tc.mu.Unlock()
		tc.persist(j)
	}()
	j.mu.Lock()
	j.Status = transferRunning
	j.mu.Unlock()
	started := time.Now()
	for i := range j.Items {
		// pause / cancel gate at item boundary
		if !tc.gate(j, stop, started) {
			j.mu.Lock()
			if j.Cancelled {
				j.Status = transferCancelled
			}
			j.mu.Unlock()
			return
		}
		j.mu.Lock()
		if j.progress[i].state == transferCompleted {
			j.mu.Unlock()
			continue
		}
		if j.Cancelled {
			j.Status = transferCancelled
			j.mu.Unlock()
			return
		}
		idx := i
		typ := j.Type
		storageID := j.StorageID
		conflict := j.Conflict
		src := j.Items[i].Src
		dst := j.Items[i].Dst
		j.progress[i].state = transferRunning
		j.Status = transferRunning
		j.mu.Unlock()
		tc.persist(j)
		conflictHit, err := tc.processItem(j, idx, typ, storageID, src, dst, conflict, stop, started)
		j.mu.Lock()
		if conflictHit {
			j.Status = transferConflict
			j.progress[idx].state = "waiting_conflict"
			j.progress[idx].err = "destination exists"
			j.mu.Unlock()
			return
		}
		if err != nil {
			if err == errCancelled || j.Cancelled {
				j.progress[idx].state = transferCancelled
				j.progress[idx].err = "cancelled"
				j.Status = transferCancelled
				j.mu.Unlock()
				return
			}
			j.progress[idx].state = transferFailed
			j.progress[idx].err = sanitizeErr(err)
			j.Status = transferFailed
			j.mu.Unlock()
			return
		}
		j.progress[idx].state = transferCompleted
		j.mu.Unlock()
		tc.persist(j)
	}
	j.mu.Lock()
	j.Status = transferCompleted
	if j.totalBytes > 0 {
		j.bytesDone = j.totalBytes
	}
	j.mu.Unlock()
	tc.persist(j)
}

// gate blocks while paused and aborts on cancel. Returns false when cancelled.
func (tc *transferController) gate(j *transferJob, stop chan struct{}, started time.Time) bool {
	for {
		select {
		case <-stop:
			return false
		default:
		}
		j.mu.Lock()
		if j.Cancelled {
			j.mu.Unlock()
			return false
		}
		if !j.Paused {
			j.mu.Unlock()
			return true
		}
		if j.Status != transferPaused {
			j.Status = transferPaused
			j.mu.Unlock()
			tc.persist(j)
		} else {
			j.mu.Unlock()
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (tc *transferController) processItem(j *transferJob, idx int, kind, storageID, src, dst, conflict string, stop chan struct{}, started time.Time) (conflictHit bool, err error) {
	switch kind {
	case "delete":
		if _, e := tc.p.Trash(context.Background(), storageID, src); e != nil {
			return false, e
		}
		return false, nil
	case "copy", "move":
		exists := false
		if _, e := os.Stat(dstResolved(tc.p, storageID, dst)); e == nil {
			exists = true
		}
		if exists {
			switch conflict {
			case conflictSkip:
				j.mu.Lock()
				j.bytesDone += fileSize(tc.p, storageID, src)
				j.mu.Unlock()
				return false, nil
			case conflictAsk:
				return true, nil
			case conflictKeepBoth:
				dst = uniqueName(tc.p, storageID, dst)
			case conflictReplace:
				full, _ := tryResolve(tc.p, storageID, dst)
				if full != "" {
					_ = os.RemoveAll(full)
				}
			}
		}
		if e := copyStreamProgress(tc.p, j, storageID, src, dst, stop, started); e != nil {
			if e == errCancelled {
				return false, errCancelled
			}
			return false, e
		}
		if kind == "move" {
			if e := tc.p.Delete(context.Background(), storageID, src); e != nil {
				return false, e
			}
		}
		return false, nil
	}
	return false, fmt.Errorf("unknown job type %q", kind)
}

func fileSize(p *MockProvider, id, rel string) int64 {
	e, err := p.Stat(context.Background(), id, rel)
	if err != nil {
		return 0
	}
	return e.Size
}

func dstResolved(p *MockProvider, id, rel string) string {
	full, _ := tryResolve(p, id, rel)
	return full
}
func tryResolve(p *MockProvider, id, rel string) (string, error) {
	return (sandbox{p.roots}).resolve(id, rel, true)
}
func uniqueName(p *MockProvider, id, rel string) string {
	dir, name := resolveRelativeParent(rel)
	dot := strings.LastIndex(name, ".")
	base := name
	ext := ""
	if dot > 0 {
		base = name[:dot]
		ext = name[dot:]
	}
	for n := 1; ; n++ {
		cand := strings.TrimSuffix(dir, "/") + "/" + fmt.Sprintf("%s(%d)%s", base, n, ext)
		if _, e := os.Stat(dstResolved(p, id, cand)); e != nil {
			return cand
		}
	}
}

var errCancelled = fmt.Errorf("cancelled")

func copyStreamProgress(p *MockProvider, j *transferJob, id, src, dst string, stop chan struct{}, started time.Time) error {
	a, e := (sandbox{p.roots}).resolve(id, src, false)
	if e != nil {
		return fmt.Errorf("read %q: not found", src)
	}
	in, err := os.Open(a)
	if err != nil {
		return fmt.Errorf("read %q: io error", src)
	}
	defer in.Close()
	b, err := (sandbox{p.roots}).resolve(id, dst, true)
	if err != nil {
		return fmt.Errorf("write %q: io error", dst)
	}
	if err := os.MkdirAll(dirOf(b), 0700); err != nil {
		return fmt.Errorf("create %q: io error", dst)
	}
	out, err := os.OpenFile(b, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("write %q: io error", dst)
	}
	defer out.Close()
	buf := make([]byte, 256*1024)
	var chunk int64
	for {
		// chunk-level pause/cancel
		for {
			select {
			case <-stop:
				return errCancelled
			default:
			}
			j.mu.Lock()
			if j.Cancelled {
				j.mu.Unlock()
				return errCancelled
			}
			if !j.Paused {
				j.mu.Unlock()
				break
			}
			if j.Status != transferPaused {
				j.Status = transferPaused
				j.mu.Unlock()
				if j.persistFn != nil {
					j.persistFn(j)
				}
			} else {
				j.mu.Unlock()
			}
			time.Sleep(5 * time.Millisecond)
		}
		n, rerr := in.Read(buf)
		if n > 0 {
			wn, werr := out.Write(buf[:n])
			if werr != nil {
				return fmt.Errorf("write %q: io error", dst)
			}
			if wn != n {
				return fmt.Errorf("write %q: short write", dst)
			}
			chunk += int64(n)
			j.mu.Lock()
			j.bytesDone += int64(n)
			elapsed := time.Since(started).Seconds()
			if elapsed > 0 {
				j.speed = int64(float64(j.bytesDone) / elapsed)
			}
			if j.speed > 0 && j.totalBytes > j.bytesDone {
				j.eta = (j.totalBytes - j.bytesDone) / j.speed
			}
			j.mu.Unlock()
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return fmt.Errorf("read %q: io error", src)
		}
	}
	return out.Sync()
}

func dirOf(p string) string {
	if i := strings.LastIndex(p, string(os.PathSeparator)); i >= 0 {
		return p[:i+1]
	}
	return p
}

// sanitizeErr keeps host paths out of transfer errors.
func sanitizeErr(e error) string {
	s := e.Error()
	s = strings.ReplaceAll(s, "/tmp/", "[tmp]/")
	return s
}

// ---- handlers ----
func (s *Server) handleTransfers(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 2 && r.Method == "GET" {
		jobs := s.be.tc.list()
		out := make([]map[string]any, 0, len(jobs))
		for _, j := range jobs {
			out = append(out, transferDTO(j))
		}
		sort.SliceStable(out, func(i, k int) bool {
			pi, _ := out[i]["priority"].(int)
			pk, _ := out[k]["priority"].(int)
			return pi > pk
		})
		s.json(w, 200, map[string]any{"items": out})
		return
	}
	if len(parts) == 2 && r.Method == "POST" {
		var req struct {
			ID               string `json:"id"`
			Type             string `json:"type"`
			StorageID        string `json:"storage_id"`
			ConflictStrategy string `json:"conflict_strategy"`
			Priority         int    `json:"priority"`
			Items            []struct {
				Src string `json:"src"`
				Dst string `json:"dst"`
			} `json:"items"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			s.error(w, 400, "invalid_json", "invalid request")
			return
		}
		if req.ID == "" {
			req.ID, _ = randID()
		}
		if req.StorageID == "" {
			s.error(w, 400, "invalid_request", "storage_id is required")
			return
		}
		if len(req.Items) == 0 {
			s.error(w, 400, "invalid_request", "at least one item is required")
			return
		}
		items := make([]transferItem, 0, len(req.Items))
		for _, it := range req.Items {
			if it.Src == "" {
				s.error(w, 400, "invalid_request", "src is required")
				return
			}
			items = append(items, transferItem{Src: it.Src, Dst: it.Dst})
		}
		if e := s.be.tc.create(req.ID, req.Type, req.StorageID, req.ConflictStrategy, items, req.Priority); e != nil {
			s.error(w, 400, "invalid_request", e.Error())
			return
		}
		j, _ := s.be.tc.get(req.ID)
		go s.be.tc.start(j)
		s.json(w, 201, transferDTO(j))
		return
	}
	if len(parts) == 3 && parts[2] == "order" && r.Method == "PUT" {
		s.handleTransferOrder(w, r)
		return
	}
	if len(parts) == 4 && r.Method == "POST" {
		s.handleTransferAction(w, r, parts[2], parts[3])
		return
	}
	s.error(w, 404, "not_found", "endpoint not found")
}

func (s *Server) handleTransferAction(w http.ResponseWriter, r *http.Request, id, action string) {
	j, e := s.be.tc.get(id)
	if e != nil {
		s.error(w, 404, "not_found", "unknown transfer")
		return
	}
	switch action {
	case "pause":
		j.mu.Lock()
		j.Paused = true
		if j.Status == transferRunning || j.Status == transferQueued {
			j.Status = transferPaused
		}
		j.mu.Unlock()
		s.be.tc.persist(j)
		s.json(w, 200, transferDTO(j))
	case "resume":
		j.mu.Lock()
		wasPaused := j.Paused
		j.Paused = false
		if j.Status == transferPaused || j.Status == transferConflict {
			j.Status = transferRunning
		}
		j.mu.Unlock()
		s.be.tc.persist(j)
		if wasPaused && !s.be.tc.active(id) {
			go s.be.tc.start(j)
		}
		s.json(w, 200, transferDTO(j))
	case "retry":
		j.mu.Lock()
		j.Paused = false
		j.Cancelled = false
		j.Status = transferQueued
		for i := range j.progress {
			if j.progress[i].state != transferCompleted {
				j.progress[i].state = transferQueued
				j.progress[i].err = ""
			}
		}
		j.mu.Unlock()
		s.be.tc.persist(j)
		go s.be.tc.start(j)
		s.json(w, 200, transferDTO(j))
	case "cancel":
		s.be.tc.mu.Lock()
		if stop, ok := s.be.tc.stop[id]; ok {
			close(stop)
			delete(s.be.tc.stop, id)
		}
		s.be.tc.mu.Unlock()
		j.mu.Lock()
		j.Cancelled = true
		j.Status = transferCancelled
		for i := range j.progress {
			if j.progress[i].state == transferRunning {
				j.progress[i].state = transferCancelled
				j.progress[i].err = "cancelled"
			}
		}
		j.mu.Unlock()
		s.be.tc.persist(j)
		s.json(w, 200, transferDTO(j))
	default:
		s.error(w, 400, "invalid_action", "action must be pause, resume, retry or cancel")
	}
}

func (s *Server) handleTransferOrder(w http.ResponseWriter, r *http.Request) {
	var ids []string
	if json.NewDecoder(r.Body).Decode(&ids) != nil {
		s.error(w, 400, "invalid_json", "invalid request")
		return
	}
	for pos, id := range ids {
		if _, e := s.be.db.Exec(`INSERT INTO transfer_order(position,id) VALUES(?,?) ON CONFLICT(position) DO UPDATE SET id=excluded.id`, pos+1, id); e != nil {
			s.error(w, 400, "operation_failed", e.Error())
			return
		}
	}
	j := 0
	for _, id := range ids {
		if job, e := s.be.tc.get(id); e == nil {
			j++
			job.mu.Lock()
			job.Priority = len(ids) - j
			job.mu.Unlock()
			s.be.tc.persist(job)
		}
	}
	s.json(w, 200, map[string]any{"status": "ok", "order": ids})
}

func transferDTO(j *transferJob) map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	states := make([]map[string]any, 0, len(j.Items))
	for i, it := range j.Items {
		st := transferQueued
		errx := ""
		if i < len(j.progress) {
			st = j.progress[i].state
			errx = j.progress[i].err
		}
		states = append(states, map[string]any{"src": it.Src, "dst": it.Dst, "state": st, "error": errx})
	}
	return map[string]any{
		"id": j.ID, "type": j.Type, "storage_id": j.StorageID, "status": j.Status, "conflict_strategy": j.Conflict,
		"paused": j.Paused, "priority": j.Priority, "bytes_done": j.bytesDone, "total_bytes": j.totalBytes,
		"speed": j.speed, "eta": j.eta, "items": states,
	}
}

func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("storage_id")
	res := s.be.getAnalytics(sid)
	s.json(w, 200, res)
}
