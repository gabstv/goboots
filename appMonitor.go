package goboots

import (
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

type appMonitor struct {
	app           *App
	activeThreads count32
}

func newMonitor(app *App) appMonitor {
	return appMonitor{
		app: app,
	}
}

func (m appMonitor) ActiveThreads() int {
	return int(m.activeThreads.get())
}
