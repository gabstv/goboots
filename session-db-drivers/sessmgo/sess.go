package sessmgo

import (
	"github.com/gabstv/goboots"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

type MongoDBSession struct {
	ms  *mgo.Session
	mdb *mgo.Database
	app *goboots.App
}

func (m *MongoDBSession) SetApp(app *goboots.App) {
	m.app = app
}

func (m *MongoDBSession) GetSession(sid string) (*goboots.Session, error) {
	if m.ms == nil {
		if er2 := m.connect(); er2 != nil {
			return nil, er2
		}
	}
	msession := &goboots.Session{}
	err := m.mdb.C("goboots_sessid").Find(bson.M{"sid": sid}).One(&msession)
	if err != nil {
		return nil, err
	}
	msession.Updated = time.Now()
	msession.Flush()
	return msession, nil
}

func (m *MongoDBSession) PutSession(session *goboots.Session) error {
	if m.ms == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	return m.mdb.C("goboots_sessid").Update(bson.M{"sid": session.SID}, session)
}

func (m *MongoDBSession) NewSession(session *goboots.Session) error {
	if m.ms == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	return m.mdb.C("goboots_sessid").Insert(session)
}

func (m *MongoDBSession) RemoveSession(session *goboots.Session) error {
	return m.mdb.C("goboots_sessid").Remove(bson.M{"sid": session.SID})
}

func (m *MongoDBSession) Cleanup(minTime time.Time) {
	if m.mdb == nil {
		return
	}
	n, _ := m.mdb.C("goboots_sessid").Find(bson.M{"updated": bson.M{"$lt": minTime}}).Count()
	err := m.mdb.C("goboots_sessid").Remove(bson.M{"updated": bson.M{"$lt": minTime}})
	if err == nil {
		log.Println("MongoDBSession::Cleanup ok", n, "entries removed")
	} else {
		log.Println("MongoDBSession::Cleanup error", err)
	}
}

func (m *MongoDBSession) Close() {
	if m.ms != nil {
		m.ms.Close()
		m.mdb = nil
		m.ms = nil
	}
}

func (m *MongoDBSession) connect() error {
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
	m.ms, err = mgo.Dial(connstr)
	if err != nil {
		return err
	}
	m.ms.SetMode(mgo.Monotonic, true)
	m.mdb = m.ms.DB(db)
	if len(un) > 0 {
		err = m.mdb.Login(un, pw)
		if err != nil {
			return err
		}
	}

	// set indexes!
	index := mgo.Index{}
	index.Key = []string{"updated"}
	index.Name = "updated"
	index.Unique = false
	m.mdb.C("goboots_sessid").EnsureIndex(index)

	return nil
}

func init() {
	goboots.RegisterSessionStorageDriver("sessmgo", &MongoDBSession{})
}
