// Package model handles the loading of models.
package model

import (
	"github.com/UNO-SOFT/szamlazo/model/note"
	"github.com/UNO-SOFT/szamlazo/model/user"

	"github.com/jmoiron/sqlx"
)

var (
	Note note.Service // Note model
	User user.Service // User model
)

// Load injects the dependencies for the models
func Load(db *sqlx.DB) {
	Note = note.Service{db}
	User = user.Service{db}
}
