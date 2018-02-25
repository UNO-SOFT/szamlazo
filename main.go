package main

import (
	"log"

	"github.com/UNO-SOFT/szamlazo/actions"
)

func main() {
	app := actions.App()
	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}
