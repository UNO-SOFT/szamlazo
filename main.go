package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/kyoto-framework/kyoto"
)

//go:embed static
var staticFS embed.FS

func ssatemplate(p kyoto.Page) *template.Template {
	return newtemplate("SSA")
}

func main() {
	if err := Main(); err != nil {
		log.Fatalf("ERROR: %+v", err)
	}
}

func Main() error {
	// Routes
	http.HandleFunc("/", kyoto.PageHandler(&PageIndex{}))

	// Statics
	if _, err := os.Stat("static"); err == nil {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/dist"))))
	} else {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}
	// SSA
	http.HandleFunc("/SSA/", kyoto.SSAHandler(ssatemplate))

	// Run
	port := os.Getenv("PORT")
	if port == "" {
		port = "25025"
	}
	log.Println("Listening on http://localhost:" + port)
	return http.ListenAndServe("localhost:"+port, nil)
}
