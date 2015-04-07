package goboots

import (
	"os"
	"time"
)

func (app *App) runRoutines() {
	if app.didRunRoutines {
		return
	}
	app.didRunRoutines = true

	// TEMPLATE CACHE
	// commenting for now since the template fix routine isn't done
	//go app.routineTemplateCacheMaitenance()

	// SESSIONS
	go app.routineSessionMaintenance()
}

func (app *App) routineTemplateCacheMaitenance() {
	time.Sleep(10 * time.Second)
	app.Logger.Println("Template cache maitenance routine started.")
	for {
		time.Sleep(120 * time.Second)
		app.Logger.Println("Template cache maitenance pass.")
		for k, v := range app.templateMap {
			info, err := os.Stat(v.path)
			if err == nil {
				if info.ModTime().After(v.lastUpdate) {
					app.Logger.Println(k + " IS MODIFIED")
					//TODO: re cache it!
				}
			}
		}
	}
}

func (app *App) routineSessionMaintenance() {
	time.Sleep(30 * time.Second)
	app.Logger.Println("Session maintenance routine started.")
	for {
		curSessionDb.Cleanup(time.Now().AddDate(0, 0, -15))
		time.Sleep(time.Minute * 15)
	}
}
