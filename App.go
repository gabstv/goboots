package goboots

import (
	by "bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Joker/jade"
	"github.com/gabstv/dson2json"
	"github.com/gabstv/i18n"
	"github.com/gabstv/i18n/po/poutil"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/fsnotify.v1"
	"gopkg.in/yaml.v2"
)

var (
	wsupgrader = websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type App struct {
	// public
	AppConfigPath string
	Config        *AppConfig
	Router        *Router
	Filters       []Filter
	StaticFilters []Filter
	ByteCaches    *ByteCacheCollection
	GenericCaches *GenericCacheCollection
	Random        *rand.Rand
	HTTPErrorFunc func(w http.ResponseWriter, r *http.Request, err int)
	GetLangFunc   func(w http.ResponseWriter, r *http.Request) string
	ServeMux      *httprouter.Router
	// private
	controllerMap     map[string]IController
	templateMap       map[string]*templateInfo
	templateFuncMap   template.FuncMap
	basePath          string
	entryHTTP         *appHTTP
	entryHTTPS        *appHTTPS
	didRunRoutines    bool
	mainChan          chan error
	loadedAll         bool
	Monitor           appMonitor
	Logger            Logger
	AccessLogger      Logger
	TemplateProcessor TemplateProcessor
	I18nProvider      i18n.Provider
	//
	globalLoadOnce sync.Once
}

func NewApp() *App {
	app := &App{}
	app.Logger = DefaultLogger()
	app.Monitor = newMonitor(app)
	app.Config = &AppConfig{}
	app.TemplateProcessor = &defaultTemplateProcessor{}
	app.ServeMux = httprouter.New()
	app.ServeMux.NotFound = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			start := time.Now()
			urls := req.URL.String()
			staticResolve(app, req, w, start, urls)
			return
		}
	})
	return app
}

func (app *App) Logvln(v ...interface{}) {
	if app.Config.Verbose {
		app.Logger.Println(v...)
	}
}

type appHTTP struct {
	app *App
}

type appHTTPS struct {
	app *App
}

func (a *appHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.app.ServeHTTP(w, r)
}

func (a *appHTTPS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.app.ServeHTTP(w, r)
}

func staticResolve(app *App, r *http.Request, w http.ResponseWriter, start time.Time, urls string) {
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && app.Config.GZipStatic {
		// use gzip
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		gzr := &gzipRespWriter{gz, w}
		staticStatus := app.servePublicFolder(gzr, r)
		gz.Close()
		if app.Config.StaticAccessLog {
			addr := r.Header.Get("X-Real-IP")
			if addr == "" {
				addr = r.Header.Get("X-Forwarded-For")
				if addr == "" {
					addr = r.RemoteAddr
				}
			}
			if app.AccessLogger == nil {
				app.Logger.Println(addr, "[RGZ] ", urls, staticStatus, time.Since(start))
			} else {
				app.AccessLogger.Println(addr, "[RGZ] ", urls, staticStatus, time.Since(start))
			}
		}
	} else {
		staticStatus := app.servePublicFolder(w, r)
		if app.Config.StaticAccessLog {
			addr := r.Header.Get("X-Real-IP")
			if addr == "" {
				addr = r.Header.Get("X-Forwarded-For")
				if addr == "" {
					addr = r.RemoteAddr
				}
			}
			if app.AccessLogger == nil {
				app.Logger.Println(addr, "[ R ] ", urls, staticStatus, time.Since(start))
			} else {
				app.AccessLogger.Println(addr, "[ R ] ", urls, staticStatus, time.Since(start))
			}
		}
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	urls := r.URL.String()
	//
	//
	reqid := app.Monitor.openConnectionPaths.Add(r)
	defer app.Monitor.openConnectionPaths.Remove(reqid)
	//
	routed := app.enroute(w, r)
	//if routes didn't find anything
	if !routed {
		if app.ServeMux != nil {
			app.ServeMux.ServeHTTP(w, r)
			return
		}
		staticResolve(app, r, w, start, urls)
		return
	}
	if app.Config.DynamicAccessLog {
		addr := r.Header.Get("X-Real-IP")
		if addr == "" {
			addr = r.Header.Get("X-Forwarded-For")
			if addr == "" {
				addr = r.RemoteAddr
			}
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || !app.Config.GZipDynamic {
			if app.AccessLogger == nil {
				app.Logger.Println(addr, "{ R } ", r.RequestURI, time.Since(start))
			} else {
				app.AccessLogger.Println(addr, "{ R } ", r.RequestURI, time.Since(start))
			}
		} else {
			if app.AccessLogger == nil {
				app.Logger.Println(addr, "{RGZ} ", r.RequestURI, time.Since(start))
			} else {
				app.AccessLogger.Println(addr, "{RGZ} ", r.RequestURI, time.Since(start))
			}
		}
	}
}

func (app *App) Listen() error {
	app.mainChan = make(chan error, 10)
	e0 := app.loadAll()
	if e0 != nil {
		app.mainChan <- e0
	}
	go func() {
		app.listen()
	}()
	go func() {
		app.listenTLS()
	}()
	app.runRoutines()

	defer CloseSessionStorage()

	var err error
	err = <-app.mainChan
	return err
}

func (app *App) listen() {
	app.loadAll()
	if len(app.Config.HostAddr) < 1 {
		return
	}
	var er3 error
	if app.Config.GracefulRestart {
		er3 = listenAndServeGracefully(app.Config.HostAddr, app.entryHTTP)
	} else {
		er3 = listenAndServe(app.Config.HostAddr, app.entryHTTP)
	}
	app.mainChan <- er3
}

func (app *App) listenTLS() {
	app.loadAll()
	if app.Config.TLSAutocert && app.Config.TLSAutocertWhitelist != nil && len(app.Config.TLSAutocertWhitelist) > 0 {
		var er4 error
		if app.Config.GracefulRestart {
			er4 = autocertServerTLSGracefully(app.Config.HostAddrTLS, app.Config.TLSAutocertWhitelist, app)
		} else {
			er4 = autocertServerTLS(app.Config.HostAddrTLS, app.Config.TLSAutocertWhitelist, app)
		}
		app.mainChan <- er4
		return
	}
	// if the raw cert and key are present, put them in a temp file
	if len(app.Config.RawTLSCert) > 128 && len(app.Config.RawTLSKey) > 128 {
		tkeyfile, err := ioutil.TempFile("", "gb_tempkey_")
		if err != nil {
			app.mainChan <- err
			return
		}
		app.Config.TLSKeyPath = tkeyfile.Name()
		tkeyfile.Write([]byte(app.Config.RawTLSKey))
		tkeyfile.Close()
		defer func(fn string) {
			os.Remove(fn)
		}(app.Config.TLSKeyPath)
		//
		tcertfile, err := ioutil.TempFile("", "gb_tempcert_")
		if err != nil {
			app.mainChan <- err
			return
		}
		app.Config.TLSCertificatePath = tcertfile.Name()
		tcertfile.Write([]byte(app.Config.RawTLSCert))
		tcertfile.Close()
		defer func(fn string) {
			os.Remove(fn)
		}(app.Config.TLSCertificatePath)
		app.Logger.Println("Temp TLS key at", app.Config.TLSKeyPath)
		app.Logger.Println("Temp TLS cert at", app.Config.TLSCertificatePath)
	}
	if len(app.Config.HostAddrTLS) < 1 || (len(app.Config.TLSCertificatePath) < 1 && len(app.Config.TLSKeyPath) < 1) {
		//TODO: error if TLS needs to be enforced (add config option)
		return
	}
	if len(app.Config.TLSCertificatePath) < 1 || len(app.Config.TLSKeyPath) < 1 {
		// app needs key and cert to do SSL
		er2 := &AppError{
			Id:      ErrTLSNil,
			Message: "Config TLSCertificatePath or TLSKeyPath is null. Cannot listen to TLS connections.",
		}
		app.mainChan <- er2
		return
	}
	var er3 error
	if app.Config.GracefulRestart {
		er3 = listenAndServeTLSGracefully(app.Config.HostAddrTLS, app.Config.TLSCertificatePath, app.Config.TLSKeyPath, app.entryHTTPS)
	} else {
		er3 = listenAndServeTLS(app.Config.HostAddrTLS, app.Config.TLSCertificatePath, app.Config.TLSKeyPath, app.entryHTTPS)
	}
	app.mainChan <- er3
}

func (a *App) RegisterController(c IController) {
	c.SetApp(a)
	v := reflect.ValueOf(c)
	//pt := v.Type()
	t := v.Elem().Type()
	name := t.Name()
	a.registerControllerMethods(c)
	if a.controllerMap == nil {
		a.controllerMap = make(map[string]IController, 0)
	}
	//
	// Register methods

	//
	c.Init()
	a.controllerMap[name] = c
	a.Logger.Printf("controller '%s' registered", name)
}

func (a *App) GetViewTemplate(localpath string) *template.Template {
	if a.I18nProvider != nil {
		localpath = localpath + "_" + a.Config.DefaultLanguage
	}
	if tpl, ok := a.templateMap[filepath.Join(a.Config.ViewsFolderPath, localpath)]; ok {
		return tpl.data
	}
	a.Logger.Printf("GetViewTemplate('%v') '%v' not found!\n", localpath, filepath.Join(a.Config.ViewsFolderPath, localpath))
	return nil
}

func (a *App) GetLocalizedViewTemplate(localpath string, w http.ResponseWriter, r *http.Request) *template.Template {
	lfn := a.GetLangFunc
	if lfn == nil {
		lfn = a.DefaultGetLang
	}
	localpath = localpath + "_" + lfn(w, r)
	if tpl, ok := a.templateMap[a.Config.ViewsFolderPath+"/"+localpath]; ok {
		return tpl.data
	}
	return nil
}

func (a *App) GetLayout(name string) *template.Template {
	ext := filepath.Ext(name)
	if len(ext) < 1 {
		ext = ".tpl"
	} else {
		name = name[:(len(name) - len(ext))]
	}
	return a.GetViewTemplate(StrConcat("layouts/", name, ext))
}

func (a *App) GetLocalizedLayout(name string, w http.ResponseWriter, r *http.Request) *template.Template {
	ext := filepath.Ext(name)
	if len(ext) < 1 {
		ext = ".tpl"
	} else {
		name = name[:(len(name) - len(ext))]
	}
	return a.GetLocalizedViewTemplate("layouts/"+name+ext, w, r)
}

func (a *App) DoHTTPError(w http.ResponseWriter, r *http.Request, err int) {
	// try the custom error function
	if a.HTTPErrorFunc != nil {
		a.HTTPErrorFunc(w, r, err)
		return
	}
	// try layouts
	if lay := a.GetLocalizedLayout(fmt.Sprint(err), w, r); lay != nil {
		w.WriteHeader(err)
		lay.Execute(w, nil)
		return
	}
	if lay := a.GetLayout(fmt.Sprint(err)); lay != nil {
		w.WriteHeader(err)
		lay.Execute(w, nil)
		return
	}
	http.Error(w, httpErrorStrings[err], err)
	return
}

func (a *App) BenchLoadAll() error {
	return a.loadAll()
}

func (a *App) loadAll() error {
	if a.loadedAll {
		return nil
	}

	a.entryHTTP = &appHTTP{a}
	a.entryHTTPS = &appHTTPS{a}
	err := a.loadConfig()
	if err != nil {
		return errors.New("loadAllConfig " + err.Error())
	}
	err = a.loadTemplates()
	if err != nil {
		return errors.New("loadAllTemplates " + err.Error())
	}

	a.loadedAll = true
	return nil
}

func (app *App) LoadConfigFile() error {
	if app.Logger == nil {
		app.Logger = DefaultLogger()
	}
	if len(app.AppConfigPath) < 1 {
		// try to get appconfig path from env
		app.AppConfigPath = os.Getenv("APPCONFIGPATH")
		if len(app.AppConfigPath) < 1 {
			app.AppConfigPath = os.Getenv("APPCONFIG")
			if len(app.AppConfigPath) < 1 {
				// try to get $cwd/AppConfig.json
				_, err := os.Stat("AppConfig.json")
				if os.IsNotExist(err) {
					return err
				}
				app.AppConfigPath = "AppConfig.json"
			}
		}
	}
	dir := FormatPath(app.AppConfigPath)
	bytes, err := ioutil.ReadFile(dir)
	if err != nil {
		return err
	}
	if app.Config == nil {
		app.Config = &AppConfig{}
	}
	if xt := filepath.Ext(app.AppConfigPath); xt == ".dson" {
		var bf0, bf1 by.Buffer
		bf0.Write(bytes)
		err = dson2json.Convert(&bf0, &bf1)
		if err != nil {
			return err
		}
		bytes = bf1.Bytes()
	} else if xt == ".yaml" || xt == ".yml" {
		return yaml.Unmarshal(bytes, app.Config)
	}
	return json.Unmarshal(bytes, app.Config)
}

func (a *App) loadRoutesNew() error {
	if a.Router == nil && a.Config == nil {
		return errors.New("loadRoutesNew: cannot load routes because Config is null")
	}
	if a.Config != nil {
		if a.Router == nil && len(a.Config.RoutesConfigPath) < 1 {
			a.Logger.Println("no routes to load (Config.RoutesConfigPath is empty)")
			//a.Router = NewRouter(a, "")
			//return errors.New("loadRoutesNew: Config.RoutesConfigPath is not set")
		}
	}
	if a.Router != nil {
		a.Logger.Println("SKIPPING loadRoutesNew because the routes were added MANUALLY")
		return nil
	}
	a.Router = NewRouter(a, a.Config.RoutesConfigPath)
	err := a.Router.Refresh()
	if err != nil {
		return errors.New("loadRoutesNew a.Router.Refresh() " + err.Error())
	}
	a.Logger.Printf("%v routes loaded.\n", len(a.Router.Routes))
	return nil
}

func (app *App) loadConfig() error {
	// setup Random
	src := rand.NewSource(time.Now().Unix())
	app.Random = rand.New(src)

	app.basePath, _ = os.Getwd()
	var err error
	//
	// LOAD AppConfig.json
	//
	if app.Config == nil {
		err = app.LoadConfigFile()
		if err != nil {
			return errors.New("loadConfig app.LoadConfigFile() " + err.Error())
		}
	}
	// set default views extension if none
	if len(app.Config.ViewsExtensions) < 1 {
		app.Config.ViewsExtensions = []string{".tpl", ".html", ".jade", ".pug"}
	}

	// parse Config
	app.Config.ParseEnv()

	app.globalLoadOnce.Do(func() {
		// load globally registered controllers
		if staticControllers != nil {
			for _, v := range staticControllers {
				app.RegisterController(v)
			}
		}
	})
	//
	// LOAD Routes
	//
	err = app.loadRoutesNew()
	if err != nil {
		return errors.New("loadRoutes " + err.Error())
	}
	//
	// LOAD Localization Files (i18n)
	//
	if len(app.Config.LocalePath) > 0 {
		locPath := FormatPath(app.Config.LocalePath)
		fi, _ := os.Stat(locPath)
		if fi == nil {
			return errors.New("could not load i18n files at path " + locPath)
		}
		if !fi.IsDir() {
			return errors.New("path " + locPath + " is not a directory")
		}
		if app.I18nProvider == nil {
			app.I18nProvider, err = poutil.LoadAll(locPath, app.Config.DefaultLanguage)
			if err != nil {
				return err
			}
		}
		app.Logger.Println("i18n loaded.")
	}
	//
	// Setup cache
	//
	app.ByteCaches = NewByteCacheCollection()
	app.GenericCaches = NewGenericCacheCollection()
	return nil
}

func (a *App) AddTemplateFuncMap(tfmap map[string]interface{}) {
	if a.templateFuncMap == nil {
		a.templateFuncMap = make(template.FuncMap)
	}
	for k, v := range tfmap {
		a.templateFuncMap[k] = v
	}
}

func (a *App) AddTemplateFunc(key string, tfunc interface{}) {
	if a.templateFuncMap == nil {
		a.templateFuncMap = make(template.FuncMap)
	}
	a.templateFuncMap[key] = tfunc
}

func (a *App) AddRouteLine(line string) error {
	joinedPath := ""
	if a.Router == nil {
		a.Router = NewRouter(a, "memory")
		a.Router.Routes = make([]*Route, 0)
	}
	//
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] == '#' {
		return nil
	}
	method, path, action, fixedArgs, tls, found, _, _ := routeParseLine(line)
	if !found {
		return nil
	}
	// this will avoid accidental double forward slashes in a route.
	// this also avoids pathtree freaking out and causing a runtime panic
	// because of the double slashes
	if strings.HasSuffix(joinedPath, "/") && strings.HasPrefix(path, "/") {
		joinedPath = joinedPath[0 : len(joinedPath)-1]
	}
	path = strings.Join([]string{joinedPath, path}, "")

	route := NewRoute(method, path, action, fixedArgs, "memory", len(a.Router.Routes), tls, a)
	a.Router.Routes = append(a.Router.Routes, route)
	//
	return a.Router.updateTree()
}

func (a *App) loadTemplates() error {
	if a.templateFuncMap == nil {
		a.templateFuncMap = make(template.FuncMap)
	}
	if a.TemplateProcessor == nil {
		if a.Logger != nil {
			a.Logger.Println("loadTemplates() TemplateProcessor was nil")
		}
		a.TemplateProcessor = &defaultTemplateProcessor{}
	}
	//
	// set default views extension if none
	if len(a.Config.ViewsExtensions) < 1 {
		a.Config.ViewsExtensions = []string{".tpl", ".html", ".jade", ".pug"}
	}
	//
	a.Logger.Println("loading template files (" + strings.Join(a.Config.ViewsExtensions, ",") + ")")
	if len(a.Config.ViewsFolderPath) == 0 {
		a.Logger.Println("ViewsFolderPath is empty")
		//return nil
	}
	//
	var fswatcher *fsnotify.Watcher
	var fserr error
	//
	if a.Config.WatchViewsFolder {
		fswatcher, fserr = fsnotify.NewWatcher()
		if fserr != nil {
			a.Logger.Printf("fsnotify error: %v\n", fserr.Error())
		}
	}
	//
	a.templateMap = make(map[string]*templateInfo, 0)
	fdir := FormatPath(a.Config.ViewsFolderPath)
	bytesLoaded := int(0)
	langs := make([]string, 0)
	if a.I18nProvider != nil {
		langs = a.I18nProvider.LanguageCodes()
	}
	vPath := func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			if fswatcher != nil {
				fswatcher.Add(path)
				a.Logger.Println("FSWATCH (dir)", path)
			}
			return nil
		}
		ext := filepath.Ext(path)
		extensionIsValid := false
		for _, v := range a.Config.ViewsExtensions {
			if v == ext {
				extensionIsValid = true
			}
		}
		if extensionIsValid {
			if fswatcher != nil {
				fswatcher.Add(path)
				a.Logger.Println("FSWATCH (file)", path)
			}
			bytes, err := a.TemplateProcessor.ReadFile(path)
			if err != nil {
				return errors.New("loadTemplates TemplateProcessor.ReadFile " + path + " " + err.Error())
			}
			ldir, _ := filepath.Split(path)
			depList := make([]string, 0, 64)
			bytes, err = a.parseTemplateIncludeDeps(ldir, bytes, &depList)
			if err != nil {
				return errors.New("loadTemplates TemplateProcessor.ReadFile " + path + " a.parseTemplateIncludeDeps " + err.Error())
			}
			if len(a.Config.LocalePath) < 1 {
				tplInfo := &templateInfo{
					path:       path,
					lastUpdate: time.Now(),
					deps:       depList,
				}
				if ext == ".pug" || ext == ".jade" {
					// it's a jade
					jadef, err := jade.Parse(path, bytes)
					if err != nil {
						return deepErrorStr("[0] error parsing pug template file " + path + ": " + err.Error())
					}
					jadef = tempJadeFix(jadef, "{{", "}}")
					bytes = []byte(jadef)
				}
				templ := template.New(path).Funcs(a.templateFuncMap)
				templ, err := templ.Parse(string(bytes))
				if err != nil {
					return errors.New("loadTemplates template.New " + path + " templ.Parse " + err.Error())
				}
				tplInfo.data = templ
				a.templateMap[path] = tplInfo
				bytesLoaded += len(bytes)
			} else {
				for _, lcv := range langs {
					tplInfo := &templateInfo{
						path:       path,
						lastUpdate: time.Now(),
						deps:       depList,
					}
					locPName := path + "_" + lcv
					templ := template.New(locPName)
					if ext == ".pug" || ext == ".jade" {
						// it's a jade
						jadef, err := jade.Parse(locPName, bytes)
						if err != nil {
							return deepErrorStr("[1] error parsing pug template file " + locPName + ": " + err.Error())
						}
						jadef = tempJadeFix(jadef, "{{", "}}")
						bytes = []byte(jadef)
					}
					templ, err := templ.Parse(LocalizeTemplate(string(bytes), lcv, a.I18nProvider))
					if err != nil {
						return errors.New("loadTemplates " + locPName + " templ.Parse LocalizeTemplate " + err.Error())
					}
					tplInfo.data = templ
					a.templateMap[locPName] = tplInfo
					bytesLoaded += len(bytes)
				}
			}
		}
		return nil
	}
	err := a.TemplateProcessor.Walk(fdir, vPath)
	if err != nil {
		return errors.New("loadTemplates TemplateProcessor.Walk " + fdir + " " + err.Error())
	}
	if fswatcher != nil {
		a.Logger.Println("will start fswatcher")
		go func() {
			defer fswatcher.Close()
			for {
				select {
				case evt := <-fswatcher.Events:
					path := evt.Name
					if evt.Op == fsnotify.Create || evt.Op == fsnotify.Write {
						ext := filepath.Ext(path)
						for _, v := range a.Config.ViewsExtensions {
							if v == ext {
								//
								bytes, err := a.TemplateProcessor.ReadFile(path)
								if err != nil {
									a.Logger.Println("FSWATCH loadTemplates TemplateProcessor.ReadFile ", path, err.Error())
									break
								}
								ldir, _ := filepath.Split(path)
								depList := make([]string, 0, 64)
								bytes, err = a.parseTemplateIncludeDeps(ldir, bytes, &depList)
								if err != nil {
									a.Logger.Println("FSWATCH loadTemplates TemplateProcessor.parseTemplateIncludeDeps ", path, err.Error())
									break
								}
								if len(a.Config.LocalePath) < 1 {
									tplInfo := &templateInfo{
										path:       path,
										lastUpdate: time.Now(),
										deps:       depList,
									}
									if ext == ".pug" || ext == ".jade" {
										// it's a jade
										jadef, err := jade.Parse(path, bytes)
										if err != nil {
											a.Logger.Println("FSWATCH error parsing pug template file", path, err.Error())
											break
										}
										jadef = tempJadeFix(jadef, "{{", "}}")
										bytes = []byte(jadef)
									}
									templ := template.New(path).Funcs(a.templateFuncMap)
									templ, err := templ.Parse(string(bytes))
									if err != nil {
										a.Logger.Println("FSWATCH loadTemplates template.New", path, "templ.Parse", err.Error())
										break
									}
									tplInfo.data = templ
									a.templateMap[path] = tplInfo
									a.Logger.Println("FSWATCH reloaded template", path)
									// update deps
									{
										for _, zv := range a.templateMap {
											for _, d := range zv.deps {
												if d == path {
													evt := fsnotify.Event{
														Name: zv.path,
														Op:   fsnotify.Write,
													}
													go func() {
														fswatcher.Events <- evt
													}()
												}
											}
										}
									}
								} else {
									for _, lcv := range langs {
										tplInfo := &templateInfo{
											path:       path,
											lastUpdate: time.Now(),
											deps:       depList,
										}
										locPName := path + "_" + lcv
										templ := template.New(locPName)
										if ext == ".pug" || ext == ".jade" {
											// it's a jade
											jadef, err := jade.Parse(locPName, bytes)
											if err != nil {
												a.Logger.Println("FSWATCH error parsing pug template file", locPName, ":", err.Error())
												break
											}
											jadef = tempJadeFix(jadef, "{{", "}}")
											bytes = []byte(jadef)
										}
										templ, err := templ.Parse(LocalizeTemplate(string(bytes), lcv, a.I18nProvider))
										if err != nil {
											a.Logger.Println("FSWATCH loadTemplates", locPName, "templ.Parse LocalizeTemplate", err.Error())
											break
										}
										tplInfo.data = templ
										a.templateMap[locPName] = tplInfo
										a.Logger.Println("FSWATCH reloaded template", locPName)
										// update deps
										{
											for _, zv := range a.templateMap {
												for _, d := range zv.deps {
													if d == path {
														evt := fsnotify.Event{
															Name: zv.path,
															Op:   fsnotify.Write,
														}
														go func() {
															fswatcher.Events <- evt
														}()
													}
												}
											}
										}
									}
								}
								//
								break
							}
						}
					} else if evt.Op == fsnotify.Remove {
						//TODO: remove template
					}
				}
			}
		}()
	}
	a.Logger.Printf("%d templates loaded (%d bytes)\n", len(a.templateMap), bytesLoaded)

	return nil
}

func (app *App) servePublicFolder(w http.ResponseWriter, r *http.Request) int {
	//niceurl, _ := url.QueryUnescape(r.URL.String())
	niceurl := r.URL.Path
	//TODO: place that into access log
	//app.Logger.Println("requested " + niceurl)
	// after all routes are dealt with
	//TODO: have an option to have these files in memory
	fdir := FormatPath(app.Config.PublicFolderPath + "/" + niceurl)
	//
	info, err := os.Stat(fdir)
	if os.IsNotExist(err) {
		app.DoHTTPError(w, r, 404)
		return 404
	}

	// run all static filters
	if app.StaticFilters != nil {

		urlbits := strings.Split(niceurl, "/")[1:]
		//
		var inObj *In
		lfn := app.GetLangFunc
		if lfn == nil {
			lfn = app.DefaultGetLang
		}
		ul := lfn(w, r)
		pageTitle := app.Config.GlobalPageTitle
		if app.I18nProvider != nil {
			if ll := app.I18nProvider.L(ul); ll != nil {
				pageTitle = ll.T(pageTitle)
			}
		}
		inObj = &In{
			r,
			w,
			nil,
			urlbits,
			nil,
			nil,
			&InContent{},
			&InContent{},
			app,
			nil,
			ul,
			pageTitle,
			make([]io.Closer, 0),
			&InBodyWrapper{r},
			"servePublicFolder",
			"",
			false,
			nil,
			nil,
			sync.Mutex{},
		}

		for _, filter := range app.StaticFilters {
			if ok := filter(inObj); !ok {
				return 200
			}
		}
	}

	if info != nil && info.IsDir() {
		// resolve static index files
		if app.Config.StaticIndexFiles != nil {
			for _, v := range app.Config.StaticIndexFiles {
				j := filepath.Join(fdir, v)
				if _, err2 := os.Stat(j); err2 == nil {
					http.ServeFile(w, r, j)
					return 200
				}
			}
		}
		app.DoHTTPError(w, r, 403)
		return 403
	}
	//
	return app.ServeFile(w, r, fdir)
}

func (app *App) enroute(w http.ResponseWriter, r *http.Request) bool {
	niceurl, _ := url.QueryUnescape(r.URL.String())
	niceurl = strings.Split(niceurl, "?")[0]
	urlbits := strings.Split(niceurl, "/")[1:]
	if app.Router != nil {
		upgrade := r.Header.Get("Upgrade")
		if r.Method == http.MethodGet && (upgrade == "websocket" || upgrade == "Websocket") {
			r.Method = "WS"
		}
		match := app.Router.Route(r)
		if match != nil {
			if match.Action == "404" {
				//TODO: clean flash
				app.DoHTTPError(w, r, 404)
				return true
			}
			// enroute based on method
			c := app.controllerMap[match.ControllerName]
			if c == nil {
				// Internal Server Error
				app.Logger.Fatalf("Controller '%s' is not registered!\n", match.ControllerName)
				app.DoHTTPError(w, r, 501)
				return true
			}
			if match.Params == nil {
				match.Params = make(Params)
			}
			//handle TLS only
			if app.Config.TLSRedirect || match.TLSOnly {
				if (r.URL.Scheme == "http" || r.URL.Scheme == "ws") || (r.URL.Scheme == "" && r.TLS == nil) {
					if hh := r.Header.Get("X-Forwarded-Proto"); hh != "https" && hh != "wss" { // don't redirect if proxy is already secure
						app.Logger.Println("X-Forwarded-Proto: ", hh)
						redir, err := app.getTLSRedirectURL(app.Config.HostAddrTLS, r.URL)
						if err != nil {
							http.Error(w, "Internal Server Error - https redirect - "+err.Error(), 501)
							return true
						}
						app.Logger.Println("TLS Redirect: ", redir)
						http.Redirect(w, r, redir, 302)
						return true
					}
				}
			}
			//
			var inObj *In
			lfn := app.GetLangFunc
			if lfn == nil {
				lfn = app.DefaultGetLang
			}
			ul := lfn(w, r)
			pageTitle := app.Config.GlobalPageTitle
			if app.I18nProvider != nil {
				if ll := app.I18nProvider.L(ul); ll != nil {
					pageTitle = ll.T(pageTitle)
				}
			}
			inObj = &In{
				r,
				w,
				nil,
				urlbits,
				match.Params,
				nil,
				&InContent{},
				&InContent{},
				app,
				c,
				ul,
				pageTitle,
				make([]io.Closer, 0),
				&InBodyWrapper{r},
				match.ControllerName,
				match.MethodName,
				false,
				nil,
				nil,
				sync.Mutex{},
			}

			if upgrade == "websocket" || upgrade == "Websocket" {
				if r.Method != "WS" {
					app.Logger.Println("websocket method not allowed " + r.URL.String())
					http.Error(w, "Method not allowed", 405)
					return true
				}
				//app.Logger.Println("websocket will upgrade")
				inObj.hijacked = true
				r.Method = http.MethodGet
				conn, err := wsupgrader.Upgrade(w, r, nil)
				if err != nil {
					app.Logger.Println("websocket upgrade error", err)
					//http.Error(w, err.Error(), 501)
					return true
				}
				//r.Method = "WS"
				inObj.Wsock = conn
			}

			return app.handleReq(c, inObj)
		}
	}
	return false
}

func (app *App) handleReq(c IController, in *In) bool {
	defer in.closeall()
	// run all filters
	if app.Filters != nil {
		for _, filter := range app.Filters {
			if ok := filter(in); !ok {
				return true
			}
		}
	}
	// run controller pre filter
	// you may want to run something before all the other methods, this is where you do it
	prec := c.PreFilter(in)
	if prec == nil {
		return true
	} else {
		if prec.kind != outPre {
			prec.Render(in.W)
			return true
		}
	}

	// run main controller function
	controllerMethod := in.methodName
	if len(controllerMethod) == 0 {
		controllerMethod = "Index"
	}
	if c == nil {
		log.Println("[FATAL] Controller '%s %s' is null.", in.controllerName, in.methodName)
		app.DoHTTPError(in.W, in.R, 501)
		return true
	}
	rVal, rValOK := c.getMethod(controllerMethod)
	if !rValOK {
		log.Println("[FATAL] Controller '%s' does not contain a method '%s', or it's not valid.", in.controllerName, in.methodName)
		app.DoHTTPError(in.W, in.R, 501)
		return true
	}
	// finally run it
	inz := make([]reflect.Value, 2)
	inz[0] = reflect.ValueOf(c)
	inz[1] = reflect.ValueOf(in)
	out := rVal.Val.Call(inz)
	o0, _ := (out[0].Interface()).(*Out)
	if o0 != nil {
		o0.Render(in.W)
	}
	if in.session != nil {
		in.session.Flash.Clear()
	} // else {
	//	in.Session().Flash.Clear()
	//}
	return true
}

func (a *App) registerControllerMethods(c IController) {
	v := reflect.ValueOf(c)
	pt := v.Type()
	inType := reflect.TypeOf((*In)(nil)).Elem()
	outType := reflect.TypeOf((*Out)(nil)).Elem()
	// mmap
	n := pt.NumMethod()
	for i := 0; i < n; i++ {
		m := pt.Method(i)
		name := m.Name
		// don't register known methods
		switch name {
		case "Init", "PreFilter":
			continue
		case "registerMethod", "getMethod":
			continue
		}
		mt := m.Type
		// outp must be 1 (interface{})
		outp := mt.NumOut()
		if outp != 1 {
			continue
		}
		//a.Logger.Printf("Method: %s, IN:%d, OUT:%d", name, inpt, outp)
		methodOut := mt.Out(0)
		if methodOut.Kind() != reflect.Ptr {
			continue
		}
		if methodOut.Elem() != outType {
			continue
		}
		// in amount is (c *Obj) (var1 type, var2 type)
		//               # 1 #     # 2 #      # 3 #
		inpt := mt.NumIn()
		// input must be 2 (controller + 1)
		if inpt != 2 {
			continue
		}
		if mt.In(1).Kind() != reflect.Ptr {
			continue
		}
		//
		if mt.In(1).Elem() != inType {
			//a.Logger.Println(mt.In(1).Elem().String(), "is not", inType.String())
			continue
		}
		c.registerMethod(name, m.Func)
	}
}
