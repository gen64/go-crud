package main

type IModel interface {
	GetID() int64
}

type Model struct {
	ID    int64  `json:"id"`
	Flags int64  `json:"flags"`
}

func (m *Model) GetID() int64 {
	return m.ID
}
