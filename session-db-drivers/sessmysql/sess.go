package sessmysql

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gabstv/goboots"
	"github.com/gabstv/goboots/session-db-drivers/sessmysql/files"
	"github.com/jmoiron/sqlx"
)

// MysqlDBSession implements goboots ISessionDBEngine
type MysqlDBSession struct {
	w   *dbwrapper
	app *goboots.App
}

type dbwrapper struct {
	dbi      *sqlx.DB
	app      *goboots.App
	lastPing time.Time
	mutex    sync.Mutex
}

func newWrapper(app *goboots.App) *dbwrapper {
	w := &dbwrapper{}
	w.app = app
	return w
}

func (w *dbwrapper) db() (*sqlx.DB, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.dbi != nil {
		if w.lastPing.IsZero() || w.lastPing.Add(time.Minute*5).Before(time.Now()) {
			// ping
			err := w.dbi.Ping()
			if err != nil {
				w.app.Logger.Println("MysqlDBSession ping error:", err.Error(), "; reconnecting...")
				w.dbi.Close()
			} else {
				w.lastPing = time.Now()
				return w.dbi, nil
			}
		} else {
			return w.dbi, nil
		}
	}
	///////////
	var connstr, sdb, un, pw string
	var err error
	str, ok := w.app.Config.SessionDb.(string)
	if ok {
		connstr = w.app.Config.Databases[str].Host
		if len(connstr) < 1 {
			connstr = w.app.Config.Databases[str].Connection
		}
		sdb = w.app.Config.Databases[str].Database
		un = w.app.Config.Databases[str].User
		pw = w.app.Config.Databases[str].Password
	} else {
		mmap := w.app.Config.SessionDb.(map[string]string)
		connstr = mmap["Host"]
		if len(connstr) < 1 {
			connstr = mmap["Connection"]
		}
		sdb = mmap["Database"]
		un = mmap["User"]
		pw = mmap["Password"]
	}
	dsn := fmt.Sprintf("%v:%v@tcp(%v)/%v", un, pw, connstr, sdb)
	w.dbi, err = sqlx.Open("mysql", dsn+"?parseTime=true")
	if err != nil {
		return nil, err
	}
	w.dbi.SetConnMaxLifetime(0)
	// check if table exists!
	var one int
	err = w.dbi.QueryRowx("SELECT 1 FROM goboots_sessid LIMIT 1").Scan(&one)
	if err != nil && err.Error() != "sql: no rows in result set" {
		// table doesn't exist!
		w.app.Logger.Println(err.Error())
		// try to create table
		sqlb, err := files.RawCreateSqlBytes()
		if err != nil {
			return nil, err
		}
		_, err = w.dbi.Exec(string(sqlb))
		if err != nil {
			w.app.Logger.Println("MysqlDBSession::Create COULD NOT CREATE SESSION TABLE goboots_sessid:", err.Error(), "; Please create the table manually (file: https://github.com/gabstv/goboots/blob/master/session-db-drivers/sessmysql/raw/create.sql)")
			return nil, err
		}
	}

	return w.dbi, nil
}

func (w *dbwrapper) close() {
	if w.dbi == nil {
		return
	}
	w.dbi.Close()
	w.dbi = nil
}

// SetApp registers the goboots App to this session engine
func (m *MysqlDBSession) SetApp(app *goboots.App) {
	m.app = app
	m.w = newWrapper(app)
}

// GetSession gets a goboots session
func (m *MysqlDBSession) GetSession(sid string) (*goboots.Session, error) {
	db, err := m.w.db()
	if err != nil {
		return nil, err
	}
	var stime time.Time
	var updated time.Time
	data := make([]byte, 0)
	var shortexpires time.Time
	var shortcount uint8

	err = db.QueryRowx("SELECT time, updated, data, shortexpires, shortcount FROM goboots_sessid WHERE sid=?", sid).Scan(&stime, &updated, &data, &shortexpires, &shortcount)
	if err != nil {
		return nil, err
	}

	switch shortcount {
	case 0:
		db.Exec("UPDATE goboots_sessid SET shortcount=1, shortexpires = DATE_ADD(NOW(), INTERVAL 30 MINUTE) WHERE sid=?", sid)
	case 1:
		db.Exec("UPDATE goboots_sessid SET shortcount=2, shortexpires = DATE_ADD(NOW(), INTERVAL 5 HOUR) WHERE sid=?", sid)
	case 2:
		db.Exec("UPDATE goboots_sessid SET shortcount=3, shortexpires = DATE_ADD(NOW(), INTERVAL 5 DAY) WHERE sid=?", sid)
	case 3:
		db.Exec("UPDATE goboots_sessid SET shortcount=4, shortexpires = DATE_ADD(NOW(), INTERVAL 5 MONTH) WHERE sid=?", sid)
	}

	ses := &goboots.Session{}
	ses.SID = sid
	ses.Time = stime
	ses.Updated = updated
	ses.Data = umshl(data)
	return ses, nil
}

// PutSession saves a goboots session
func (m *MysqlDBSession) PutSession(session *goboots.Session) error {
	db, err := m.w.db()
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE goboots_sessid SET updated=NOW(), expires=?, data=? WHERE sid=?", session.GetExpires(), mshl(session.Data), session.SID)
	return err
}

// NewSession creates a new session
func (m *MysqlDBSession) NewSession(session *goboots.Session) error {
	db, err := m.w.db()
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO goboots_sessid SET sid=?, updated=NOW(), time=NOW(), expires=?, data=?, shortexpires = DATE_ADD(NOW(), INTERVAL 10 MINUTE), shortcount=0", session.SID, session.GetExpires(), mshl(session.Data))
	return err
}

// RemoveSession deletes a session from the mysql
func (m *MysqlDBSession) RemoveSession(session *goboots.Session) error {
	db, err := m.w.db()
	if err != nil {
		return err
	}
	if m.app != nil && m.app.Config != nil && m.app.Config.SessionDebug {
		db.Exec("UPDATE goboots_sessid SET delete_info=? WHERE sid=?", "goboots RemoveSession")
	}
	_, err = db.Exec("DELETE FROM goboots_sessid WHERE sid=?", session.SID)
	return err
}

func umshl(data []byte) map[string]interface{} {
	outp := make(map[string]interface{})
	json.Unmarshal(data, &outp)
	return outp
}

func mshl(data map[string]interface{}) []byte {
	if data == nil {
		return nil
	}
	bts, _ := json.Marshal(data)
	return bts
}

// Cleanup removes old sessions (abandoned sessions)
func (m *MysqlDBSession) Cleanup(minTime time.Time) {
	db, err := m.w.db()
	if err != nil {
		m.app.Logger.Println("MysqlDBSession cleanup error (could not get database):", err.Error())
		return
	}
	// remove shortcount sessions
	if m.app != nil && m.app.Config != nil && m.app.Config.SessionDebug {
		db.Exec("UPDATE goboots_sessid SET delete_info=? WHERE shortcount < 4 AND shortexpires < NOW()", "goboots shortcount")
	}
	affected, err := db.Exec("DELETE FROM goboots_sessid WHERE shortcount < 4 AND shortexpires < NOW()")
	if err != nil {
		m.app.Logger.Println("MysqlDBSession cleanup error (shortcount):", err.Error())
		return
	}
	af1, _ := affected.RowsAffected()
	// remove expired sessions
	if m.app != nil && m.app.Config != nil && m.app.Config.SessionDebug {
		db.Exec("UPDATE goboots_sessid SET delete_info=? WHERE expires < ?", "goboots default expires", minTime)
	}
	affected, err = db.Exec("DELETE FROM goboots_sessid WHERE expires < ?", minTime)
	if err != nil {
		m.app.Logger.Println("MysqlDBSession::Cleanup ok", af1, "entries removed")
		m.app.Logger.Println("MysqlDBSession cleanup error (expired):", err.Error())
		return
	}
	af2, _ := affected.RowsAffected()
	m.app.Logger.Println("MysqlDBSession::Cleanup ok", af1+af2, "entries removed")
}

// Close closes the mysql connection
func (m *MysqlDBSession) Close() {
	m.w.close()
}

func init() {
	goboots.RegisterSessionStorageDriver("sessmysql", &MysqlDBSession{})
}
