package goboots

import (
	//"encoding/json"
	//"fmt"
	//"github.com/gabstv/i18ngo"
	//"io/ioutil"
	//"labix.org/v2/mgo"
	"log"
	//"math/rand"
	//"net/http"
	//"net/url"
	"os"
	//"path/filepath"
	//"reflect"
	//"strings"
	//"sync"
	//"text/template"
	"time"
)

func (app *App) runRoutines() {
	if app.didRunRoutines {
		return
	}
	app.didRunRoutines = true
	// commenting for now since the template fix routine isn't done
	//go func() {
	//	app.routineTemplateCacheMaitenance()
	//}()
}

func (app *App) routineTemplateCacheMaitenance() {
	time.Sleep(10 * time.Second)
	log.Println("Template cache maitenance routine started.")
	for {
		time.Sleep(120 * time.Second)
		log.Println("Template cache maitenance pass.")
		for k, v := range app.templateMap {
			info, err := os.Stat(v.path)
			if err == nil {
				if info.ModTime().After(v.lastUpdate) {
					log.Println(k + " IS MODIFIED")
					//TODO: re cache it!
				}
			}
		}
	}
}
