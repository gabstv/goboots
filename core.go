package goboots

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gabstv/i18ngo"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"
)

const (
	ErrNil = 0
	// TLS
	ErrTLSNil = 100
)

var (
	sessionDbs   map[string]ISessionDBEngine
	curSessionDb ISessionDBEngine
)

func RegisterSessionStorageDriver(name string, engine ISessionDBEngine) {
	if sessionDbs == nil {
		sessionDbs = make(map[string]ISessionDBEngine, 0)
	}
	sessionDbs[name] = engine
}

func InitSessionStorage(driver string) error {
	if sessionDbs == nil {
		return errors.New("No session storage registered.")
	}
	if _, ok := sessionDbs[driver]; !ok {
		return errors.New("The session storage driver " + driver + " does not exist.")
	}
	curSessionDb = sessionDbs[driver]
	return nil
}

func CloseSessionStorage() {
	if curSessionDb != nil {
		curSessionDb.Close()
	}
}

func initAnySessionStorage() {
	if curSessionDb != nil {
		return
	}
	if sessionDbs == nil {
		//TODO: should panic
		return
	}
	for _, v := range sessionDbs {
		curSessionDb = v
		break
	}
}

type AppError struct {
	Id      int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

type env struct {
	path       string
	controller string
	action     string
	params     []string
	request    *http.Request
	body       string
	form       map[string][]string
}

type ISession interface {
	GetData() (string, map[string]interface{}, time.Time)
}

type Session struct {
	SID  string
	Data map[string]interface{}
	Time time.Time
}

func (s *Session) GetData() (string, map[string]interface{}, time.Time) {
	return s.SID, s.Data, s.Time
}

func (s *Session) Flush() {
	FlushSession(s)
}

func (s *Session) GetBool(key string) (bool, bool) {
	iface, ok := s.Data[key]
	if !ok {
		return false, false
	}
	return bool(reflect.ValueOf(iface).Bool()), true
}

func (s *Session) GetBoolD(key string, defaultValue bool) bool {
	iface, ok := s.Data[key]
	if !ok {
		return defaultValue
	}
	return bool(reflect.ValueOf(iface).Bool())
}

func (s *Session) GetInt32(key string) (int, bool) {
	iface, ok := s.Data[key]
	if !ok {
		return 0, false
	}
	return int(reflect.ValueOf(iface).Int()), true
}

func (s *Session) GetInt32D(key string, defaultValue int) int {
	iface, ok := s.Data[key]
	if !ok {
		return defaultValue
	}
	return int(reflect.ValueOf(iface).Int())
}

func (s *Session) GetString(key string) (string, bool) {
	iface, ok := s.Data[key]
	if !ok {
		return "", false
	}
	return reflect.ValueOf(iface).String(), true
}

func (s *Session) GetStringD(key string, defaultValue string) string {
	iface, ok := s.Data[key]
	if !ok {
		return defaultValue
	}
	return reflect.ValueOf(iface).String()
}

func GetSession(w http.ResponseWriter, r *http.Request) *Session {
	initAnySessionStorage()
	var cookie *http.Cookie
	var sid string
	var err error
	//LOAD
	cookie, err = r.Cookie("goboots_sessid")
	if err == nil {
		sid = cookie.Value
		if _validateSidString(sid) {
			var msession *Session
			// try get from cache
			msession = APP.SessionCache.GetSession(sid)
			if msession != nil {
				cache := APP.SessionCache.GetCache(sid)
				mu_session.Lock()
				cache.UpdateTime()
				mu_session.Unlock()
				return msession
			}
			// get from a saved location
			msession, err = curSessionDb.GetSession(sid)
			if err == nil {
				APP.SessionCache.SetCache(msession)
				return msession
			}
			fmt.Printf("SESSION ERROR :( [%s] %s\n", sid, err.Error())
			// not found, generate a new sid
		} else {
			fmt.Printf("COULD NOT VALIDATE :( [%s]\n", sid)
		}
	}
	// gen session
	rand.Seed(time.Now().UnixNano())
	var uuid [16]byte
	for i := 0; i < 16; i++ {
		uuid[i] = byte(rand.Intn(255))
	}
	// secrets
	//uuid[6] = (4 << 4) | (uuid[6] & 15)
	//uuid[8] = (2 << 4) | (uuid[8] & 15)
	//sid = fmt.Sprintf("%x%x%x%x%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
	sid = fmt.Sprintf("%x", uuid)
	log.Println(sid)
	session := &Session{
		SID:  sid,
		Data: make(map[string]interface{}),
		Time: time.Now(),
	}
	err = curSessionDb.NewSession(session)
	__panic(err)
	SetCookieAdv(w, "goboots_sessid", sid, "/", "", time.Now().AddDate(0, 1, 0), 0, false, true)
	return session
}

func GetUserLang(w http.ResponseWriter, r *http.Request) string {
	var l string
	// get lang from cookies
	cookie, err := r.Cookie("lang")
	if err != nil && cookie != nil {
		l = cookie.Value
		return l
	}
	// get lang from http request
	alh := r.Header.Get("Accept-Language")
	// 2013-07-29 : not all clients actually send this header (googlebot/wget etc)
	if len(alh) > 1 {
		validlangs := i18ngo.GetLanguageCodes()
		// break alh
		alh0 := strings.Split(alh, ",")
		for _, v := range alh0 {
			lc := strings.Split(v, ";")[0][0:2]
			if StringIndexOf(validlangs, lc) != -1 {
				l = lc
				break
			}
		}
		if len(l) > 0 {
			// just update the cookies
			SetCookieSimple(w, "lang", l)
		} else {
			l = i18ngo.GetDefaultLanguageCode()
		}
		return l
	}
	return i18ngo.GetDefaultLanguageCode()
}

func SetUserLang(w http.ResponseWriter, r *http.Request, langcode string) {
	SetCookieSimple(w, "lang", langcode)
}

func FlushSession(s *Session) error {
	return curSessionDb.PutSession(s)
}

func DestroySession(w http.ResponseWriter, r *http.Request, s *Session) {
	curSessionDb.RemoveSession(s)
	APP.SessionCache.DeleteCache(s.SID)
	SetCookieAdv(w, "goboots_sessid", "", "/", "", time.Now(), 1, false, true)
}

func _validateSidString(sid string) bool {
	if len(sid) != 32 {
		return false
	}
	for _, c := range sid {
		// check if sid contains invalid chars
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

//func _parseCookies(r *http.Request) (map[string]string, error) {
//cookies := make(map[string]string)
//c := r.Cookies()
//c[0].
//}

type templateInfo struct {
	path       string
	data       *template.Template
	lastUpdate time.Time
}

func SetCookieAdv(w http.ResponseWriter, name string, value string, path string, domain string, expires time.Time, maxage int, secure bool, httpOnly bool) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   domain,
		Expires:  expires,
		MaxAge:   maxage,
		Secure:   secure,
		HttpOnly: httpOnly,
	}
	http.SetCookie(w, cookie)
}

func SetCookieSimple(w http.ResponseWriter, key string, value string) {
	SetCookieAdv(w, key, value, "/", "", time.Now().AddDate(100, 0, 0), 0, false, false)
}

func StrConcat(strings ...string) string {
	var buffer bytes.Buffer
	for _, str := range strings {
		buffer.WriteString(str)
	}
	return buffer.String()
}

func DeleteItem(slice interface{}, i int) {
	v := reflect.ValueOf(slice).Elem()
	v.Set(reflect.AppendSlice(v.Slice(0, i), v.Slice(i+1, v.Len())))
}

func InsertItem(slice interface{}, i int, val interface{}) {
	v := reflect.ValueOf(slice).Elem()
	v.Set(reflect.AppendSlice(v.Slice(0, i+1), v.Slice(i, v.Len())))
	v.Index(i).Set(reflect.ValueOf(val))
}

func StringIndexOf(haystack []string, needle string) int {
	for i, v := range haystack {
		if v == needle {
			return i
		}
	}
	return -1
}

func IntMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func __panic(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

func FormatPath(rawpath string) string {
	cwd, _ := os.Getwd()
	if strings.HasPrefix(rawpath, "./") {
		return filepath.Clean(cwd + rawpath[1:])
	} else if strings.HasPrefix(rawpath, "../") {
		return filepath.Clean(cwd + "/" + rawpath)
	}
	return filepath.Clean(rawpath)
}
