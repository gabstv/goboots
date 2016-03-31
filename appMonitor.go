package goboots

import (
	"sync"
	"sync/atomic"
)

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

type connectionPaths struct {
	list   map[string]int
	locker sync.Mutex
}

func (c *connectionPaths) Add(p string) {
	c.locker.Lock()
	defer c.locker.Unlock()
	if v, ok := c.list[p]; ok {
		c.list[p] = v + 1
	} else {
		c.list[p] = 1
	}
}

func (c *connectionPaths) Remove(p string) {
	c.locker.Lock()
	defer c.locker.Unlock()
	if v, ok := c.list[p]; ok {
		if v == 1 {
			delete(c.list, p)
		} else {
			c.list[p] = v - 1
		}
	}
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
