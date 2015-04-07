package sessmemory

import (
	"errors"
	"github.com/gabstv/goboots"
	"time"
)

type MemoryDbSession struct {
	gcsid     chan string
	gcs       chan *goboots.Session
	scs       chan *goboots.Session
	rcs       chan *goboots.Session
	sessions  map[string]*goboots.Session
	connected bool
	app       *goboots.App
}

func (m *MemoryDbSession) SetApp(app *goboots.App) {
	m.app = app
}

func (m *MemoryDbSession) GetSession(sid string) (*goboots.Session, error) {
	m.connect()
	m.gcsid <- sid

	sess := <-m.gcs

	if sess == nil {
		return nil, errors.New("Not found.")
	}

	sess.Updated = time.Now()
	sess.Flush()

	return sess, nil
}

func (m *MemoryDbSession) PutSession(session *goboots.Session) error {
	m.connect()
	m.scs <- session
	return nil
}

func (m *MemoryDbSession) NewSession(session *goboots.Session) error {
	return m.PutSession(session)
}

func (m *MemoryDbSession) RemoveSession(session *goboots.Session) error {
	if session == nil {
		return nil
	}
	m.connect()
	m.rcs <- session
	return nil
}

func (m *MemoryDbSession) Cleanup(minTime time.Time) {
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
	m.connected = false
}

func (m *MemoryDbSession) getSessionWorker() {
	for m.connected {
		sid := <-m.gcsid
		m.gcs <- m.sessions[sid]
	}
}

func (m *MemoryDbSession) setSessionWorker() {
	for m.connected {
		session := <-m.scs
		m.sessions[session.SID] = session
	}
}

func (m *MemoryDbSession) delSessionWorker() {
	for m.connected {
		session := <-m.rcs
		delete(m.sessions, session.SID)
	}
}

func (m *MemoryDbSession) connect() error {
	if m.connected {
		return nil
	}

	m.gcsid = make(chan string)
	m.gcs = make(chan *goboots.Session)
	m.scs = make(chan *goboots.Session)
	m.rcs = make(chan *goboots.Session)

	m.sessions = make(map[string]*goboots.Session, 0)
	m.connected = true
	go m.getSessionWorker()
	go m.setSessionWorker()
	go m.delSessionWorker()
	return nil
}

func init() {
	goboots.RegisterSessionStorageDriver("sessmemory", &MemoryDbSession{})
}
