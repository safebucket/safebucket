package common

type GenericRepo[T any] interface {
	Create(T) (T, error)
	GetList() []T
	GetOne(uint) (T, error)
	Update(uint, T) (T, error)
	Delete(uint) error
}

type Page[T any] struct {
	Data []T `json:"data"`
}

type Error struct {
	Status int      `json:"status"`
	Error  []string `json:"error"`
}
