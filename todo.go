// todo.go
package main

import (
	"database/sql"
	"flag"
	"log"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/UNO-SOFT/szamlazo/handlers"
)

func main() {
	flagDB := flag.String("db", "db.sqlite", "database name")
	flag.Parse()
	addr := flag.Arg(0)
	if addr == "" {
		addr = ":8080"
	}

	log.Fatal(Main(addr, *flagDB))
}

func Main(addr, dbURI string) error {
	db := initDB(dbURI)
	defer db.Close()
	migrate(db)
	// Create a new instance of Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.File("/", "public/index.html")
	e.File("/index.html", "public/index.html")
	g := e.Group("/api/v1")
	g.GET("/tasks", handlers.GetTasks(db))
	g.PUT("/tasks", handlers.PutTask(db))
	g.DELETE("/tasks/:id", handlers.DeleteTask(db))

	// Start as a web server
	log.Println("Start listening on " + addr + "...")
	return e.Start(addr)
}

func migrate(db *sql.DB) {
	qry := `CREATE TABLE IF NOT EXISTS tasks(
		id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
		name VARCHAR NOT NULL
	);`
	if _, err := db.Exec(qry); err != nil {
		panic(errors.Wrap(err, qry))
	}
}

func initDB(dbURI string) *sql.DB {
	db, err := sql.Open("sqlite3", dbURI)
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("db is nil: " + dbURI)
	}
	return db
}
