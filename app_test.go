package goboots

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
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
	_, msg, err := in.Wsock.ReadMessage()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Receive: %s\n", msg)

	err = in.Wsock.WriteMessage(websocket.TextMessage, []byte("hello"))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Sent: %s\n", "hello")
	return nil
}

func TestApp(t *testing.T) {
	RegisterSessionStorageDriver("sessmemory", &testDBSession{})
	app := NewApp()
	app.Config = &AppConfig{
		Name:            "Test App",
		HostAddr:        ":8001",
		GlobalPageTitle: "Test App - ",
	}
	app.InitSessionStorage("sessmemory")
	ctrlr := &testController{}

	app.RegisterController(ctrlr)
	app.Filters = []Filter{
		CompressFilter,
	}

	app.ServeMux.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Write([]byte(`{"success":true,"error":""}`))
	})

	t.Log("TESTING APP\n")

	go func() {
		err90 := app.Listen()
		if err90 != nil {
			t.Log("error listening", err90)
		}
	}()

	time.Sleep(time.Second * 2)

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

type testTemplateProcessor struct {
}

func (t *testTemplateProcessor) Walk(root string, walkFn filepath.WalkFunc) error {
	log.Println("testTemplateProcessor Walk", root)
	walkFn("test.pug", NewMockFileInfo("test.pug", 1000, os.ModeDevice, time.Now(), false), nil)
	return nil
}
func (t *testTemplateProcessor) ReadFile(filename string) ([]byte, error) {
	if filename == "test.pug" {
		out := `doctype 5:
html:
  body:
    p Hello world!`
		return []byte(out), nil
	}
	return nil, errors.New("file not found")
}

func TestTemplateProcessor(t *testing.T) {
	RegisterSessionStorageDriver("sessmemory", &testDBSession{})
	app := NewApp()
	app.TemplateProcessor = &testTemplateProcessor{}
	app.Config = &AppConfig{
		Name:            "Test App",
		HostAddr:        ":8001",
		GlobalPageTitle: "Test App - ",
	}
	err := app.loadTemplates()
	if err != nil {
		t.Fatalf("TestTemplateProcessor ERROR %v\n", err.Error())
	}
	tppll := app.GetViewTemplate("test.pug")
	if tppll == nil {
		t.Fatalf("app.GetViewTemplate('%v') is nil\n", "test.pug")
	}
	rw := &MockResponseWriter{}
	err = tppll.Execute(rw, nil)
	if err != nil {
		t.Fatalf("TestTemplateProcessor ERROR 2 %v\n", err.Error())
	}
	str0 := string(rw.Body())
	str1 := `
<!DOCTYPE html>
<html>
    <body>
        <p>Hello world!</p>
    </body>
</html>`
	if str0 != str1 {
		t.Fatalf("Template is \n'%v'\nshould be:\n'%v'\n", str0, str1)
	}
}
