package main

import (
	"flag"
	"html/template"
	"net/http"
	"os"

	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	log "github.com/sirupsen/logrus"
	"github.com/tbellembois/gowebskel/handlers"
	"github.com/tbellembois/gowebskel/models"
)

func main() {

	var (
		err       error
		logf      *os.File
		dbname    = "./storage.db"
		datastore *models.SQLiteDataStore
	)

	// getting the program parameters
	listenPort := flag.String("port", "8080", "the port to listen")
	logfile := flag.String("logfile", "", "log to the given file")
	debug := flag.Bool("debug", false, "debug (verbose log), default is error")
	flag.Parse()

	// logging to file if logfile parameter specified
	if *logfile != "" {
		if logf, err = os.OpenFile(*logfile, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
			log.Panic(err)
		} else {
			log.SetOutput(logf)
		}
	}

	// setting the log level
	if *debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.ErrorLevel)
	}

	// database initialization
	if datastore, err = models.NewDBstore(dbname); err != nil {
		log.Panic(err)
	}
	if err = datastore.CreateDatabase(); err != nil {
		log.Panic(err)
	}

	// environment creation
	env := handlers.Env{
		DB:        datastore,
		Templates: make(map[string]*template.Template),
	}

	// templates compilation
	tplbase := []string{"static/templates/base.html",
		"static/templates/head.html",
		"static/templates/foot.html",
		"static/templates/header.html",
		"static/templates/footer.html",
		"static/templates/menu.html",
	}
	tplbasenomenu := []string{"static/templates/basenomenu.html",
		"static/templates/head.html",
		"static/templates/foot.html",
		"static/templates/header.html",
		"static/templates/footer.html",
	}

	tplhome := append(tplbase, "static/templates/home/index.html")
	tpllogin := append(tplbasenomenu, "static/templates/login/index.html")
	tplentities := append(tplbase, "static/templates/entity/index.html")
	tplentityc := append(tplbase, "static/templates/entity/create.html")
	env.Templates["home"] = template.Must(template.ParseFiles(tplhome...))
	env.Templates["login"] = template.Must(template.ParseFiles(tpllogin...))
	env.Templates["entities"] = template.Must(template.ParseFiles(tplentities...))
	env.Templates["entityc"] = template.Must(template.ParseFiles(tplentityc...))

	// router definition
	r := mux.NewRouter()
	commonChain := alice.New(env.HeadersMiddleware, env.LogingMiddleware)
	securechain := alice.New(env.HeadersMiddleware, env.LogingMiddleware, env.AuthorizeMiddleware)
	r.Handle("/", commonChain.Then(env.AppMiddleware(env.HomeHandler))).Methods("GET")
	r.Handle("/login", commonChain.Then(env.AppMiddleware(env.VLoginHandler))).Methods("GET")
	r.Handle("/get-token", commonChain.Then(env.AppMiddleware(env.GetTokenHandler))).Methods("POST")

	r.Handle("/v/test", commonChain.Then(env.AppMiddleware(env.VTestHandler))).Methods("GET")

	r.Handle("/{view:v}/{item:entities}", securechain.Then(env.AppMiddleware(env.VGetEntitiesHandler))).Methods("GET")
	r.Handle("/{view:vc}/{item:entity}", securechain.Then(env.AppMiddleware(env.VCreateEntityHandler))).Methods("GET")

	r.Handle("/{item:entities}", securechain.Then(env.AppMiddleware(env.GetEntitiesHandler))).Methods("GET")
	r.Handle("/{item:entity}/{id}", securechain.Then(env.AppMiddleware(env.GetEntityHandler))).Methods("GET")
	r.Handle("/{item:entity}", securechain.Then(env.AppMiddleware(env.CreateEntityHandler))).Methods("POST")
	r.Handle("/{item:entity}/{id}", securechain.Then(env.AppMiddleware(env.UpdateEntityHandler))).Methods("PUT")
	r.Handle("/{item:entity}/{id}", securechain.Then(env.AppMiddleware(env.DeleteEntityHandler))).Methods("DELETE")

	r.Handle("/validate/login/name", env.AppMiddleware(env.ValidateLoginNameHandler)).Methods("GET")

	cssBox := rice.MustFindBox("static/css")
	cssFileServer := http.StripPrefix("/css/", http.FileServer(cssBox.HTTPBox()))
	http.Handle("/css/", cssFileServer)

	jsBox := rice.MustFindBox("static/js")
	jsFileServer := http.StripPrefix("/js/", http.FileServer(jsBox.HTTPBox()))
	http.Handle("/js/", jsFileServer)

	imgBox := rice.MustFindBox("static/img")
	imgFileServer := http.StripPrefix("/img/", http.FileServer(imgBox.HTTPBox()))
	http.Handle("/img/", imgFileServer)

	http.Handle("/", r)

	if err = http.ListenAndServe(":"+*listenPort, nil); err != nil {
		panic(err)
	}
}
