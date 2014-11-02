package goboots

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

type testController struct {
	Controller
}

func (t *testController) Test(in *In) *Out {
	v := struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}{true, ""}
	return in.OutputJSON(v)
}

func TestApp(t *testing.T) {
	RegisterSessionStorageDriver("sessmemory", &testDBSession{})
	var app App
	app.Config.Name = "Test App"
	app.Config.HostAddr = ":8001"
	app.Config.GlobalPageTitle = "Test App - "
	app.Config.OldRouteMethod = true
	app.InitSessionStorage("sessmemory")
	r0 := OldRoute{}
	r0.Path = "/"
	r0.Controller = "testController"
	r0._t = routeMethodExact
	r0.Method = "Test"
	app.Routes = []OldRoute{r0}

	app.RegisterController(&testController{})
	app.Filters = []Filter{
		CompressFilter,
	}

	t.Log("TESTING APP\n")

	go app.Listen()

	resp, err := http.Get("http://localhost:8001/")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.Header)
	//t.Log(string(b), len(b))

	if string(b) != `{"success":true,"error":""}` {
		t.Fatal("expected output mismatch!")
	}

	cl := &http.Client{}
	req, _ := http.NewRequest("GET", "http://localhost:8001/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err = cl.Do(req)

	if err != nil {
		t.Fatal(err)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.Header)
	//t.Log(b, len(b))

	if bytes.Compare(b, []byte{
		31, 139, 8, 0, 0, 9, 110, 136, 0, 255, 170, 86, 42, 46, 77, 78,
		78, 45, 46, 86, 178, 42, 41, 42, 77, 213, 81, 74, 45, 42, 202, 47,
		82, 178, 82, 82, 170, 5, 4, 0, 0, 255, 255, 234, 150, 84, 37, 27,
		0, 0, 0,
	}) != 0 {
		t.Fatal("gzipped output mismatch!")
	}
}

type testDBSession struct {
	sessions map[string]*Session
	app      *App
}

func (m *testDBSession) SetApp(app *App) {
	m.app = app
}

func (m *testDBSession) GetSession(sid string) (*Session, error) {

	if m.sessions == nil {
		m.sessions = make(map[string]*Session)
	}

	sess, ok := m.sessions[sid]

	if !ok {
		return nil, errors.New("Not found.")
	}

	sess.Updated = time.Now()
	sess.Flush()

	return sess, nil
}

func (m *testDBSession) PutSession(session *Session) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*Session)
	}
	m.sessions[session.SID] = session
	return nil
}

func (m *testDBSession) NewSession(session *Session) error {
	return m.PutSession(session)
}

func (m *testDBSession) RemoveSession(session *Session) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*Session)
	}
	if session == nil {
		return nil
	}
	delete(m.sessions, session.SID)
	return nil
}

func (m *testDBSession) Cleanup(minTime time.Time) {
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
	log.Println("testDBSession::Cleanup ok", len(delList), "entries removed")
}

func (m *testDBSession) Close() {

}
