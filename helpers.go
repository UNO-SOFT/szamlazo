package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kyoto-framework/kyoto"
	"github.com/tgulacsi/go/zipfs"
)

var _ embed.FS

//go:generate sh -c "rm -f html.zip; zip -j9 html.zip *.html uikit/twui/*.html"
//go:embed html.zip
var htmlZIP []byte

//go:generate sh -c "rm -f static.zip; zip -r9 static.zip static/css static/img"
//go:embed static.zip
var staticZIP []byte

var (
	htmlFS   = newGlobOrZipFS("*.html", htmlZIP)
	staticFS = newGlobOrZipFS("static/", staticZIP)

	_ = newtemplate("page.index.html")
	_ = newtemplate("SASS")
)

func newtemplate(page string) *template.Template {
	t := template.New(page)
	t = t.Funcs(kyoto.TFuncMap())
	var err error
	if t, err = t.ParseFS(htmlFS, "*.html"); err != nil {
		panic(fmt.Sprintf("htmlfs: %+v", err))
	}

	return t
}

type mergeFS struct {
	A, B fs.FS
}

var _ = fs.FS(mergeFS{})

func (m mergeFS) Open(name string) (fs.File, error) {
	if f, err := m.A.Open(name); err == nil {
		return f, nil
	}
	return m.B.Open(name)
}
func newGlobOrZipFS(pattern string, zipBytes []byte) fs.FS {
	fsys, err := func() (fs.FS, error) {
		names, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("%q: %w", pattern, err)
		}
		if len(names) == 0 {
			return zipfs.MustNewZipFS(zipfs.BytesSectionReader(zipBytes)), nil
		}
		files := make(map[string]struct{}, len(names))
		dirs := make(map[string]struct{})
		for _, f := range names {
			files[f] = struct{}{}
			dirs[filepath.Dir(f)] = struct{}{}
		}
		return limitFS{files: files, dirs: dirs, fsys: os.DirFS(".")}, nil
	}()
	if err != nil {
		panic(err)
	}
	if err := fs.WalkDir(fsys, ".", func(path string, de fs.DirEntry, err error) error {
		return nil
	}); err != nil {
		panic(err)
	}
	if _, err = fs.Glob(fsys, "*.html"); err != nil {
		panic(err)
	}
	return fsys
}

type limitFS struct {
	files map[string]struct{}
	dirs  map[string]struct{}
	fsys  fs.FS
}

func (lf limitFS) Open(name string) (fs.File, error) {
	_, ok := lf.files[name]
	if !ok {
		_, ok = lf.dirs[name]
	}
	if ok || name == "" || name == "." || name == "/" {
		return lf.fsys.Open(name)
	}
	return nil, fmt.Errorf("%q: %w", name, fs.ErrNotExist)
}

func RequestLoggerMiddleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		sw := NewStatusResponseWriter(w)

		defer func() {
			log.Printf(
				"[%s] [%v] [%d] %s %s %s",
				req.Method,
				time.Since(start),
				sw.statusCode,
				req.Host,
				req.URL.Path,
				req.URL.RawQuery,
			)
		}()

		next.ServeHTTP(sw, req)
	})
}

// WriteHeader assigns status code and header to ResponseWriter of statusResponseWriter object
func (sw *statusResponseWriter) WriteHeader(statusCode int) {
	sw.statusCode = statusCode
	sw.ResponseWriter.WriteHeader(statusCode)
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewStatusResponseWriter returns pointer to a new statusResponseWriter object
func NewStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}
