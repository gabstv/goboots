package main

import (
	"github.com/gabstv/goboots"
	_ "github.com/gabstv/goboots/session-db-drivers/sessmemory"
	"log"
	// controllers
	"github.com/gabstv/goboots/examples/basic/controller"
)

func main() {
	app := goboots.NewApp()
	app.AppConfigPath = "config/AppConfig.yaml"
	err := app.LoadConfigFile()
	if err != nil {
		panic(err)
	}
	app.InitSessionStorage("sessmemory")

	// support gzip compression
	app.Filters = []goboots.Filter{goboots.CompressFilter}
	//
	// Register controllers manually
	app.RegisterController(&controller.HomeController{})

	// Run
	err = app.Listen()
	log.Println("Could not listen:", err)
}
