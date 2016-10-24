// Package note provides access to the note table in the database.
package note

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"

	"gopkg.in/guregu/null.v3"
)

var (
	// table is the table name.
	table = "note"
)

// Item defines the model.
type Item struct {
	ID        uint32    `db:"id"`
	Name      string    `db:"name"`
	UserID    uint32    `db:"user_id"`
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

// ByID gets an item by ID.
func (s Service) ByID(ID string, userID string) (Item, bool, error) {
	result := Item{}
	qry := fmt.Sprintf(`
		SELECT id, name, user_id, created_at, updated_at, deleted_at
		FROM %q
		WHERE id = $1
			AND user_id = $2
			AND deleted_at IS NULL
		LIMIT 1
		`, table)
	err := s.DB.Get(&result, qry, ID, userID)
	return result, err == sql.ErrNoRows, errors.Wrap(err, qry)
}

// ByUserID gets all entities for a user.
func (s Service) ByUserID(userID string) ([]Item, bool, error) {
	var result []Item
	qry := fmt.Sprintf(`
		SELECT id, name, user_id, created_at, updated_at, deleted_at
		FROM %q
		WHERE user_id = $1
			AND deleted_at IS NULL
		`, table)
	err := s.DB.Select(&result, qry, userID)
	return result, err == sql.ErrNoRows, errors.Wrap(err, qry)
}

// Create adds an item.
func (s Service) Create(name string, userID string) (sql.Result, error) {
	qry := fmt.Sprintf(`
		INSERT INTO %q
		(name, user_id)
		VALUES
		($1,$2)
		`, table)
	result, err := s.DB.Exec(qry, name, userID)
	return result, errors.Wrap(err, qry)
}

// Update makes changes to an existing item.
func (s Service) Update(name string, ID string, userID string) (sql.Result, error) {
	qry := fmt.Sprintf(`
		UPDATE %q
		SET name = $1
		WHERE id = $2
			AND user_id = $3
			AND deleted_at IS NULL
		LIMIT 1
		`, table)
	result, err := s.DB.Exec(qry, name, ID, userID)
	return result, errors.Wrap(err, qry)
}

// DeleteHard removes an item.
func (s Service) DeleteHard(ID string, userID string) (sql.Result, error) {
	qry := fmt.Sprintf(`
		DELETE FROM %q
		WHERE id = $1
			AND user_id = $2
			AND deleted_at IS NULL
		`, table)
	result, err := s.DB.Exec(qry, ID, userID)
	return result, errors.Wrap(err, qry)
}

// DeleteSoft marks an item as removed.
func (s Service) DeleteSoft(ID string, userID string) (sql.Result, error) {
	qry := fmt.Sprintf(`
		UPDATE %q
		SET deleted_at = NOW()
		WHERE id = $1
			AND user_id = $2
			AND deleted_at IS NULL
		LIMIT 1
		`, table)
	result, err := s.DB.Exec(qry, ID, userID)
	return result, errors.Wrap(err, qry)
}
