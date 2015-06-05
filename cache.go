package goboots

import (
	"sync"
	"time"
)

type ByteCache struct {
	Name       string
	Content    []byte
	LastUpdate time.Time
	IsValid    bool
}

type GenericCache struct {
	Name       string
	Content    interface{}
	LastUpdate time.Time
	IsValid    bool
}

type ByteCacheCollection struct {
	mu_bytecache sync.Mutex
	caches       map[string]*ByteCache
	maxTimeSpan  time.Duration
}

type GenericCacheCollection struct {
	mu_genericcache sync.Mutex
	caches          map[string]*GenericCache
	maxTimeSpan     time.Duration
}

func NewByteCacheCollection() *ByteCacheCollection {
	return &ByteCacheCollection{
		caches:      make(map[string]*ByteCache, 0),
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
	c.mu_bytecache.Lock()
	val, ok := c.caches[name]
	if !ok {
		val = &ByteCache{
			Name:    name,
			IsValid: false,
		}
		c.caches[name] = val
	}
	c.mu_bytecache.Unlock()
	return val
}

func (c *GenericCacheCollection) GetCache(name string) *GenericCache {
	c.mu_genericcache.Lock()
	val, ok := c.caches[name]
	if !ok {
		val = &GenericCache{
			Name:    name,
			IsValid: false,
		}
		c.caches[name] = val
	}
	c.mu_genericcache.Unlock()
	return val
}

func (c *ByteCacheCollection) SetCache(name string, data []byte) {
	cache := c.GetCache(name)
	c.mu_bytecache.Lock()
	cache.Content = data
	cache.LastUpdate = time.Now()
	cache.IsValid = true
	c.mu_bytecache.Unlock()
}

func (c *GenericCacheCollection) SetCache(name string, data interface{}) {
	cache := c.GetCache(name)
	c.mu_genericcache.Lock()
	cache.Content = data
	cache.LastUpdate = time.Now()
	cache.IsValid = true
	c.mu_genericcache.Unlock()
}

func (c *ByteCacheCollection) DeleteCache(name string) {
	c.mu_bytecache.Lock()
	delete(c.caches, name)
	c.mu_bytecache.Unlock()
}

func (c *GenericCacheCollection) DeleteCache(name string) {
	c.mu_genericcache.Lock()
	delete(c.caches, name)
	c.mu_genericcache.Unlock()
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

type ISessionDBEngine interface {
	SetApp(app *App)
	GetSession(sid string) (*Session, error)
	PutSession(session *Session) error
	NewSession(session *Session) error
	RemoveSession(session *Session) error
	Cleanup(minTime time.Time)
	Close()
}

var devNullSess = &Session{SID: "nil"}

type SessionDevNull struct {
	app *App
}

func (s *SessionDevNull) SetApp(app *App) {
	s.app = app
}

func (s *SessionDevNull) GetSession(sid string) (*Session, error) {
	return devNullSess, nil
}

func (s *SessionDevNull) PutSession(s2 *Session) error {
	return nil
}

func (s *SessionDevNull) NewSession(s2 *Session) error {
	return nil
}

func (s *SessionDevNull) RemoveSession(s2 *Session) error {
	return nil
}

func (s *SessionDevNull) Cleanup(minTime time.Time) {

}

func (s *SessionDevNull) Close() {

}
