package goboots

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	requestidn_r *rand.Rand
	requestidn_m sync.Mutex
)

func requestidn() int {
	requestidn_m.Lock()
	defer requestidn_m.Unlock()
	if requestidn_r == nil {
		requestidn_r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return requestidn_r.Intn(9999)
}

func newRequestId() string {
	now := time.Now()
	n := requestidn()
	return fmt.Sprintf("%d%04d", now.UnixNano(), n)
}

type count32 int32

func (c *count32) increment() int32 {
	var next int32
	for {
		next = int32(*c) + 1
		if atomic.CompareAndSwapInt32((*int32)(c), int32(*c), next) {
			return next
		}
	}
}

func (c *count32) subtract() int32 {
	var next int32
	for {
		next = int32(*c) - 1
		if atomic.CompareAndSwapInt32((*int32)(c), int32(*c), next) {
			return next
		}
	}
}

func (c *count32) get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

type ConnectionData struct {
	Id      string
	Path    string
	Request *http.Request
	Started time.Time
}

type connectionPaths struct {
	list    map[string]int
	entries map[string]ConnectionData
	locker  sync.Mutex
}

func (c *connectionPaths) Add(r *http.Request) string {
	c.locker.Lock()
	defer c.locker.Unlock()
	if r == nil {
		return ""
	}
	p := r.URL.String()
	if c.list == nil {
		c.list = make(map[string]int)
	}
	if c.entries == nil {
		c.entries = make(map[string]ConnectionData)
	}
	if v, ok := c.list[p]; ok {
		c.list[p] = v + 1
	} else {
		c.list[p] = 1
	}
	reqid := newRequestId()
	c.entries[reqid] = ConnectionData{
		Id:      reqid,
		Path:    p,
		Request: r,
		Started: time.Now(),
	}
	return reqid
}

func (c *connectionPaths) Remove(id string) {
	c.locker.Lock()
	defer c.locker.Unlock()
	if c.list == nil {
		c.list = make(map[string]int)
	}
	if c.entries == nil {
		c.entries = make(map[string]ConnectionData)
	}
	// get path
	if cdata, ok := c.entries[id]; ok {
		p := cdata.Path
		if v, ok := c.list[p]; ok {
			if v == 1 {
				delete(c.list, p)
			} else {
				c.list[p] = v - 1
			}
		}
		delete(c.entries, id)
	}
}

func (c *connectionPaths) GetSlow(d time.Duration) []ConnectionData {
	c.locker.Lock()
	defer c.locker.Unlock()
	now := time.Now()
	results := make([]ConnectionData, 0)
	for _, data := range c.entries {
		if now.Sub(data.Started) > d {
			results = append(results, data)
		}
	}
	return results
}

func (c *connectionPaths) Cancel(id string) bool {
	c.locker.Lock()
	defer c.locker.Unlock()
	if data, ok := c.entries[id]; ok {
		if data.Request == nil {
			return false
		}
		if ctx := data.Request.Context(); ctx != nil {
			_, cancelfn := context.WithCancel(ctx)
			cancelfn()
			return true
		}
	}
	return false
}

func (c *connectionPaths) Get() map[string]int {
	c.locker.Lock()
	defer c.locker.Unlock()
	clone := make(map[string]int)
	for k, v := range c.list {
		clone[k] = v
	}
	return clone
}

type appMonitor struct {
	app                 *App
	activeThreads       count32
	openConnectionPaths connectionPaths
}

func newMonitor(app *App) appMonitor {
	return appMonitor{
		app: app,
	}
}

func (m appMonitor) ActiveThreads() int {
	return int(m.activeThreads.get())
}

func (m appMonitor) ActiveConnectionPaths() map[string]int {
	return m.openConnectionPaths.Get()
}

func (m *appMonitor) SlowConnectionPaths(d time.Duration) []ConnectionData {
	return m.openConnectionPaths.GetSlow(d)
}

func (m *appMonitor) Cancel(id string) bool {
	return m.openConnectionPaths.Cancel(id)
}
