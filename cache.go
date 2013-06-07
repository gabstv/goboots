package goboots

import (
	"sync"
	"time"
)

var (
	mu_session      sync.Mutex
	mu_bytecache    sync.Mutex
	mu_genericcache sync.Mutex
)

type ByteCache struct {
	Name       string
	Content    []byte
	LastUpdate time.Time
	IsValid    bool
}

type SessionCache struct {
	SessionID     string
	SessionObject *Session
	LastUpdate    time.Time
	Expires       time.Time
}

func (c *SessionCache) UpdateTime() {
	c.LastUpdate = time.Now()
	c.Expires = c.LastUpdate.Add(time.Hour)
}

type GenericCache struct {
	Name       string
	Content    interface{}
	LastUpdate time.Time
	IsValid    bool
}

type ByteCacheCollection struct {
	caches      map[string]*ByteCache
	maxTimeSpan time.Duration
}

type SessionCacheCollection struct {
	caches      map[string]*SessionCache
	maxTimeSpan time.Duration
}

type GenericCacheCollection struct {
	caches      map[string]*GenericCache
	maxTimeSpan time.Duration
}

func NewByteCacheCollection() *ByteCacheCollection {
	return &ByteCacheCollection{
		caches:      make(map[string]*ByteCache, 0),
		maxTimeSpan: time.Hour,
	}
}

func NewSessionCacheCollection() *SessionCacheCollection {
	return &SessionCacheCollection{
		caches:      make(map[string]*SessionCache, 0),
		maxTimeSpan: time.Hour,
	}
}

func NewGenericCacheCollection() *GenericCacheCollection {
	return &GenericCacheCollection{
		caches:      make(map[string]*GenericCache, 0),
		maxTimeSpan: time.Hour,
	}
}

func (c *ByteCacheCollection) GetCache(name string) *ByteCache {
	mu_bytecache.Lock()
	val, ok := c.caches[name]
	if !ok {
		val = &ByteCache{
			Name:    name,
			IsValid: false,
		}
		c.caches[name] = val
	}
	mu_bytecache.Unlock()
	return val
}

func (c *SessionCacheCollection) GetCache(sessionID string) *SessionCache {
	mu_session.Lock()
	val, ok := c.caches[sessionID]
	mu_session.Unlock()
	if !ok {
		return nil
	}
	return val
}

func (c *SessionCacheCollection) GetSession(sessionID string) *Session {
	var cache *SessionCache
	cache = c.GetCache(sessionID)
	if cache == nil {
		return nil
	}
	return cache.SessionObject
}

func (c *GenericCacheCollection) GetCache(name string) *GenericCache {
	mu_genericcache.Lock()
	val, ok := c.caches[name]
	if !ok {
		val = &GenericCache{
			Name:    name,
			IsValid: false,
		}
		c.caches[name] = val
	}
	mu_genericcache.Unlock()
	return val
}

func (c *ByteCacheCollection) SetCache(name string, data []byte) {
	cache := c.GetCache(name)
	mu_bytecache.Lock()
	cache.Content = data
	cache.LastUpdate = time.Now()
	cache.IsValid = true
	mu_bytecache.Unlock()
}

func (c *SessionCacheCollection) SetCache(session *Session) {
	var cache *SessionCache
	cache = c.GetCache(session.SID)
	if cache == nil {
		mu_session.Lock()
		cache = &SessionCache{
			SessionID:     session.SID,
			SessionObject: session,
			LastUpdate:    time.Now(),
			Expires:       time.Now().Add(time.Hour),
		}
		c.caches[session.SID] = cache
		mu_session.Unlock()
	}
}

func (c *GenericCacheCollection) SetCache(name string, data interface{}) {
	cache := c.GetCache(name)
	mu_genericcache.Lock()
	cache.Content = data
	cache.LastUpdate = time.Now()
	cache.IsValid = true
	mu_genericcache.Unlock()
}

func (c *ByteCacheCollection) DeleteCache(name string) {
	mu_bytecache.Lock()
	delete(c.caches, name)
	mu_bytecache.Unlock()
}

func (c *SessionCacheCollection) DeleteCache(sessionID string) {
	mu_session.Lock()
	delete(c.caches, sessionID)
	mu_session.Unlock()
}

func (c *GenericCacheCollection) DeleteCache(name string) {
	mu_genericcache.Lock()
	delete(c.caches, name)
	mu_genericcache.Unlock()
}

func (c *ByteCacheCollection) IsValid(name string) bool {
	cache := c.GetCache(name)
	if !cache.IsValid {
		return false
	}
	if time.Since(cache.LastUpdate) > c.maxTimeSpan {
		return false
	}
	return true
}

func (c *GenericCacheCollection) IsValid(name string) bool {
	cache := c.GetCache(name)
	if !cache.IsValid {
		return false
	}
	if time.Since(cache.LastUpdate) > c.maxTimeSpan {
		return false
	}
	return true
}

func (c *ByteCacheCollection) InvalidateCache(name string) {
	cache := c.GetCache(name)
	cache.IsValid = false
}

func (c *GenericCacheCollection) InvalidateCache(name string) {
	cache := c.GetCache(name)
	cache.IsValid = false
}
