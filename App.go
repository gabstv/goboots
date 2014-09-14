package goboots

import (
	by "bytes"
	"encoding/json"
	"fmt"
	"github.com/gabstv/dson2json"
	"github.com/gabstv/i18ngo"
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
	// "public"
	AppConfigPath string
	Config        AppConfig
	Routes        []Route
	Filters       []Filter
	ByteCaches    *ByteCacheCollection
	GenericCaches *GenericCacheCollection
	Random        *rand.Rand
	// "private"
	controllerMap   map[string]IController
	templateMap     map[string]*templateInfo
	templateFuncMap template.FuncMap
	basePath        string
	entryHTTP       *appHTTP
	entryHTTPS      *appHTTPS
	didRunRoutines  bool
	mainChan        chan error
	loadedAll       bool
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
		log.Println("TLS Redirect: ", r.URL.String(), "https://"+h0[0]+urls)
		http.Redirect(w, r, "https://"+h0[0]+urls, 301)
		return
	}
	a.app.ServeHTTP(w, r)
}

func (a *appHTTPS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.app.ServeHTTP(w, r)
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("R:", r.URL.String())
	routed := app.enroute(w, r)
	//if routes didn't find anything
	if !routed {
		app.servePublicFolder(w, r)
	}
}

func (app *App) Listen() error {
	app.mainChan = make(chan error)
	app.loadAll()
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
	c.Init(a)
	a.controllerMap[name] = c
	log.Printf("controller '%s' registered", name)
}

func (a *App) GetViewTemplate(localpath string) *template.Template {
	if len(a.Config.LocalePath) > 0 {
		localpath = localpath + "_" + i18ngo.GetDefaultLanguageCode()
	}
	//pieces := strings.Split(localpath, "/")
	//path := strings.Join(append([]string{a.basePath, a.AppConfigPath, "view"}, pieces...), string(os.PathSeparator))
	//return a.templateMap[path].data
	return a.templateMap[a.Config.ViewsFolderPath+"/"+localpath].data
}

func (a *App) GetLocalizedViewTemplate(localpath string, w http.ResponseWriter, r *http.Request) *template.Template {
	localpath = localpath + "_" + GetUserLang(w, r)
	//pieces := strings.Split(localpath, "/")
	//path := strings.Join(append([]string{a.basePath, a.AppConfigPath, "view"}, pieces...), string(os.PathSeparator))
	//return a.templateMap[path].data
	//log.Println("GET-TEMPLATE" + localpath)
	//TODO: fix get/set templates!
	return a.templateMap[a.Config.ViewsFolderPath+"/"+localpath].data
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

func (a *App) loadAll() {
	if a.loadedAll {
		return
	}

	// load routes if they were added statically
	if controllers != nil {
		for _, v := range controllers {
			a.RegisterController(v)
		}
	}

	a.entryHTTP = &appHTTP{a}
	a.entryHTTPS = &appHTTPS{a}
	a.loadConfig()
	a.loadTemplates()
	a.loadedAll = true
}

func (app *App) LoadConfigFile() error {
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
		log.Println("such program very dson wow")
	}
	return json.Unmarshal(bytes, &app.Config)
}

func (app *App) loadConfig() {
	// setup Random
	src := rand.NewSource(time.Now().Unix())
	app.Random = rand.New(src)

	app.basePath, _ = os.Getwd()
	var bytes []byte
	var err error
	//
	// LOAD AppConfig.json
	//
	if len(app.Config.Name) == 0 {
		err = app.LoadConfigFile()
		__panic(err)
	}
	// set default views extension if none
	if len(app.Config.ViewsExtensions) < 1 {
		app.Config.ViewsExtensions = []string{".tpl", ".html"}
	}

	// parse Config
	app.Config.ParseEnv()
	//
	// LOAD Routes.json
	//
	if app.Routes == nil {
		app.Routes = make([]Route, 0)
	}
	if len(app.Config.RoutesConfigPath) > 0 {
		// 2014-07-22 Now accepts multiple paths, separated by semicolons
		routespaths := strings.Split(app.Config.RoutesConfigPath, ";")
		for _, rpath := range routespaths {
			rpath = strings.TrimSpace(rpath)
			fdir := FormatPath(rpath)
			bytes, err = ioutil.ReadFile(fdir)
			__panic(err)
			tempslice := make([]Route, 0)
			if xt := filepath.Ext(fdir); xt == ".dson" {
				var bf0, bf1 by.Buffer
				bf0.Write(bytes)
				err = dson2json.Convert(&bf0, &bf1)
				__panic(err)
				bytes = bf1.Bytes()
			}
			err = json.Unmarshal(bytes, &tempslice)
			__panic(err)
			for _, v := range tempslice {
				log.Println("Route `" + v.Path + "` loaded.")
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
	//
	// LOAD Localization Files (i18n)
	//
	if len(app.Config.LocalePath) > 0 {
		locPath := FormatPath(app.Config.LocalePath)
		fi, _ := os.Stat(locPath)
		if fi == nil {
			log.Fatal("Could not load i18n files at path " + locPath + "\n")
			return
		}
		if !fi.IsDir() {
			log.Fatal("Path " + locPath + " is not a directory!\n")
			return
		}
		i18ngo.LoadPoAll(locPath)
		log.Println("i18n loaded.")
	}
	//
	// Setup cache
	//
	app.ByteCaches = NewByteCacheCollection()
	app.GenericCaches = NewGenericCacheCollection()
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

func (a *App) loadTemplates() {
	if a.templateFuncMap == nil {
		a.templateFuncMap = make(template.FuncMap)
	}
	log.Println("loading template files (" + strings.Join(a.Config.ViewsExtensions, ",") + ")")
	if len(a.Config.ViewsFolderPath) == 0 {
		log.Println("ViewsFolderPath is empty")
		return
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
			bytes, _ := ioutil.ReadFile(path)
			if len(a.Config.LocalePath) < 1 {
				tplInfo := &templateInfo{
					path:       path,
					lastUpdate: time.Now(),
				}
				templ := template.New(path).Funcs(a.templateFuncMap)
				templ, err0 := templ.Parse(string(bytes))
				__panic(err0)
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
					templ, err0 := templ.Parse(LocalizeTemplate(string(bytes), lcv))
					__panic(err0)
					tplInfo.data = templ
					a.templateMap[locPName] = tplInfo
					bytesLoaded += len(bytes)
				}
			}
		}
		return nil
	}
	err := filepath.Walk(fdir, vPath)
	__panic(err)
	log.Printf("%d templates loaded (%d bytes)\n", len(a.templateMap), bytesLoaded)
}

func (app *App) servePublicFolder(w http.ResponseWriter, r *http.Request) {
	//niceurl, _ := url.QueryUnescape(r.URL.String())
	niceurl := r.URL.Path
	//TODO: place that into access log
	//log.Println("requested " + niceurl)
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

func (app *App) enroute(w http.ResponseWriter, r *http.Request) bool {
	niceurl, _ := url.QueryUnescape(r.URL.String())
	niceurl = strings.Split(niceurl, "?")[0]
	urlbits := strings.Split(niceurl, "/")[1:]
	for _, v := range app.Routes {
		if v.IsMatch(niceurl) {
			// enroute based on method
			c := app.controllerMap[v.Controller]
			if c == nil {
				//TODO: display page error instead of panic
				log.Fatalf("Controller '%s' is not registered!\n", v.Controller)
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
					log.Println("TLS Redirect: ", r.URL.String(), "https://"+h0[0]+urls)
					http.Redirect(w, r, "https://"+h0[0]+urls, 302)
					return true
				}
			}

			var inObj *In
			ul := GetUserLang(w, r)
			inObj = &In{
				r,
				w,
				urlbits,
				nil,
				&InContent{},
				&InContent{},
				app,
				c,
				ul,
				i18ngo.TL(ul, app.Config.GlobalPageTitle),
				make([]io.Closer, 0),
			}
			// close all io.Closer if present
			defer inObj.closeall()
			// run all filters
			if app.Filters != nil {
				for _, filter := range app.Filters {
					if ok := filter(inObj); !ok {
						return true
					}
				}
			}
			// run controller pre filter
			// you may want to run something before all the other methods, this is where you do it
			prec := c.PreFilter(inObj)
			if prec == nil {
				return true
			} else {
				if prec.kind != outPre {
					prec.render(inObj.W)
					return true
				}
			}
			// run main controller function
			controllerMethod := v.Method
			if len(controllerMethod) == 0 {
				controllerMethod = "Index"
			}
			rVal, rValOK := c.getMethod(controllerMethod)
			if !rValOK {
				//TODO: display page error instead of panic
				log.Fatalf("Controller '%s' does not contain a method '%s', or it's not valid.", v.Controller, v.Method)
			} else {
				// finally run it
				in := make([]reflect.Value, 2)
				in[0] = reflect.ValueOf(c)
				in[1] = reflect.ValueOf(inObj)
				out := rVal.Val.Call(in)
				o0, _ := (out[0].Interface()).(*Out)
				if o0 != nil {
					o0.render(inObj.W)
				}
				if inObj.session != nil {
					inObj.session.Flash.Clear()
				} else {
					inObj.Session().Flash.Clear()
				}
				return true
			}
			inObj.Session().Flash.Clear()
			return false
		}
	}
	return false
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
		//log.Printf("Method: %s, IN:%d, OUT:%d", name, inpt, outp)
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
			//log.Println(mt.In(1).Elem().String(), "is not", inType.String())
			continue
		}
		c.registerMethod(name, m.Func)
	}
}
