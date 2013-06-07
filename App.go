package goboots

import (
	"encoding/json"
	"fmt"
	"github.com/gabstv/i18ngo"
	"io/ioutil"
	"labix.org/v2/mgo"
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
)

var (
	DB       *mgo.Database
	APP      *App
	once_app sync.Once
	wg       *sync.WaitGroup
)

type App struct {
	// "public"
	ProjectFolder string
	Config        AppConfig
	Routes        []Route
	DbSession     *mgo.Session
	Db            *mgo.Database
	ByteCaches    *ByteCacheCollection
	SessionCache  *SessionCacheCollection
	GenericCaches *GenericCacheCollection
	Random        *rand.Rand
	// "private"
	controllerMap map[string]IController
	templateMap   map[string]*templateInfo
	basePath      string
	entryHTTP     *appHTTP
	entryHTTPS    *appHTTPS
}

type appHTTP struct {
}

type appHTTPS struct {
}

func (a *appHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if APP.Config.TLSRedirect {
		// redirect to https
		//TODO: redirect it properly
		log.Println("TODO: redirect to https")
	}
	APP.ServeHTTP(w, r)
}

func (a *appHTTPS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	APP.ServeHTTP(w, r)
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	routed := app.enroute(w, r)
	//if routes didn't find anything
	if !routed {
		app.servePublicFolder(w, r)
	}
}

func (app *App) Listen() error {
	if len(app.ProjectFolder) < 1 {
		app.ProjectFolder = "app"
	}
	onceBody := func() {
		app.loadAll()
	}
	once_app.Do(onceBody)
	defer app.DbSession.Close() // this will run when the function quits
	return http.ListenAndServe(app.Config.HostAddr, app.entryHTTP)
}

func (app *App) ListenTLS() error {
	if len(app.ProjectFolder) < 1 {
		app.ProjectFolder = "app"
	}
	onceBody := func() {
		app.loadAll()
	}
	once_app.Do(onceBody)
	defer app.DbSession.Close()
	if len(app.Config.TLSCertificatePath) < 1 || len(app.Config.TLSKeyPath) < 1 {
		// app needs key and cert to do SSL
		er2 := &AppError{
			Id:      ErrTLSNil,
			Message: "Config TLSCertificatePath or TLSKeyPath is null. Cannot listen to TLS connections.",
		}
		return er2
	}
	return http.ListenAndServeTLS(app.Config.HostAddrTLS, app.Config.TLSCertificatePath, app.Config.TLSKeyPath, app.entryHTTPS)
}

func (app *App) ListenAll() error {
	wg = &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		log.Fatal(app.Listen())
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		log.Fatal(app.ListenTLS())
		wg.Done()
	}()
	wg.Wait()
	return nil
}

func (a *App) RegisterController(c IController) {
	v := reflect.ValueOf(c)
	//pt := v.Type()
	t := v.Elem().Type()
	name := t.Name()
	a._registerControllerMethods(c)
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
	pieces := strings.Split(localpath, "/")
	path := strings.Join(append([]string{a.basePath, a.ProjectFolder, "view"}, pieces...), string(os.PathSeparator))
	return a.templateMap[path].data
}

func (a *App) GetLocalizedViewTemplate(localpath string, w http.ResponseWriter, r *http.Request) *template.Template {
	localpath = localpath + "_" + GetUserLang(w, r)
	pieces := strings.Split(localpath, "/")
	path := strings.Join(append([]string{a.basePath, a.ProjectFolder, "view"}, pieces...), string(os.PathSeparator))
	return a.templateMap[path].data
}

func (a *App) GetLayout(name string) *template.Template {
	return a.GetViewTemplate(StrConcat("layouts/", name, ".tpl"))
}

func (a *App) GetLocalizedLayout(name string, w http.ResponseWriter, r *http.Request) *template.Template {
	return a.GetLocalizedViewTemplate("layouts/"+name+".tpl", w, r)
}

func (a *App) DoHTTPError(w http.ResponseWriter, r *http.Request, err int) {
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

func (app *App) loadAll() {
	app.entryHTTP = &appHTTP{}
	app.entryHTTPS = &appHTTPS{}
	app.loadConfig()
	app.loadMongo()
	app.loadTemplates()
}

func (app *App) loadMongo() {
	var err error
	app.DbSession, err = mgo.Dial(app.Config.MongoDbs)
	__panic(err)
	// Optional -> switch the session to a monotonic behavior
	app.DbSession.SetMode(mgo.Monotonic, true)
	app.Db = app.DbSession.DB(app.Config.Database)
	DB = app.Db
	APP = app
}

func (app *App) loadConfig() {
	// setup Random
	src := rand.NewSource(time.Now().Unix())
	app.Random = rand.New(src)

	app.basePath, _ = os.Getwd()
	ps := string(os.PathSeparator)
	var dir string
	var bytes []byte
	var err error
	//
	// LOAD AppConfig.json
	//
	dir = strings.Join([]string{app.basePath, app.ProjectFolder, "config", "AppConfig.json"}, ps)
	bytes, err = ioutil.ReadFile(dir)
	__panic(err)
	err = json.Unmarshal(bytes, &app.Config)
	__panic(err)
	//
	// LOAD Routes.json
	//
	dir = strings.Join([]string{app.basePath, app.ProjectFolder, "config", "Routes.json"}, ps)
	bytes, err = ioutil.ReadFile(dir)
	__panic(err)
	err = json.Unmarshal(bytes, &app.Routes)
	//fmt.Println("trying to parse routes")
	__panic(err)
	for i := 0; i < len(app.Routes); i++ {
		if strings.Index(app.Routes[i].Path, "^") == 0 {
			app.Routes[i]._t = 2
		} else if strings.Index(app.Routes[i].Path, "*") == int(len(app.Routes[i].Path))-1 {
			//NOTE: use strings.RuneCountInString for non ASCII characters, since
			//len(string) counts the bytes (not the characters!)
			app.Routes[i]._t = 1
		}
	}
	//
	// LOAD Localization Files (i18n)
	//
	if len(app.Config.LocalePath) > 0 {
		locPath := app.Config.LocalePath
		if strings.Index(locPath, "./") == 0 {
			locPath = filepath.Clean(app.basePath + ps + app.ProjectFolder + ps + locPath[2:])
		} else if strings.Index(locPath, "../") == 0 {
			locPath = filepath.Clean(app.basePath + ps + locPath[3:])
		} else if strings.Index(locPath, ":\\\\") == 1 || strings.Index(locPath, "/") == 0 {
			locPath = filepath.Clean(locPath)
		} else {
			locPath = filepath.Clean(app.basePath + ps + app.ProjectFolder + ps + locPath)
		}
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
	app.SessionCache = NewSessionCacheCollection()
	app.GenericCaches = NewGenericCacheCollection()
}

func (a *App) loadTemplates() {
	log.Println("loading template files (.tpl)")
	a.templateMap = make(map[string]*templateInfo, 0)
	fdirs := []string{a.basePath, a.ProjectFolder, "view"}
	fdir := strings.Join(fdirs, string(os.PathSeparator))
	bytesLoaded := int(0)
	langs := i18ngo.GetLanguageCodes()
	vPath := func(path string, f os.FileInfo, err error) error {
		//fmt.Printf("parsing %s [%d]\n", path, strings.LastIndex(path, ".tpl"))
		if strings.LastIndex(path, ".tpl") == len(path)-4 {
			//fmt.Printf("[[%s]] is a template file!\n", path)
			bytes, _ := ioutil.ReadFile(path)
			if len(a.Config.LocalePath) < 1 {
				tplInfo := &templateInfo{
					path:       path,
					lastUpdate: time.Now(),
				}
				templ := template.New(path)
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
					log.Printf("templates sprinted %s\n", locPName)
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
	niceurl, _ := url.QueryUnescape(r.URL.String())
	urlparts := strings.Split(niceurl, "/")[1:]
	// after all routes are dealt with
	//TODO: check if file exists, if not, throw a 404
	fdirs := append([]string{app.basePath, app.ProjectFolder, "public"}, urlparts...)
	fdir := strings.Join(fdirs, string(os.PathSeparator))
	//
	//if _, err := os.Stat(fdir); os.IsNotExist(err) {
	//	w.WriteHeader(404)
	//	return
	//}
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
				log.Fatalf("Controller '%s' is not registered!", v.Controller)
			}
			// run pre filter
			// you may want to run something before all the other methods, this is where you do it
			prec := c.PreFilter(w, r, urlbits)
			if prec != nil {
				c.Render(w, r, prec)
				return true
			}
			// run main controller function
			var content interface{}
			if len(v.Method) == 0 {
				content = c.Run(w, r, urlbits)
			} else {
				rVal, rValOK := c._getMethod(v.Method)
				if !rValOK {
					//TODO: display page error instead of panic
					log.Fatalf("Controller '%s' does not contain a method '%s', or it's not valid.", v.Controller, v.Method)
				} else {
					// finally run it
					in := make([]reflect.Value, 4)
					in[0] = reflect.ValueOf(c)
					in[1] = reflect.ValueOf(w)
					in[2] = reflect.ValueOf(r)
					in[3] = reflect.ValueOf(urlbits)
					var out []reflect.Value
					out = rVal.Call(in)
					content = out[0].Interface()
				}
			}

			//c.SetContext(c)
			if content == nil {
				return true
			}
			c.Render(w, r, content)
			return true
		}
	}
	return false
}

func (a *App) _registerControllerMethods(c IController) {
	v := reflect.ValueOf(c)
	pt := v.Type()
	//t := v.Elem().Type()
	//name := t.Name()
	//log.Printf("_registerControllerMethods: %s", name)
	// mmap
	n := pt.NumMethod()
	for i := 0; i < n; i++ {
		m := pt.Method(i)
		name := m.Name
		// don't register known methods
		switch name {
		case "Init", "Run", "Render", "PreFilter", "ParseContent":
			continue
		case "Redirect", "PageError", "_registerMethod", "_getMethod":
			continue
		}
		mt := m.Type
		// outp must be 1 (interface{})
		outp := mt.NumOut()
		if outp != 1 {
			continue
		}
		// in amount is (c *Obj) (var1 type, var2 type)
		//               # 1 #     # 2 #      # 3 #
		inpt := mt.NumIn()
		// input must be 4 (controller + 3)
		if inpt != 4 {
			continue
		}
		//log.Printf("Method: %s, IN:%d, OUT:%d", name, inpt, outp)
		methodOut := mt.Out(0)
		if methodOut.Kind() == reflect.Interface {
			//log.Printf("MethodOut PkgPath: %s, Name:%d", methodOut.PkgPath(), methodOut.Name())
		} else {
			continue
		}
		if mt.In(1).Kind() != reflect.Interface {
			continue
		}
		if mt.In(2).Kind() != reflect.Ptr {
			continue
		}
		if mt.In(3).Kind() != reflect.Slice {
			continue
		}
		//log.Printf("Method: %s is valid!", name)
		c._registerMethod(name, m.Func)
	}
}
