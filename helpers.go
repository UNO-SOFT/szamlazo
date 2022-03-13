package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kyoto-framework/kyoto"
	"github.com/tgulacsi/go/zipfs"
)

var _ embed.FS

//go:generate sh -c "zip -9 html.zip *.html"
//go:embed html.zip
var htmlZIP []byte

//go:generate sh -c "zip -9 twui.zip uikit/twui/*.html"
//go:embed twui.zip
var uikitZIP []byte

var (
	htmlFS  = newGlobOrZipFS("*.html", htmlZIP)
	uikitFS = newGlobOrZipFS("uikit/twui/*.html", uikitZIP)
)

func newtemplate(page string) *template.Template {
	t := template.New(page)
	t = t.Funcs(kyoto.TFuncMap())
	var err error
	if t, err = t.ParseFS(htmlFS, "*.html"); err != nil {
		panic(fmt.Sprintf("htmlfs: %+v", err))
	}

	if t, err = t.ParseFS(uikitFS, "uikit/twui/*.html"); err != nil {
		panic(fmt.Sprintf("uikitfs: %+v", err))
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
