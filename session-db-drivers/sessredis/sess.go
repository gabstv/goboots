package sessredis

import (
	"encoding/json"
	"github.com/gabstv/goboots"
	"github.com/hoisie/redis"
	"strconv"
)

type RedisDBSession struct {
	client *redis.Client
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
	return msession, nil
}

func (m *RedisDBSession) PutSession(session *goboots.Session) error {
	if m.client == nil {
		if er2 := m.connect(); er2 != nil {
			return er2
		}
	}
	b, _ := json.Marshal(session)
	err := m.client.Set("goboots_sessid:"+sid, b)
	if err != nil {
		return err
	}
	return m.client.Expire("goboots_sessid:"+sid, 60*60*24*60)
}

func (m *RedisDBSession) NewSession(session *goboots.Session) error {
	return m.PutSession(session)
}

func (m *RedisDBSession) RemoveSession(session *goboots.Session) error {
	_, err := m.client.Del("goboots_sessid:" + sid)
	return err
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

	str, ok := goboots.APP.Config.SessionDb.(string)
	if ok {
		host = goboots.APP.Config.Databases[str].Host
		auth = goboots.APP.Config.Databases[str].Password
		if len(goboots.APP.Config.Databases[str].Database) > 0 {
			db = int(strconv.ParseInt(goboots.APP.Config.Databases[str].Database, 10, 32))
		}
	} else {
		mmap := goboots.APP.Config.SessionDb.(map[string]string)

		host = mmap["Host"]
		auth, _ = mmap["Password"]
		if _, ok := mmap["Database"]; ok {
			db = int(strconv.ParseInt(mmap["Database"], 10, 32))
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
