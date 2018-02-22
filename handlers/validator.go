package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/tbellembois/gowebskel/models"
)

type LoginNameResp struct {
	Name string `json:"name"`
}

func (env *Env) ValidateLoginNameHandler(w http.ResponseWriter, r *http.Request) *models.AppError {
	var (
		resp string
		name string
	)
	vars := r.URL.Query()
	if pname, ok := vars["name"]; !ok {
		resp = "the name must contain the word \"ilovego\""
	} else {
		name = pname[0]
		if strings.Contains(name, "ilovego") {
			resp = "true"
		} else {
			resp = "the name must contain the word \"ilovego\""
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	return nil
}
