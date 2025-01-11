// Package main is a package declaration

package main

import (
	"fmt"
	"mtgo/cli"
	"mtgo/mods"
	"mtgo/server"
	"time"

	"mtgo/data"
)

func main() {
	startTime := time.Now()
	data.SetPrimaryDatabase()

	mods.Init()
	data.LoadBundleManifests()
	data.LoadCustomItems()

	data.SetCache()
	data.SetFlea()
	endTime := time.Now()
	fmt.Printf("Database initialized in %s\n\n", endTime.Sub(startTime))

	server.Start()
	cli.Start()
}
