package main

type IModel interface {
	IsModel() bool
}

type Model struct {
}

func (m *Model) IsModel() bool {
	return true
}
