package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kyoto-framework/kyoto"
	"github.com/peterbourgon/ff/v3/ffcli"
)

func ssatemplate(p kyoto.Page) *template.Template {
	return newtemplate("SSA")
}

func main() {
	if err := Main(); err != nil {
		log.Fatalf("ERROR: %+v", err)
	}
}

func Main() error {
	fs := flag.NewFlagSet("szamlazo", flag.ContinueOnError)
	port := os.Getenv("PORT")
	if port == "" {
		port = "25025"
	}
	flagAddr := fs.String("addr", ":"+port, "HTTP address to listen on")
	app := ffcli.Command{Name: "szamlazo", FlagSet: fs,
		ShortUsage: "szamlazo [options] <-|csv>",
		Exec: func(ctx context.Context, args []string) error {
			r := io.ReadCloser(os.Stdin)
			inName := "stdin"
			if len(args) != 0 && args[0] != "" && args[0] != "-" {
				var err error
				if r, err = os.Open(args[0]); err != nil {
					return fmt.Errorf("%q: %w", args[0], err)
				}
				inName = args[0]
			}
			log.Printf("Reading csv from %q", inName)
			cr := csv.NewReader(r)
			cr.Comma = ';'
			rows, err := cr.ReadAll()
			r.Close()
			if err != nil {
				return err
			}

			// Routes
			http.HandleFunc("/", kyoto.PageHandler(&PageIndex{}))
			http.HandleFunc("/dashboard", kyoto.PageHandler(&Dashboard{rows: rows}))

			// Statics
			http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
			// SSA
			http.HandleFunc("/SSA/", kyoto.SSAHandler(ssatemplate))

			// Run
			if port == "" {
				port = "25025"
			}
			server := http.Server{Handler: RequestLoggerMiddleware(http.DefaultServeMux), Addr: *flagAddr}
			go func() {
				<-ctx.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				_ = server.Shutdown(ctx)
				cancel()
			}()
			log.Println("Listening on " + server.Addr)
			return server.ListenAndServe()
		},
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	return app.ParseAndRun(ctx, os.Args[1:])
}
