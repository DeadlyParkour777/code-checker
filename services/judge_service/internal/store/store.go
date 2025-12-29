package store

type Store interface{}

type store struct{}

func NewStore() Store {
	return &store{}
}
