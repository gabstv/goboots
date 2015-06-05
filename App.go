package goboots

import (
	by "bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gabstv/dson2json"
	"github.com/gabstv/i18ngo"
	"golang.org/x/net/websocket"
	"gopkg.in/yaml.v2"
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
	"text/template"
	"time"
)

type App struct {
	// public
	AppConfigPath string
	Config        *AppConfig
	Routes        []OldRoute
	Router        *Router
	Filters       []Filter
	ByteCaches    *ByteCacheCollection
	GenericCaches *GenericCacheCollection
	Random        *rand.Rand
	// private
	controllerMap   map[string]IController
	templateMap     map[string]*templateInfo
	templateFuncMap template.FuncMap
	basePath        string
	entryHTTP       *appHTTP
	entryHTTPS      *appHTTPS
	didRunRoutines  bool
	mainChan        chan error
	loadedAll       bool
	Monitor         appMonitor
	Logger          Logger
}

func NewApp() *App {
	app := &App{}
	app.Logger = DefaultLogger()
	app.Monitor = newMonitor(app)
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
	if a.app.Config.TLSRedirect {
		// redirect to https
		h0 := strings.Split(r.Host, ":")
		h1 := strings.Split(a.app.Config.HostAddrTLS, ":")
		h0o := h0[0]
		if len(h1) > 1 {
			if h1[1] != "443" {
				h0[0] = h0[0] + ":" + h1[1]
			}
		}
		urls := r.URL.String()
		if strings.Contains(urls, h0o) {
			urls = strings.Replace(urls, h0o, "", 1)
		}
		a.app.Logger.Println("TLS Redirect: ", r.URL.String(), "https://"+h0[0]+urls)
		http.Redirect(w, r, "https://"+h0[0]+urls, 301)
		return
	}
	a.app.ServeHTTP(w, r)
}

func (a *appHTTPS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.app.ServeHTTP(w, r)
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.Monitor.activeThreads.increment()
	defer app.Monitor.activeThreads.subtract()
	routed := app.enroute(w, r)
	//if routes didn't find anything
	if !routed {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && app.Config.GZipStatic {
			// use gzip
			if app.Config.StaticAccessLog {
				app.Logger.Println(hostonly(r.RemoteAddr), "[RGZ] ", r.URL.String())
			}
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			gzr := &gzipRespWriter{gz, w}
			app.servePublicFolder(gzr, r)
			gz.Close()
		} else {
			if app.Config.StaticAccessLog {
				app.Logger.Println(hostonly(r.RemoteAddr), "[ R ] ", r.URL.String())
			}
			app.servePublicFolder(w, r)
		}
	}
}

func (app *App) Listen() error {
	app.mainChan = make(chan error)
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
	er3 := http.ListenAndServe(app.Config.HostAddr, app.entryHTTP)
	app.mainChan <- er3
}

func (app *App) listenTLS() {
	app.loadAll()
	if len(app.Config.HostAddrTLS) < 1 {
		//TODO: error is TLS needs to be enforced (add config option)
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
	er3 := http.ListenAndServeTLS(app.Config.HostAddrTLS, app.Config.TLSCertificatePath, app.Config.TLSKeyPath, app.entryHTTPS)
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
	if len(a.Config.LocalePath) > 0 {
		localpath = localpath + "_" + i18ngo.GetDefaultLanguageCode()
	}
	if tpl, ok := a.templateMap[a.Config.ViewsFolderPath+"/"+localpath]; ok {
		return tpl.data
	}
	return nil
}

func (a *App) GetLocalizedViewTemplate(localpath string, w http.ResponseWriter, r *http.Request) *template.Template {
	localpath = localpath + "_" + GetUserLang(w, r)
	if tpl, ok := a.templateMap[a.Config.ViewsFolderPath+"/"+localpath]; ok {
		return tpl.data
	}
	return nil
}

func (a *App) GetLayout(name string) *template.Template {
	return a.GetViewTemplate(StrConcat("layouts/", name, ".tpl"))
}

func (a *App) GetLocalizedLayout(name string, w http.ResponseWriter, r *http.Request) *template.Template {
	return a.GetLocalizedViewTemplate("layouts/"+name+".tpl", w, r)
}

func (a *App) DoHTTPError(w http.ResponseWriter, r *http.Request, err int) {
	//TODO: i18n HTTP Errors
	w.WriteHeader(err)
	errorLayout := a.GetLayout("error")
	if errorLayout == nil {
		http.Error(w, httpErrorStrings[err], err)
		return
	}
	var erDesc string
	switch err {
	case 400:
		erDesc = "<strong>Bad Request</strong> - The request cannot be fulfilled due to bad syntax."
	case 401:
		erDesc = "<strong>Unauthorized</strong> - You must authenticate to view the source."
	case 403:
		erDesc = "<strong>Forbidden</strong> - You're not authorized to view the requested source."
	case 404:
		erDesc = "<strong>Not Found</strong> - The requested resource could not be found."
	case 405:
		erDesc = "<strong>Method Not Allowed</strong> - A request was made of a resource using a request method not supported by that resource."
	case 406:
		erDesc = "<strong>Not Acceptable</strong> - The requested resource is only capable of generating content not acceptable according to the Accept headers sent in the request."
	default:
		erDesc = "<a href=\"http://en.wikipedia.org/wiki/List_of_HTTP_status_codes\">The request could not be fulfilled.</a>"
	}
	if errorLayout == nil {
		fmt.Fprint(w, erDesc)
		return
	}
	page := &ErrorPageContent{
		Title:        a.Config.Name + " - " + fmt.Sprintf("%d", err),
		ErrorTitle:   fmt.Sprintf("%d", err),
		ErrorMessage: erDesc,
		Content:      " ",
	}
	errorLayout.Execute(w, page)
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
		return errors.New("loadAll " + err.Error())
	}
	err = a.loadTemplates()
	if err != nil {
		return errors.New("loadAll " + err.Error())
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
	if xt := filepath.Ext(app.AppConfigPath); xt == ".dson" {
		var bf0, bf1 by.Buffer
		bf0.Write(bytes)
		err = dson2json.Convert(&bf0, &bf1)
		if err != nil {
			return err
		}
		bytes = bf1.Bytes()
	} else if xt == ".yaml" {
		return yaml.Unmarshal(bytes, app.Config)
	}
	return json.Unmarshal(bytes, app.Config)
}

// deprecated routes method
func (app *App) loadRoutesOld() error {
	var bytes []byte
	var err error
	if app.Routes == nil {
		app.Routes = make([]OldRoute, 0)
	}
	if len(app.Config.RoutesConfigPath) > 0 {
		// 2014-07-22 Now accepts multiple paths, separated by semicolons
		routespaths := strings.Split(app.Config.RoutesConfigPath, ";")
		for _, rpath := range routespaths {
			rpath = strings.TrimSpace(rpath)
			fdir := FormatPath(rpath)
			bytes, err = ioutil.ReadFile(fdir)
			if err != nil {
				return errors.New("loadRoutesOld ioutil.ReadFile " + fdir + " " + err.Error())
			}
			tempslice := make([]OldRoute, 0)
			if xt := filepath.Ext(fdir); xt == ".dson" {
				var bf0, bf1 by.Buffer
				bf0.Write(bytes)
				err = dson2json.Convert(&bf0, &bf1)
				if err != nil {
					return errors.New("loadRoutesOld dson2json.Convert " + fdir + " " + err.Error())
				}
				bytes = bf1.Bytes()
			}
			err = json.Unmarshal(bytes, &tempslice)
			if err != nil {
				return errors.New("loadRoutesOld json.Unmarshal " + fdir + " " + err.Error())
			}
			for _, v := range tempslice {
				app.Logger.Println("Route `" + v.Path + "` loaded.")
				app.Routes = append(app.Routes, v)
			}
		}
	}

	for i := 0; i < len(app.Routes); i++ {
		if strings.Index(app.Routes[i].Path, "^") == 0 {
			app.Routes[i]._t = routeMethodRegExp
		} else if strings.HasSuffix(app.Routes[i].Path, "*") {
			app.Routes[i]._t = routeMethodRemainder
		} else if strings.HasSuffix(app.Routes[i].Path, "/?") {
			app.Routes[i]._t = routeMethodIgnoreTrail
		}
	}
	return nil
}

func (a *App) loadRoutesNew() error {
	if a.Router == nil && a.Config == nil {
		return errors.New("loadRoutesNew: cannot load routes because Congif is null")
	}
	if a.Config != nil {
		if a.Router == nil && len(a.Config.RoutesConfigPath) < 1 {
			return errors.New("loadRoutesNew: Config.RoutesConfigPath is not set")
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
		app.Config.ViewsExtensions = []string{".tpl", ".html"}
	}

	// parse Config
	app.Config.ParseEnv()
	//
	// LOAD Routes
	//
	if app.Config.OldRouteMethod {
		err = app.loadRoutesOld()
	} else {
		err = app.loadRoutesNew()
	}
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
		i18ngo.LoadPoAll(locPath)
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
	method, path, action, fixedArgs, found := parseRouteLine(line)
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

	route := NewRoute(method, path, action, fixedArgs, "memory", len(a.Router.Routes), a)
	a.Router.Routes = append(a.Router.Routes, route)
	//
	return a.Router.updateTree()
}

func (a *App) loadTemplates() error {
	if a.templateFuncMap == nil {
		a.templateFuncMap = make(template.FuncMap)
	}
	a.Logger.Println("loading template files (" + strings.Join(a.Config.ViewsExtensions, ",") + ")")
	if len(a.Config.ViewsFolderPath) == 0 {
		a.Logger.Println("ViewsFolderPath is empty")
		return nil
	}
	a.templateMap = make(map[string]*templateInfo, 0)
	fdir := FormatPath(a.Config.ViewsFolderPath)
	bytesLoaded := int(0)
	langs := i18ngo.GetLanguageCodes()
	vPath := func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(path)
		extensionIsValid := false
		for _, v := range a.Config.ViewsExtensions {
			if v == ext {
				extensionIsValid = true
			}
		}
		if extensionIsValid {
			bytes, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.New("loadTemplates ioutil.ReadFile " + path + " " + err.Error())
			}
			ldir, _ := filepath.Split(path)
			bytes, err = a.parseTemplateIncludeDeps(ldir, bytes)
			if err != nil {
				return errors.New("loadTemplates ioutil.ReadFile " + path + " a.parseTemplateIncludeDeps " + err.Error())
			}
			if len(a.Config.LocalePath) < 1 {
				tplInfo := &templateInfo{
					path:       path,
					lastUpdate: time.Now(),
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
					}
					locPName := path + "_" + lcv
					templ := template.New(locPName)
					templ, err := templ.Parse(LocalizeTemplate(string(bytes), lcv))
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
	err := filepath.Walk(fdir, vPath)
	if err != nil {
		return errors.New("loadTemplates filepath.Walk " + fdir + " " + err.Error())
	}
	a.Logger.Printf("%d templates loaded (%d bytes)\n", len(a.templateMap), bytesLoaded)
	return nil
}

func (app *App) servePublicFolder(w http.ResponseWriter, r *http.Request) {
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
		return
	}
	if info.IsDir() {
		app.DoHTTPError(w, r, 403)
		return
	}
	//
	http.ServeFile(w, r, fdir)
}

func (a *App) oldRouteMatch(niceurl string) *OldRoute {
	for _, v := range a.Routes {
		if v.IsMatch(niceurl) {
			return &v
		}
	}
	return nil
}

func (app *App) enrouteOld(niceurl string, urlbits []string, w http.ResponseWriter, r *http.Request) bool {
	v := app.oldRouteMatch(niceurl)
	if v == nil {
		return false
	}
	// enroute based on method
	c := app.controllerMap[v.Controller]
	if c == nil {
		app.Logger.Fatalf("Controller '%s' is not registered!\n", v.Controller)
		app.DoHTTPError(w, r, 501)
		return true
	}
	if v.RedirectTLS {
		if r.TLS == nil {
			// redirect to https
			h0 := strings.Split(r.Host, ":")
			h1 := strings.Split(app.Config.HostAddrTLS, ":")
			h0o := h0[0]
			if len(h1) > 1 {
				if h1[1] != "443" {
					h0[0] = h0[0] + ":" + h1[1]
				}
			}
			urls := r.URL.String()
			if strings.Contains(urls, h0o) {
				urls = strings.Replace(urls, h0o, "", 1)
			}
			app.Logger.Println("TLS Redirect: ", r.URL.String(), "https://"+h0[0]+urls)
			http.Redirect(w, r, "https://"+h0[0]+urls, 302)
			return true
		}
	}

	var inObj *In
	ul := GetUserLang(w, r)
	inObj = &In{
		r,
		w,
		nil,
		urlbits,
		Params{},
		nil,
		&InContent{},
		&InContent{},
		app,
		c,
		ul,
		i18ngo.TL(ul, app.Config.GlobalPageTitle),
		make([]io.Closer, 0),
		&InBodyWrapper{r},
		v.Controller,
		v.Method,
	}

	upgrade := r.Header.Get("Upgrade")
	if upgrade == "websocket" || upgrade == "Websocket" {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", 405)
			return true
		}
		websocket.Handler(func(ws *websocket.Conn) {
			// overwrite Read/Write timeout to something reasobnable for a websocket
			ws.SetDeadline(time.Now().Add(time.Hour * 24))
			r.Method = "WS"
			inObj.Wsock = ws
			app.handleReq(c, inObj)
		}).ServeHTTP(w, r)
		return true
	}

	return app.handleReq(c, inObj)
}

func (app *App) enroute(w http.ResponseWriter, r *http.Request) bool {
	niceurl, _ := url.QueryUnescape(r.URL.String())
	niceurl = strings.Split(niceurl, "?")[0]
	urlbits := strings.Split(niceurl, "/")[1:]
	if app.Config.OldRouteMethod {
		return app.enrouteOld(niceurl, urlbits, w, r)
	}
	if app.Router != nil {
		match := app.Router.Route(r)
		//app.Logger.Println("new router match!", match)
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
			//TODO: handle TLS redirect (on revel fork first)
			var inObj *In
			ul := GetUserLang(w, r)
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
				i18ngo.TL(ul, app.Config.GlobalPageTitle),
				make([]io.Closer, 0),
				&InBodyWrapper{r},
				match.ControllerName,
				match.MethodName,
			}

			upgrade := r.Header.Get("Upgrade")
			if upgrade == "websocket" || upgrade == "Websocket" {
				websocket.Handler(func(ws *websocket.Conn) {
					// overwrite Read/Write timeout to something reasobnable for a websocket
					ws.SetDeadline(time.Now().Add(time.Hour * 24))
					r.Method = "WS"
					inObj.Wsock = ws
					app.handleReq(c, inObj)
				}).ServeHTTP(w, r)
				return true
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
			prec.render(in.W)
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
		o0.render(in.W)
	}
	if in.session != nil {
		in.session.Flash.Clear()
	} else {
		in.Session().Flash.Clear()
	}
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
