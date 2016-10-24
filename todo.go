// todo.go
package main

import (
	"flag"
	"log"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	flag.Parse()
	addr := flag.Arg(0)
	if addr == "" {
		addr = ":8080"
	}
	log.Fatal(Main(addr))
}

func Main(addr string) error {
	// Create a new instance of Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	g := e.Group("/api/v1")
	g.GET("/tasks", func(c echo.Context) error { return c.JSON(200, "GET Tasks") })
	g.PUT("/tasks", func(c echo.Context) error { return c.JSON(200, "PUT Tasks") })
	g.DELETE("/tasks/:id", func(c echo.Context) error { return c.JSON(200, "DELETE Task "+c.Param("id")) })

	// Start as a web server
	log.Println("Start listening on " + addr + "...")
	return e.Start(addr)
}
