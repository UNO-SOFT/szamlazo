// +build !appengine
// Above is a special build command: https://blog.golang.org/the-app-engine-sdk-and-workspaces-gopath

package main

//go:generate go generate ./...

import (
	"database/sql"
	"flag"
	"log"
	"net/http"

	"github.com/UNO-SOFT/szamlazo/backend"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	flagDB := flag.String("db", "db.sqlite", "")
	addr := flag.Arg(0)
	if addr == "" {
		addr = ":8080"
	}

	db, err := sql.Open("sqlite3", *flagDB)
	if err != nil {
		log.Fatal(err)
	}
	mux := http.DefaultServeMux
	backend.RegisterHandler(mux, db)
	log.Printf("Listening on " + addr + "...")
	http.ListenAndServe(addr, mux)
}
