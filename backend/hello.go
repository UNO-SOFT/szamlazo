package backend

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func init() {

	// Had to do this because returns svg as text/xml when running on AppEngine: http://goo.gl/hwZSp2
	mime.AddExtensionType(".svg", "image/svg+xml")
}
func RegisterHandler(m interface {
	Handle(string, http.Handler)
}, db *sql.DB) {
	r := mux.NewRouter()
	sr := r.PathPrefix("/api").Subrouter()
	sr.HandleFunc("/posts", posts{db: db}.Posts)
	r.HandleFunc("/{rest:.*}", handler)
	m.Handle("/", r)
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("path:", r.URL.Path)
	http.ServeFile(w, r, "frontend/"+r.URL.Path)
}

type Post struct {
	Uid      int    `json:"uid"`
	Text     string `json:"text"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Favorite bool   `json:"favorite"`
}

type posts struct {
	db *sql.DB
}

func (ps posts) Posts(w http.ResponseWriter, r *http.Request) {
	qry := "SELECT id, name FROM tasks"
	rows, err := ps.db.Query(qry)
	if err != nil {
		ReturnError(w, errors.Wrap(err, qry))
		return
	}
	defer rows.Close()

	w.Write([]byte{'['})
	enc := json.NewEncoder(w)
	for rows.Next() {
		var p Post
		if err = rows.Scan(&p.Uid, &p.Username); err == nil {
			err = errors.Wrapf(enc.Encode(p), "%#v", p)
		}
		if err != nil {
			ReturnError(w, err)
			return
		}
	}
	w.Write([]byte{']'})
}

func ReturnError(w http.ResponseWriter, err error) {
	fmt.Fprint(w, "{\"error\": \"%+v\"}", err)
}
