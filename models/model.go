package models

import (
	"net/http"
)

// AppHandlerFunc is an HandlerFunc returning an AppError
type AppHandlerFunc func(http.ResponseWriter, *http.Request) *AppError

// AppError is the error type returned by the custom handlers
type AppError struct {
	Error   error
	Message string
	Code    int
}

// Entity represent a department, a laboratory...
type Entity struct {
	ID          int    `db:"entity_id" json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Person
}

// Person represent a person
type Person struct {
	ID       int    `db:"person_id" json:"id"`
	Email    string `json:"email" schema:"email"`
	Password string `json:"password" schema:"password"`
}

// Permission represent who is able to do what on something
type Permission struct {
	ID int `db:"permission_id"`
	Person
	Perm   string // ex: read
	Item   string // ex: entity
	ItemID int    // ex: 8
}
