package main

import (
	"context"
	"sync"
	"time"
)

// scanProgress tracks an in-flight analytics scan so the API can report real
// progress instead of a field that is always zero.
type scanProgress struct {
	Backgrounded bool
	Progress     float64
	Files        int
}

type scanJob struct {
	mu       sync.Mutex
	progress scanProgress
	result   *analyticsResult
	done     chan struct{}
}

type scanJobs struct {
	mu   sync.Mutex
	jobs map[string]*scanJob
}

func newScanJobs() *scanJobs { return &scanJobs{jobs: map[string]*scanJob{}} }

// Threshold above which a scan is handed to a goroutine. Small trees (the mock
// volumes and typical folders) stay synchronous so the first response is
// complete rather than a progress-only payload.
const asyncScanThreshold = 500

func (s *scanJobs) begin(storageID string) (*scanJob, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.jobs[storageID]; ok {
		select {
		case <-existing.done:
			// Finished job: reuse the completed result.
			return existing, false
		default:
			return existing, true
		}
	}
	job := &scanJob{done: make(chan struct{})}
	s.jobs[storageID] = job
	return job, false
}

func (s *scanJobs) lookup(storageID string) *scanJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[storageID]
}

// getAnalytics returns cached results, a running scan's progress, or starts a
// background scan when the tree is large enough to be worth deferring.
func (b *Backend) getAnalyticsWithProgress(ctx context.Context, storageID string) (analyticsResult, bool) {
	if res, ok := b.sc.get(storageID); ok {
		return res, false
	}
	// Count a shallow sample to decide whether this tree is large.
	if b.estimateTreeSize(storageID, asyncScanThreshold) < asyncScanThreshold {
		res := b.computeAnalytics(storageID)
		b.sc.set(storageID, res)
		return res, false
	}

	job, running := b.sj.begin(storageID)
	if running {
		job.mu.Lock()
		progress, result := job.progress, job.result
		job.mu.Unlock()
		if result != nil {
			b.sc.set(storageID, *result)
			return *result, false
		}
		placeholder := b.capacityOnly(storageID)
		placeholder.Backgrounded = true
		placeholder.Progress = progress.Progress
		return placeholder, true
	}

	go func() {
		defer close(job.done)
		res := b.computeAnalyticsTracked(storageID, job)
		job.mu.Lock()
		job.result = &res
		job.progress.Backgrounded = false
		job.progress.Progress = 1
		job.mu.Unlock()
		b.sc.set(storageID, res)
	}()

	placeholder := b.capacityOnly(storageID)
	placeholder.Backgrounded = true
	placeholder.Progress = 0
	return placeholder, true
}

// capacityOnly fills total/used/free without walking the tree.
func (b *Backend) capacityOnly(storageID string) analyticsResult {
	d, _ := b.getDevice(storageID)
	res := analyticsResult{
		StorageID: storageID, Mock: d.Mock, CategoryCounts: map[string]int{},
		LargestFiles: []map[string]any{}, LargestDirectories: []map[string]any{},
		CacheTimestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if total, used, free, err := b.storageUsage(storageID); err == nil {
		res.Total, res.Used, res.Free = total, used, free
	}
	return res
}

// estimateTreeSize counts entries breadth-first up to limit, so deciding whether
// to background a scan costs only a bounded number of provider calls.
func (b *Backend) estimateTreeSize(storageID string, limit int) int {
	storageIDs := []string{storageID}
	if storageID == "" {
		storageIDs = nil
		for _, d := range b.p.AllDevices() {
			storageIDs = append(storageIDs, d.ID)
		}
	}
	count := 0
	for _, id := range storageIDs {
		queue := []string{""}
		for len(queue) > 0 && count < limit {
			rel := queue[0]
			queue = queue[1:]
			items, err := b.p.Browse(context.Background(), id, rel)
			if err != nil {
				continue
			}
			for _, item := range items {
				count++
				if count >= limit {
					return count
				}
				if item.Kind == "directory" {
					queue = append(queue, item.ID)
				}
			}
		}
	}
	return count
}
