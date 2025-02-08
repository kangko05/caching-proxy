package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type CacheStrategy string

const (
	FIFO CacheStrategy = "first in first out"
	LRU  CacheStrategy = "least recently used"
	LFU  CacheStrategy = "least frequently used"
)

type CacheClient interface {
	Add(string, http.Header, []byte)
	Check(string) (CacheItem, bool)
	Run()
	Stop()
}

type CacheItem struct {
	uri            string
	responseBody   []byte
	responseHeader http.Header
}

type FIFOCache struct {
	cacheQueue *Queue[CacheItem]
	cacheMap   map[string]int // request uri - q idx
	size       int

	ctx     context.Context
	cancel  context.CancelFunc
	addCh   chan CacheItem
	clearCh chan struct{}
	mu      sync.RWMutex
}

func InitCacheClient(strategy CacheStrategy, size int) CacheClient {
	ctx, cancel := context.WithCancel(context.Background())

	switch strategy {
	case FIFO:
		fallthrough
	default:
		return &FIFOCache{
			cacheQueue: NewQueue[CacheItem](),
			cacheMap:   make(map[string]int, size),
			size:       size,
			ctx:        ctx,
			cancel:     cancel,
			addCh:      make(chan CacheItem),
			clearCh:    make(chan struct{}),
			mu:         sync.RWMutex{},
		}
	}
}

func (fc *FIFOCache) Add(reqUri string, respHeader http.Header, respBody []byte) {
	fc.addCh <- CacheItem{uri: reqUri, responseHeader: respHeader, responseBody: respBody}
}

func (fc *FIFOCache) addCacheItem(cacheItem CacheItem) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if fc.cacheQueue.Len() >= fc.size {
		it, ok := fc.cacheQueue.Get()
		if !ok {
			return
		}

		delete(fc.cacheMap, it.uri)

		for key := range fc.cacheMap {
			fc.cacheMap[key] -= 1
		}
	}

	idx := fc.cacheQueue.Add(cacheItem)
	fc.cacheMap[cacheItem.uri] = idx
}

// if exists -> it, exists
func (fc *FIFOCache) Check(reqUri string) (CacheItem, bool) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	idx, ok := fc.cacheMap[reqUri]
	if ok {
		return fc.cacheQueue.Peek(idx), true
	}

	return CacheItem{}, false
}

func (fc *FIFOCache) ClearCache() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.cacheQueue = NewQueue[CacheItem]()
	fc.cacheMap = make(map[string]int, 0)
}

func (fc *FIFOCache) Run() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case it := <-fc.addCh:
			fc.addCacheItem(it)

		case <-fc.clearCh:
			fc.ClearCache()

		case <-fc.ctx.Done():
			return

		case <-ticker.C:
			if fc.cacheQueue.Len() > 0 {
				it, _ := fc.cacheQueue.Get()
				delete(fc.cacheMap, it.uri)
			}
		}
	}
}

func (fc *FIFOCache) Stop() {
	fc.cancel()
	close(fc.addCh)
	close(fc.clearCh)
}
