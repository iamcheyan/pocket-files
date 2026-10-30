package main

import (
	"sync"
	"time"
)

type scanCacheEntry struct {
	res analyticsResult
	ts  time.Time
}

type scanCache struct {
	mu sync.Mutex
	m  map[string]scanCacheEntry
}

func newScanCache() *scanCache { return &scanCache{m: map[string]scanCacheEntry{}} }

func (c *scanCache) get(storageID string) (analyticsResult, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[storageID]
	if !ok {
		return analyticsResult{}, false
	}
	if time.Since(e.ts) > 5*time.Minute {
		delete(c.m, storageID)
		return analyticsResult{}, false
	}
	return e.res, true
}

func (c *scanCache) set(storageID string, res analyticsResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[storageID] = scanCacheEntry{res: res, ts: time.Now()}
}
