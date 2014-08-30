package sessredis

import (
	"encoding/json"
	"github.com/gabstv/goboots"
	"github.com/hoisie/redis"
	"strconv"
	"time"
)

type RedisDBSession struct {
	client *redis.Client
	app    *goboots.App
}

func (m *RedisDBSession) SetApp(app *goboots.App) {
	m.app = app
}

func (m *RedisDBSession) GetSession(sid string) (*goboots.Session, error) {
	if m.client == nil {
		if er2 := m.connect(); er2 != nil {
			return nil, er2
		}
	}
	msession := &goboots.Session{}
	b, err := m.client.Get("goboots_sessid:" + sid)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(b, msession)

	msession.Updated = time.Now()
	msession.Flush()
	//TODO: put expire time in a config
	m.client.Expire("goboots_sessid:"+sid, 60*60*24*30)

	return msession, nil
}

func (m *RedisDBSession) PutSession(session *goboots.Session) error {
	if m.client == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	b, _ := json.Marshal(session)
	err := m.client.Set("goboots_sessid:"+session.SID, b)
	if err != nil {
		return err
	}
	_, err = m.client.Expire("goboots_sessid:"+session.SID, 60*60*24*60)
	return err
}

func (m *RedisDBSession) NewSession(session *goboots.Session) error {
	return m.PutSession(session)
}

func (m *RedisDBSession) RemoveSession(session *goboots.Session) error {
	_, err := m.client.Del("goboots_sessid:" + session.SID)
	return err
}

func (m *RedisDBSession) Cleanup(minTime time.Time) {
	// Redis keys are set to expire after each update
}

func (m *RedisDBSession) Close() {
	// this redis library does not implement a close method
}

func (m *RedisDBSession) connect() error {
	var host, auth string
	var db int

	if m.client == nil {
		m.client = &redis.Client{}
	}

	str, ok := m.app.Config.SessionDb.(string)
	if ok {
		host = m.app.Config.Databases[str].Host
		auth = m.app.Config.Databases[str].Password
		if len(m.app.Config.Databases[str].Database) > 0 {
			v, _ := strconv.ParseInt(m.app.Config.Databases[str].Database, 10, 32)
			db = int(v)
		}
	} else {
		mmap := m.app.Config.SessionDb.(map[string]string)

		host = mmap["Host"]
		auth, _ = mmap["Password"]
		if _, ok := mmap["Database"]; ok {
			v, _ := strconv.ParseInt(mmap["Database"], 10, 32)
			db = int(v)
		}
	}

	m.client.Addr = host
	if len(auth) > 0 {
		m.client.Password = auth
	}
	if db > 0 {
		m.client.Db = db
	}
	_, err := m.client.Dbsize()
	return err
}

func init() {
	goboots.RegisterSessionStorageDriver("sessredis", &RedisDBSession{})
}
