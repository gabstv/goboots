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

	// MONITOR
	go app.routineMonitoring()
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

func (app *App) routineMonitoring() {
	time.Sleep(5 * time.Minute)
	app.Logger.Println("Monitoring routine started.")
	for {
		if app.Monitor.autoClose {
			cps := app.Monitor.SlowConnectionPaths(app.Monitor.autoCloseDur)
			for _, v := range cps {
				app.Logger.Println("closing slow client", v.Path, v.Started.Format("02/01/2006 15:04"))
				app.Monitor.Cancel(v.Id)
			}
		}
		time.Sleep(time.Minute)
	}
}
