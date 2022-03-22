package main

import (
	"bytes"
	"context"
	_ "embed"
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
	"golang.org/x/text/encoding/charmap"
)

func ssatemplate(p kyoto.Page) *template.Template {
	return newtemplate("SSA")
}

func main() {
	if err := Main(); err != nil {
		log.Fatalf("ERROR: %+v", err)
	}
}

var dashboardTableRows []DashboardTableRow

//go:embed testdata/nyilvtart-jan.csv
var nyilvtartCSV []byte

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
			if len(args) == 0 || args[0] == "" {
				r = struct {
					io.Reader
					io.Closer
				}{bytes.NewReader(nyilvtartCSV), io.NopCloser(nil)}
			} else if len(args) != 0 && args[0] != "-" {
				var err error
				if r, err = os.Open(args[0]); err != nil {
					return fmt.Errorf("%q: %w", args[0], err)
				}
				inName = args[0]
			}
			defer r.Close()
			log.Printf("Reading csv from %q", inName)

			cr := csv.NewReader(charmap.ISO8859_2.NewDecoder().Reader(r))
			cr.Comma = ';'
			cr.ReuseRecord = true
			var err error
			for {
				row, err := cr.Read()
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
				if row[1] == "F_CEG" {
					log.Println(row)
					continue
				}
				var dr DashboardTableRow
				if err = dr.ParseFields(row); err != nil {
					return fmt.Errorf("%v: %w", row, err)
				}
				dashboardTableRows = append(dashboardTableRows, dr)
			}
			r.Close()
			if err != nil {
				return err
			}

			// Routes
			http.HandleFunc("/", kyoto.PageHandler(&PageIndex{}))
			http.HandleFunc("/dashboard", kyoto.PageHandler(&Dashboard{}))

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
