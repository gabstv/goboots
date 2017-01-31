package sessmemory

import (
	"errors"
	"github.com/gabstv/goboots"
	"sync"
	"time"
)

type MemoryDbSession struct {
	sessions      map[string]*goboots.Session
	app           *goboots.App
	sessions_lock sync.RWMutex
}

func (m *MemoryDbSession) SetApp(app *goboots.App) {
	m.app = app
}

func (m *MemoryDbSession) GetSession(sid string) (*goboots.Session, error) {
	m.sessions_lock.Lock()
	defer m.sessions_lock.Unlock()

	if m.sessions == nil {
		return nil, errors.New("not found")
	}

	if sessfile, ok := m.sessions[sid]; ok {
		sessfile.Updated = time.Now()
		sessfile.Flush()
		return sessfile, nil
	}

	return nil, errors.New("not found")
}

func (m *MemoryDbSession) PutSession(session *goboots.Session) error {
	if session == nil {
		return errors.New("session is nil")
	}
	m.sessions_lock.Lock()
	defer m.sessions_lock.Unlock()

	if m.sessions == nil {
		m.sessions = make(map[string]*goboots.Session)
	}

	m.sessions[session.SID] = session

	return nil
}

func (m *MemoryDbSession) NewSession(session *goboots.Session) error {
	return m.PutSession(session)
}

func (m *MemoryDbSession) RemoveSession(session *goboots.Session) error {
	if session == nil || m.sessions == nil {
		return nil
	}

	m.sessions_lock.Lock()
	defer m.sessions_lock.Unlock()

	delete(m.sessions, session.SID)

	return nil
}

func (m *MemoryDbSession) Cleanup(minTime time.Time) {
	m.sessions_lock.Lock()
	defer m.sessions_lock.Unlock()

	if m.sessions == nil {
		return
	}
	//TODO: implement a faster cleanup method
	delList := make([]string, 0, len(m.sessions))
	for k, v := range m.sessions {
		if minTime.After(v.Updated) {
			delList = append(delList, k)
		}
	}
	for _, v := range delList {
		delete(m.sessions, v)
	}
	m.app.Logger.Println("MemoryDbSession::Cleanup ok", len(delList), "entries removed")
}

func (m *MemoryDbSession) Close() {

}

func init() {
	goboots.RegisterSessionStorageDriver("sessmemory", &MemoryDbSession{})
}
