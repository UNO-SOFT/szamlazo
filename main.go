// +build !appengine
// Above is a special build command: https://blog.golang.org/the-app-engine-sdk-and-workspaces-gopath

package main

import (
	"flag"
	"net/http"
)

func main() {
	addr := flag.Arg(0)
	if addr == "" {
		addr = ":8080"
	}
	http.ListenAndServe(addr, nil)
}
