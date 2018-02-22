package models

// Datastore is an interface to be implemented
// to store data
type Datastore interface {
	FlushErrors() error
	CreateDatabase() error
	GetEntities() ([]Entity, error)
	GetEntity(int) (Entity, error)
	DeleteEntity(int) error
	CreateEntity(Entity) error
	UpdateEntity(Entity) error
	HasPermission(string, string, string, int) (bool, error)
}
