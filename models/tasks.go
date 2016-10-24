package models

import (
	"database/sql"

	"github.com/pkg/errors"
)

type Task struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TaskCollection struct {
	Tasks []Task `json:"items"`
}

func GetTasks(db *sql.DB) (TaskCollection, error) {
	var result TaskCollection
	qry := "SELECT * FROM tasks"
	rows, err := db.Query(qry)
	// Exit if the SQL doesn't work for some reason
	if err != nil {
		return result, errors.Wrap(err, qry)
	}
	// make sure to cleanup when the program exits
	defer rows.Close()

	for rows.Next() {
		task := Task{}
		if err := rows.Scan(&task.ID, &task.Name); err != nil {
			return result, errors.Wrap(err, qry)
		}
		result.Tasks = append(result.Tasks, task)
	}
	return result, nil
}

func PutTask(db *sql.DB, name string) (int64, error) {
	qry := "INSERT INTO tasks(name) VALUES(?)"

	// Create a prepared SQL statement
	stmt, err := db.Prepare(qry)
	// Exit if we get an error
	if err != nil {
		return 0, errors.Wrap(err, qry)
	}
	// Make sure to cleanup after the program exits
	defer stmt.Close()

	// Replace the '?' in our prepared statement with 'name'
	result, err := stmt.Exec(name)
	// Exit if we get an error
	if err != nil {
		return 0, errors.Wrapf(err, "%s\n%v", qry, name)
	}

	return result.LastInsertId()
}

func DeleteTask(db *sql.DB, id int) (int64, error) {
	qry := "DELETE FROM tasks WHERE id = ?"

	// Create a prepared SQL statement
	stmt, err := db.Prepare(qry)
	// Exit if we get an error
	if err != nil {
		return 0, errors.Wrap(err, qry)
	}

	// Replace the '?' in our prepared statement with 'id'
	result, err := stmt.Exec(id)
	// Exit if we get an error
	if err != nil {
		return 0, errors.Wrapf(err, "%s\n%v", qry, id)
	}

	return result.RowsAffected()
}
