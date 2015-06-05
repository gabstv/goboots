package goboots

import (
	"bytes"
	"errors"
	"golang.org/x/net/websocket"
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

func (t *testController) TestWebsocket(in *In) *Out {
	msg := make([]byte, 512)
	n, err := in.Wsock.Read(msg)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Receive: %s\n", msg[:n])

	m, err := in.Wsock.Write(msg[:n])
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Sent: %s\n", msg[:m])
	return nil
}

func TestApp(t *testing.T) {
	RegisterSessionStorageDriver("sessmemory", &testDBSession{})
	app := NewApp()
	app.Config = &AppConfig{
		Name:            "Test App",
		HostAddr:        ":8001",
		GlobalPageTitle: "Test App - ",
		OldRouteMethod:  true,
	}
	app.InitSessionStorage("sessmemory")
	r0 := OldRoute{}
	r0.Path = "/"
	r0.Controller = "testController"
	r0._t = routeMethodExact
	r0.Method = "Test"
	// ws
	r1 := OldRoute{}
	r1.Path = "/ws"
	r1.Controller = "testController"
	r1._t = routeMethodExact
	r1.Method = "TestWebsocket"
	app.Routes = []OldRoute{r0, r1}

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
		t.Fatal("expected output mismatch!", string(b), `{"success":true,"error":""}`)
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
		123, 34, 115, 117, 99, 99, 101, 115,
		115, 34, 58, 116, 114, 117, 101, 44,
		34, 101, 114, 114, 111, 114, 34, 58, 34, 34, 125,
	}) != 0 {
		t.Fatal("gzipped output mismatch!", b)
	}

	// test websocket
	ws, err := websocket.Dial("ws://localhost:8001/ws", "", "http://localhost/")
	if err != nil {
		t.Fatal("could not dial (websocket)", err)
	}
	message := []byte("hello, websocket!")
	_, err = ws.Write(message)
	if err != nil {
		t.Fatal("could not write (websocket)", err)
	}
	var msg = make([]byte, 512)
	n, err := ws.Read(msg)
	if err != nil {
		t.Fatal("could not read (websocket)", err)
	}
	t.Log(string(msg[:n]))
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
