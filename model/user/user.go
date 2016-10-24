// Package user provides access to the user table in the database.
package user

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/guregu/null.v3"
)

var (
	// table is the table name.
	table = "user"
)

// Item defines the model.
type Item struct {
	ID        uint32    `db:"id"`
	FirstName string    `db:"first_name"`
	LastName  string    `db:"last_name"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	StatusID  uint8     `db:"status_id"`
	CreatedAt null.Time `db:"created_at"`
	UpdatedAt null.Time `db:"updated_at"`
	DeletedAt null.Time `db:"deleted_at"`
}

// Service defines the database connection.
type Service struct {
	DB Connection
}

// Connection is an interface for making queries.
type Connection interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

// ByEmail gets user information from email.
func (c Service) ByEmail(email string) (Item, bool, error) {
	result := Item{}
	qry := fmt.Sprintf(`
		SELECT id, password, status_id, first_name
		FROM %q
		WHERE email = $1
			AND deleted_at IS NULL
		LIMIT 1
		`, table)
	err := c.DB.Get(&result, qry, email)
	return result, err == sql.ErrNoRows, errors.Wrap(err, qry)
}

// Create creates user.
func (c Service) Create(firstName, lastName, email, password string) (sql.Result, error) {
	result, err := c.DB.Exec(fmt.Sprintf(`
		INSERT INTO %q
		(first_name, last_name, email, password)
		VALUES
		($1,$2,$3,$4)
		`, table),
		firstName, lastName, email, password)
	return result, err
}
