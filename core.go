package goboots

import (
	"bytes"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/gabstv/go-uuid/uuid"
)

const (
	ErrNil = 0
	// TLS
	ErrTLSNil = 100
)

var (
	staticControllers []IController
	sessionDbs        map[string]ISessionDBEngine
	curSessionDb      ISessionDBEngine
	httpErrorStrings  = map[int]string{
		400: "Bad Request",
		401: "Unauthorized",
		403: "Forbidden",
		404: "Not Found",
		405: "Method Not Allowed",
		406: "Not Acceptable",
		501: "Internal Server Error",
	}
)

type Params map[string]string

func hostonly(hostport string) string {
	h, _, _ := net.SplitHostPort(hostport)
	return h
}

func trimhost(hostport string) string {
	return strings.Split(hostport, ":")[0]
}

func (a *App) getTLSRedirectURL(hostaddrtls string, uri *url.URL) (string, error) {
	if uri == nil {
		return "", errors.New("url is null")
	}
	p := a.Config.TLSRedirectPort
	var err error
	if p == "" {
		_, p, err = net.SplitHostPort(hostaddrtls)
		if err != nil {
			return "", err
		}
	}
	if uri.Scheme == "ws" {
		uri.Scheme = "wss"
	} else if uri.Scheme == "http" {
		uri.Scheme = "https"
	} else {
		// localhost may leave this empty
		uri.Scheme = "https"
	}
	if uri.Host == "" {
		if a.Config.DomainName != "" {
			uri.Host = a.Config.DomainName
		} else {
			// it's localhost
			uri.Host = "localhost"
		}
	}
	if p != "443" {
		uri.Host = uri.Host + ":" + p
	}
	return uri.String(), nil
}

func RegisterSessionStorageDriver(name string, engine ISessionDBEngine) {
	if sessionDbs == nil {
		sessionDbs = make(map[string]ISessionDBEngine, 0)
	}
	sessionDbs[name] = engine
}

func RegisterControllerGlobal(controller IController) {
	if staticControllers == nil {
		staticControllers = make([]IController, 0)
	}
	staticControllers = append(staticControllers, controller)
}

func (app *App) InitSessionStorage(driver string) error {
	if sessionDbs == nil {
		return errors.New("No session storage registered.")
	}
	if _, ok := sessionDbs[driver]; !ok {
		return errors.New("The session storage driver " + driver + " does not exist.")
	}
	curSessionDb = sessionDbs[driver]
	curSessionDb.SetApp(app)
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
		curSessionDb = &SessionDevNull{}
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
	SID     string
	Data    map[string]interface{}
	Flash   SessFlash `json:"-" bson:"-"` // never save flash
	Time    time.Time
	Updated time.Time
	r       *http.Request
	w       http.ResponseWriter
	domain  string
	path    string
}

func (s *Session) RequestURI() string {
	if s.r != nil {
		return s.r.RequestURI
	}
	return "nil"
}

type SessFlash struct {
	vals map[string]interface{}
}

func (s *SessFlash) init() {
	if s.vals == nil {
		s.vals = make(map[string]interface{})
	}
}

func (s *SessFlash) Set(key string, val interface{}) {
	s.init()
	s.vals[key] = val
}

func (s *SessFlash) Get(key string) interface{} {
	s.init()
	return s.vals[key]
}

func (s *SessFlash) Get2(key string) (interface{}, bool) {
	s.init()
	v, ok := s.vals[key]
	return v, ok
}

func (s *SessFlash) Del(key string) {
	s.init()
	delete(s.vals, key)
}

func (s *SessFlash) Clear() {
	s.vals = nil
}

func (s *SessFlash) All() map[string]interface{} {
	s.init()
	return s.vals
}

func (s *Session) DeleteData(key string) {
	if s.Data != nil {
		if _, ok := s.Data[key]; ok {
			delete(s.Data, key)
		}
	}
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
	if s.Data == nil {
		return 0, false
	}
	iface, ok := s.Data[key]
	if !ok {
		return 0, false
	}
	if vvv, ok := iface.(int); ok {
		return vvv, true
	}
	if vvv, ok := iface.(float64); ok {
		return int(vvv), true
	}
	return 0, false
}

func (s *Session) GetInt32D(key string, defaultValue int) int {
	if s.Data == nil {
		return defaultValue
	}
	iface, ok := s.Data[key]
	if !ok {
		return defaultValue
	}
	if vvv, ok := iface.(int); ok {
		return vvv
	}
	if vvv, ok := iface.(float64); ok {
		return int(vvv)
	}
	return defaultValue
}

func (s *Session) GetInt64(key string) (int64, bool) {
	if s.Data == nil {
		return 0, false
	}
	iface, ok := s.Data[key]
	if !ok {
		return 0, false
	}
	if vvv, ok := iface.(int64); ok {
		return vvv, true
	}
	if vvv, ok := iface.(float64); ok {
		return int64(vvv), true
	}
	return 0, false
}

func (s *Session) GetInt64D(key string, defaultValue int64) int64 {
	if s.Data == nil {
		return defaultValue
	}
	iface, ok := s.Data[key]
	if !ok {
		return defaultValue
	}
	if vvv, ok := iface.(int64); ok {
		return vvv
	}
	if vvv, ok := iface.(float64); ok {
		return int64(vvv)
	}
	return defaultValue
}

func (s *Session) GetString(key string) (string, bool) {
	if s.Data == nil {
		return "", false
	}
	iface, ok := s.Data[key]
	if !ok {
		return "", false
	}
	return reflect.ValueOf(iface).String(), true
}

func (s *Session) GetStringD(key string, defaultValue string) string {
	if s.Data == nil {
		return defaultValue
	}
	iface, ok := s.Data[key]
	if !ok {
		return defaultValue
	}
	return reflect.ValueOf(iface).String()
}

func (s *Session) Expire(t time.Time) {
	SetCookieAdv(s.w, "goboots_sessid", s.SID, s.path, s.domain, t, 0, false, true)
}

func (s *Session) GetExpires() time.Time {
	if s.r == nil {
		return time.Now().Add(time.Hour * 1)
	}
	cookie, err := s.r.Cookie("goboots_sessid")
	if err != nil {
		log.Println("Error @ (s *Session) GetExpires()", err)
		return time.Now()
	}
	if cookie.Expires.IsZero() {
		log.Println("goboots GetExpires IS ZERO TIME")
		return time.Now().Add(time.Hour * 3)
	}
	if cookie.Expires.Year() < 1000 {
		log.Println("goboots GetExpires IS ZERO TIME (2)")
		return time.Now().Add(time.Hour * 4)
	}
	return cookie.Expires
}

func (app *App) GetSession(w http.ResponseWriter, r *http.Request) *Session {
	if w == nil || r == nil {
		return nil
	}
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
			// get from a saved location
			msession, err = curSessionDb.GetSession(sid)
			if err == nil {
				msession.r = r
				msession.w = w
				msession.domain = app.Config.CookieDomain
				msession.path = app.Config.CookiePath
				return msession
			}
			app.Logger.Printf("SESSION ERROR :( [%s] %s\n", sid, err.Error())
			// not found, generate a new sid
		} else {
			app.Logger.Printf("COULD NOT VALIDATE :( [%s]\n", sid)
		}
	}
	// gen session
	sid = uuid.New()

	session := &Session{
		SID:     sid,
		Data:    make(map[string]interface{}),
		Time:    time.Now(),
		Updated: time.Now(),
		domain:  app.Config.CookieDomain,
		path:    app.Config.CookiePath,
	}
	err = curSessionDb.NewSession(session)
	if err != nil {
		log.Println("[FATAL] [curSessionDb.NewSession] Could not get session!", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("The server encountered an error while processing your request [ERR_CONN_NEW_SESSION]."))
		return nil
	}
	SetCookieAdv(w, "goboots_sessid", sid, session.path, session.domain, time.Now().AddDate(0, 1, 0), 0, false, true)
	session.w = w
	session.r = r
	return session
}

func (a *App) DefaultGetLang(w http.ResponseWriter, r *http.Request) string {
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
		validlangs := make([]string, 0)
		if a.I18nProvider != nil {
			validlangs = a.I18nProvider.LanguageCodes()
		}
		// break alh
		alh0 := strings.Split(alh, ",")
		for _, v := range alh0 {
			lcd := strings.Split(v, ";")[0]
			var lc string
			if len(lcd) >= 2 {
				lc = lcd[0:2]
			}
			lc = lcd
			if StringIndexOf(validlangs, lc) != -1 {
				l = lc
				break
			}
		}
		if len(l) > 0 {
			// just update the cookies
			if w != nil {
				SetCookieSimple(w, "lang", l)
			}
		} else {
			l = a.Config.DefaultLanguage
		}
		return l
	}
	return a.Config.DefaultLanguage
}

func SetUserLang(w http.ResponseWriter, r *http.Request, langcode string) {
	SetCookieSimple(w, "lang", langcode)
}

func FlushSession(s *Session) error {
	return curSessionDb.PutSession(s)
}

func DestroySession(w http.ResponseWriter, r *http.Request, s *Session) {
	curSessionDb.RemoveSession(s)
	SetCookieAdv(w, "goboots_sessid", "", s.path, s.domain, time.Now(), 1, false, true)
}

// DEPRECATED
func _validateSidString(sid string) bool {
	if len(sid) < 32 {
		return false
	}
	for _, c := range sid {
		// check if sid contains invalid chars
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || (c == '-')) {
			return false
		}
	}
	return true
}

type templateInfo struct {
	path       string
	data       *template.Template
	lastUpdate time.Time
	deps       []string
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

// SetCookieSimple sets an unsafe cookie
func SetCookieSimple(w http.ResponseWriter, key string, value string) {
	SetCookieAdv(w, key, value, "", "", time.Now().AddDate(100, 0, 0), 0, false, false)
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

func FormatPath(rawpath string) string {
	cwd, _ := os.Getwd()
	if strings.HasPrefix(rawpath, "./") {
		return filepath.Clean(cwd + rawpath[1:])
	} else if strings.HasPrefix(rawpath, "../") {
		return filepath.Clean(cwd + "/" + rawpath)
	}
	return filepath.Clean(rawpath)
}
