package main

import (
	"archive/zip"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path"

	"github.com/kyoto-framework/kyoto"
)

var _ embed.FS

//go:generate sh -c "zip -9 html.zip *.html"
//go:embed html.zip
var htmlZIP []byte

//go:generate sh -c "zip -9 twui.zip uikit/twui/*.html"
//go:embed twui.zip
var uikitZIP []byte

var (
	htmlFS  = NewZipFS(htmlZIP)
	uikitFS = NewZipFS(uikitZIP)
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

type ZipFS struct {
	z *zip.Reader
	m map[string]int
}

type zipFile struct {
	io.ReadCloser
	fh fs.FileInfo
}

func (zf zipFile) Stat() (fs.FileInfo, error) { return zf.fh, nil }

func (f ZipFS) Open(name string) (fs.File, error) {
	if i, ok := f.m[name]; ok {
		F := f.z.File[i]
		rc, err := F.Open()
		if err != nil {
			return nil, err
		}
		return zipFile{ReadCloser: rc, fh: F.FileInfo()}, nil
	}
	return nil, fs.ErrNotExist
}
func (f ZipFS) Glob(pattern string) ([]string, error) {
	des := make([]string, 0, len(f.z.File))
	for _, F := range f.z.File {
		if ok, err := path.Match(pattern, F.Name); ok {
			des = append(des, F.Name)
		} else if err != nil {
			return des, err
		}
	}
	return des, nil
}

var _ = fs.GlobFS(ZipFS{})

func NewZipFS(zipData []byte) fs.FS {
	z, err := zip.NewReader(
		bytes.NewReader(zipData), int64(len(zipData)),
	)
	if err != nil {
		panic(err)
	}
	m := make(map[string]int, len(z.File))
	for i, F := range z.File {
		m[F.Name] = i
	}
	return ZipFS{z: z, m: m}
}
