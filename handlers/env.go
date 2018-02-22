package handlers

import (
	"html/template"

	"github.com/tbellembois/gowebskel/models"
)

// Env is a structure used to pass objects throughout the application.
type Env struct {
	DB        models.Datastore
	Templates map[string]*template.Template
}
