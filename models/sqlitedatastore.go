package models

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
	log "github.com/sirupsen/logrus"
)

const (
	dbdriver = "sqlite3"
)

// SQLiteDataStore implements the Datastore interface
// to store data in SQLite3
type SQLiteDataStore struct {
	*sqlx.DB
	err error
}

// NewDBstore returns a database connection to the given dataSourceName
// ie. a path to the sqlite database file
func NewDBstore(dataSourceName string) (*SQLiteDataStore, error) {
	var (
		db  *sqlx.DB
		err error
	)
	if db, err = sqlx.Connect(dbdriver, dataSourceName); err != nil {
		return &SQLiteDataStore{}, err
	}
	return &SQLiteDataStore{db, nil}, nil
}

// FlushErrors returns the last DB errors and flushes it.
func (db *SQLiteDataStore) FlushErrors() error {
	// saving the last thrown error
	lastError := db.err
	// resetting the error
	db.err = nil
	// returning the last error
	return lastError
}

// CreateDatabase creates the database tables
func (db *SQLiteDataStore) CreateDatabase() error {
	// activate the foreign keys feature
	if _, db.err = db.Exec("PRAGMA foreign_keys = ON"); db.err != nil {
		return db.err
	}

	// schema definition
	schema := `CREATE TABLE IF NOT EXISTS person(
		person_id INTEGER PRIMARY KEY,
		email string NOT NULL,
		password string NOT NULL);
	CREATE TABLE IF NOT EXISTS entity (
		entity_id INTEGER PRIMARY KEY,
		name string NOT NULL,
		description string,
		person integer,
		FOREIGN KEY (person) references person(person_id));
	CREATE TABLE IF NOT EXISTS permission (
		permission_id INTEGER PRIMARY KEY,
		person integer NOT NULL,
		perm string NOT NULL,
		item string NOT NULL,
		itemid integer,
		FOREIGN KEY (person) references person(person_id));`

	// tables creation
	if _, db.err = db.Exec(schema); db.err != nil {
		return db.err
	}

	// inserting sample values if tables are empty
	var c int
	_ = db.Get(&c, `SELECT count(*) FROM person`)
	log.WithFields(log.Fields{"c": c}).Debug("CreateDatabase")
	if c == 0 {
		people := `INSERT INTO person (email, password) VALUES (?, ?)`
		entities := `INSERT INTO entity (name, description, person) VALUES (?, ?, ?)`
		permissions := `INSERT INTO permission (person, perm, item, itemid) VALUES (?, ?, ?, ?)`

		res1 := db.MustExec(people, "john.doe@foo.com", "johndoe")
		res2 := db.MustExec(people, "mickey.mouse@foo.com", "mickeymouse")
		res3 := db.MustExec(people, "obione.kenobi@foo.com", "obionekenobi")
		res4 := db.MustExec(people, "dark.vader@foo.com", "darkvader")
		johnid, _ := res1.LastInsertId()
		mickeyid, _ := res2.LastInsertId()
		obioneid, _ := res3.LastInsertId()
		darkid, _ := res4.LastInsertId()
		db.MustExec(entities, "entity1", "sample entity one", johnid)
		db.MustExec(entities, "entity2", "sample entity two", mickeyid)
		db.MustExec(entities, "entity3", "sample entity three", obioneid)
		db.MustExec(permissions, johnid, "read", "entities", nil)
		db.MustExec(permissions, johnid, "all", "entity", 1)
		db.MustExec(permissions, mickeyid, "read", "entities", nil)
		db.MustExec(permissions, mickeyid, "read", "entity", 2)
		db.MustExec(permissions, mickeyid, "create", "entity", 2)
		db.MustExec(permissions, mickeyid, "update", "entity", 2)
		db.MustExec(permissions, obioneid, "all", "all", nil)
		db.MustExec(permissions, darkid, "read", "entities", nil)
		db.MustExec(permissions, darkid, "read", "all", nil)
	}
	return nil
}

func (db *SQLiteDataStore) DeleteEntity(id int) error {
	var (
		sql string
	)
	sql = `DELETE FROM entity 
	WHERE entity_id = ?`
	if _, db.err = db.Exec(sql, id); db.err != nil {
		return db.err
	}
	return nil
}

func (db *SQLiteDataStore) UpdateEntity(e Entity) error {
	var (
		sql string
	)
	sql = `UPDATE entity SET name = ?, description = ?
	WHERE entity_id = ?`
	if _, db.err = db.Exec(sql, e.Name, e.Description, e.ID); db.err != nil {
		return db.err
	}
	return nil
}

func (db *SQLiteDataStore) CreateEntity(e Entity) error {
	var (
		sql string
	)
	// Hardcoding the manager
	sql = `INSERT INTO entity(name, description, person) VALUES (?, ?, 3)`
	if _, db.err = db.Exec(sql, e.Name, e.Description); db.err != nil {
		return db.err
	}
	return nil
}

func (db *SQLiteDataStore) GetEntities() ([]Entity, error) {
	var (
		entities []Entity
		sql      string
	)

	sql = "SELECT e.entity_id, e.name, e.description, p.person_id, p.email, p.password FROM entity AS e, person AS p WHERE e.person = p.person_id"
	if db.err = db.Select(&entities, sql); db.err != nil {
		return nil, db.err
	}
	return entities, nil
}

func (db *SQLiteDataStore) GetEntity(ID int) (Entity, error) {
	var (
		entity Entity
		sql    string
	)

	sql = "SELECT e.entity_id, e.name, e.description, p.person_id, p.email, p.password FROM entity AS e, person AS p WHERE e.person = p.person_id AND e.entity_id = ?"
	if db.err = db.Get(&entity, sql, ID); db.err != nil {
		return Entity{}, db.err
	}
	log.WithFields(log.Fields{"ID": ID, "entity": entity}).Debug("GetEntity")
	return entity, nil
}

func (db *SQLiteDataStore) HasPermission(pemail string, perm string, item string, itemid int) (bool, error) {
	var (
		res       bool
		count, id int
		sql       string
	)

	// getting the person id
	sql = "SELECT person_id FROM person WHERE email = ?"
	if db.err = db.Get(&id, sql, pemail); db.err != nil {
		return false, db.err
	}
	log.WithFields(log.Fields{"id": id,
		"pemail": pemail,
		"perm":   perm,
		"item":   item,
		"itemid": itemid}).Debug("HasPermission")

	// then counting the permissions matching the parameters
	if itemid == -1 {
		sql = `SELECT count(*) FROM permission WHERE 
		person = ? AND perm = ? AND item = ? OR 
		person = ? AND perm = "all"`
		if db.err = db.Get(&count, sql, id, perm, item, id); db.err != nil {
			return false, db.err
		}
	} else {
		sql = `SELECT count(*) FROM permission WHERE 
		person = ? AND perm = ? AND item = ? AND itemid = ? OR 
		person = ? AND perm = ? AND item = "all" OR
		person = ? AND perm = "all" AND item = ? OR
		person = ? AND perm = "all" AND item = "all"`
		if db.err = db.Get(&count, sql, id, perm, item, itemid, id, perm, id, item, id); db.err != nil {
			return false, db.err
		}
	}
	log.WithFields(log.Fields{"count": count}).Debug("HasPermission")

	if count == 0 {
		res = false
	} else {
		res = true
	}
	return res, nil
}
