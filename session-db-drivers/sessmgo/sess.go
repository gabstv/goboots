package sessmgo

import (
	"github.com/gabstv/goboots"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type MongoDBSession struct {
	ms  *mgo.Session
	mdb *mgo.Database
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

func (m *MongoDBSession) Close() {
	if m.ms != nil {
		m.ms.Close()
		m.mdb = nil
		m.ms = nil
	}
}

func (m *MongoDBSession) connect() error {
	var connstr, db string
	var err error
	str, ok := goboots.APP.Config.SessionDb.(string)
	if ok {
		connstr = goboots.APP.Config.Databases[str].Connection
		db = goboots.APP.Config.Databases[str].Database
	} else {
		mmap := goboots.APP.Config.Databases[str].(map[string]string)
		connstr = mmap["Connection"]
		db = mmap["Database"]
	}
	m.ms, err = mgo.Dial(connstr)
	if err != nil {
		return err
	}
	m.ms.SetMode(mgo.Monotonic, true)
	m.mdb = m.ms.DB(db)
	return nil
}

func init() {
	goboots.RegisterSessionStorageDriver("sessmgo", &MongoDBSession{})
}
