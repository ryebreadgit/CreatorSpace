package main

import (
	"fmt"
	"os"

	"github.com/ryebreadgit/CreatorSpace/internal/database"
	"github.com/ryebreadgit/CreatorSpace/internal/general"
	"github.com/ryebreadgit/CreatorSpace/internal/server"
	"github.com/ryebreadgit/CreatorSpace/internal/tasking"
)

func main() {
	db, err := database.GetDatabase()
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		// launch webserver Setup
		server.SetupDatabase()
	}
	settings, err := database.GetSettings(db)
	if err != nil {
		panic(err)
	}
	if settings.DatabasePath == "" {
		panic(err)
	}

	general.InitLogging()
	go tasking.InitTasking()
	server.Run()
}

func init() {
	_ = os.MkdirAll("./data/log/", os.ModePerm)
	_ = os.RemoveAll("./data/tmp/")
	_ = os.MkdirAll("./data/tmp/", os.ModePerm)
}
