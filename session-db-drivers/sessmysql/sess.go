package sessmysql

import (
	"encoding/json"
	"fmt"
	"github.com/gabstv/goboots"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"io/ioutil"
	"time"
)

type MysqlDBSession struct {
	mdb *sqlx.DB
	app *goboots.App
}

func (m *MysqlDBSession) SetApp(app *goboots.App) {
	m.app = app
}

func (m *MysqlDBSession) GetSession(sid string) (*goboots.Session, error) {
	if m.mdb == nil {
		if er2 := m.connect(); er2 != nil {
			return nil, er2
		}
	}
	msession := &goboots.Session{}
	var stime time.Time
	var updated time.Time
	//var expires time.Time
	var data string
	//
	qrx := m.mdb.QueryRowx("SELECT time AS stime, updated, data FROM goboots_sessid WHERE sid=?", sid)

	err := qrx.Scan(&stime, &updated, &data)
	if err != nil {
		return nil, err
	}
	msession.Time = stime
	msession.Updated = updated
	msession.Data = umshl(data)
	msession.Flush()
	return msession, nil
}

func (m *MysqlDBSession) PutSession(session *goboots.Session) error {
	if m.mdb == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	_, err := m.mdb.Exec("UPDATE goboots_sessid SET updated=NOW(), expires=?, data=? WHERE sid=?", session.GetExpires(), mshl(session.Data), session.SID)
	return err
}

func (m *MysqlDBSession) NewSession(session *goboots.Session) error {
	if m.mdb == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	_, err := m.mdb.Exec("INSERT INTO goboots_sessid SET sid=?, updated=NOW(), time=NOW(), expires=?, data=?", session.SID, session.GetExpires(), mshl(session.Data))
	return err
}

func (m *MysqlDBSession) RemoveSession(session *goboots.Session) error {
	if m.mdb == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	_, err := m.mdb.Exec("DELETE FROM goboots_sessid WHERE sid=?", session.SID)
	return err
}

func (m *MysqlDBSession) Cleanup(minTime time.Time) {
	if m.mdb == nil {
		return
	}
	result, err := m.mdb.Exec("DELETE FROM goboots_sessid WHERE expires < ?", minTime)

	if err == nil {
		raf, _ := result.RowsAffected()
		m.app.Logger.Println("MysqlDBSession::Cleanup ok", raf, "entries removed")
	} else {
		m.app.Logger.Println("MysqlDBSession::Cleanup error", err)
	}
}

func (m *MysqlDBSession) Close() {
	if m.mdb != nil {
		m.mdb.Close()
		m.mdb = nil
	}
}

func (m *MysqlDBSession) connect() error {
	var connstr, db, un, pw string
	var err error
	str, ok := m.app.Config.SessionDb.(string)
	if ok {
		connstr = m.app.Config.Databases[str].Connection
		db = m.app.Config.Databases[str].Database
		un = m.app.Config.Databases[str].User
		pw = m.app.Config.Databases[str].Password
	} else {
		mmap := m.app.Config.SessionDb.(map[string]string)
		connstr = mmap["Connection"]
		db = mmap["Database"]
		un = mmap["User"]
		pw = mmap["Password"]
	}
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v", un, pw, connstr, db)
	m.mdb, err = sqlx.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		return err
	}
	// check if table exists!
	sqlsb, err := ioutil.ReadFile("create.sql")
	if err != nil {
		return err
	}
	m.mdb.Exec(string(sqlsb))

	return nil
}

func init() {
	goboots.RegisterSessionStorageDriver("sessmysql", &MysqlDBSession{})
}

func mshl(data map[string]interface{}) string {
	bts, _ := json.Marshal(data)
	return string(bts)
}

func umshl(data string) map[string]interface{} {
	outp := make(map[string]interface{})
	json.Unmarshal([]byte(data), &outp)
	return outp
}
