package main

import (
	"flag"
	"log"
	"net/http"
)

//go:generate go get -v github.com/oskca/gopherjs-vue
//go:generate go generate ./basic

func main() {
	flag.Parse()
	addr := flag.Arg(0)
	if addr == "" {
		addr = ":8080"
	}
	log.Println("Listening on " + addr)
	log.Fatal(http.ListenAndServe(addr, http.FileServer(http.Dir("."))))
}
