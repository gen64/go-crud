package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"log"
)

type ModelController struct {
	dbConn *sql.DB
	modelHelpers map[string]*ModelHelper
}

func (mc *ModelController) AttachDBConn(db *sql.DB) {
	mc.dbConn = db
}

func (mc *ModelController) GetHelper(m IModel) (*ModelHelper, error) {
	v := reflect.ValueOf(m)
	i := reflect.Indirect(v)
	s := i.Type()
	n := s.Name()

	if mc.modelHelpers == nil {
		mc.modelHelpers = make(map[string]*ModelHelper)
	}
	if mc.modelHelpers[n] == nil {
		h, err := NewModelHelper(m)
		if err != nil {
			return nil, fmt.Errorf("error with NewModelHelper in GetHelper: %s", err)
		}
		mc.modelHelpers[n] = h
	}
	return mc.modelHelpers[n], nil
}

func (mc *ModelController) ValidateMdl(m IModel) error {
	return nil
}

func (mc *ModelController) SaveToDB(m IModel) error {
	h, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in SaveToDB: %s", err)
	}
	log.Print(h)
	return nil
}

func (mc *ModelController) SetFromDB(m IModel, id string) error {
	return nil
}

func (mc *ModelController) DeleteFromDB(m IModel) error {
	return nil
}
