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
	"reflect"
	"strings"
	"time"

	"github.com/kyoto-framework/kyoto"
	"github.com/tgulacsi/go/zipfs"
)

var _ embed.FS

//go:generate go generate ./static/

//go:generate sh -c "rm -f html.zip; zip -j2 html.zip *.html"
//go:embed html.zip
var htmlZIP []byte

//go:generate sh -c "rm -f uikit.zip; zip -2 uikit.zip uikit/twui/*.html"
//go:embed uikit.zip
var uikitZIP []byte

//go:generate sh -c "rm -f static.zip; (cd static && zip -r2 ../../static.zip dist)"
//go:embed static.zip
var staticZIP []byte

var (
	htmlFS   = newGlobOrZipFS("*.html", htmlZIP)
	uikitFS  = newGlobOrZipFS("uikit/twui/*.html", uikitZIP)
	staticFS = newGlobOrZipFS("static/dist", staticZIP)
)

func newtemplate(page string) *template.Template {
	t := template.New(page)
	t = t.Funcs(kyoto.TFuncMap())
	t = t.Funcs(template.FuncMap{
		"structFieldSlice": func(field string, arr interface{}) interface{} {
			arrV := reflect.ValueOf(arr)
			tf, _ := arrV.Type().Elem().FieldByName(field)
			fieldsV := reflect.MakeSlice(reflect.SliceOf(tf.Type), 0, arrV.Len())
			for i := 0; i < arrV.Len(); i++ {
				fieldsV = reflect.Append(fieldsV, arrV.Index(i).FieldByName(field))
			}
			return fieldsV.Interface()
		},
		"uniq": func(slice interface{}) interface{} {
			switch x := slice.(type) {
			case []string:
				return uniq[string](x)
			}
			arrV := reflect.ValueOf(slice)
			uniqV := reflect.MakeSlice(arrV.Type(), 0, arrV.Len())
			token := reflect.ValueOf(true)
			seenV := reflect.MakeMapWithSize(reflect.MapOf(arrV.Type().Elem(), token.Type()), arrV.Len())
			for i := 0; i < arrV.Len(); i++ {
				kV := arrV.Index(i)
				if !seenV.MapIndex(kV).IsZero() {
					continue
				}
				seenV.SetMapIndex(kV, token)
				reflect.Append(uniqV, kV)
			}
			return uniqV.Interface()
		},
		"now": func() time.Time {
			return time.Now()
		},
	})
	var err error
	if t, err = t.ParseFS(htmlFS, "*.html"); err != nil {
		panic(fmt.Sprintf("htmlfs: %+v", err))
	}
	if t, err = t.ParseFS(uikitFS, "uikit/twui/*.html"); err != nil {
		panic(fmt.Sprintf("uikitfs: %+v", err))
	}
	if t.Lookup(page) == nil {
		panic(fmt.Sprintf("newtemplate %q: no such template", page))
	}

	return t
}

/*
func structFieldSlice[Struct, Field any](fieldName string, slice []Struct) []Field {
	arrV := reflect.ValueOf(arr)
	fieldsV := reflect.MakeSlice(arrV.Type().Elem(), 0, arrV.Len())
	for i := 0; i < arrV.Len(); i++ {
		reflect.Append(fieldsV, arrV.Index(i).FieldByName(field))
	}
	return fieldsV.Interface()
}
*/
func uniq[Elem comparable](slice []Elem) []Elem {
	result := make([]Elem, 0, len(slice))
	seen := make(map[Elem]struct{}, len(slice))
	var token struct{}
	for _, e := range slice {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = token
		result = append(result, e)
	}
	return result
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
		if strings.Contains(pattern, "*") {
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
		}

		if _, err := os.Stat(filepath.Clean(pattern)); err != nil {
			return zipfs.MustNewZipFS(zipfs.BytesSectionReader(zipBytes)), nil
		}
		return os.DirFS(pattern), nil
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

var _ = http.Flusher((*statusResponseWriter)(nil))

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
func (sw *statusResponseWriter) Flush() {
	sw.ResponseWriter.(http.Flusher).Flush()
}
