package main

import (
	"jstio/compass"
	"jstio/internel"
)

func main() {
	defer internel.CoreDump()

	app := compass.NewCompass()
	_ = app.Run()
}
