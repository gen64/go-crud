package main

import (
	"database/sql"
	"fmt"
	"reflect"
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

func (mc *ModelController) Validate(m IModel) (bool, []int, error) {
	xi := []int{}
	b := true

	h, err := mc.GetHelper(m)
	if err != nil {
		return false, xi, fmt.Errorf("error with GetHelper in Validate: %s", err)
	}

	val := reflect.ValueOf(m).Elem()
	for j := 0; j < len(h.reqFields); j++ {
		valueField := val.Field(h.reqFields[j])
		if valueField.Type().Name() == "string" && valueField.String() == "" {
			xi = append(xi, h.reqFields[j])
			b = false
		}
	}
	for j := 0; j < len(h.lenFields); j++ {
		valueField := val.Field(h.lenFields[j][0])
		if valueField.Type().Name() != "string" {
			continue
		}
		if h.lenFields[j][1] > -1 && len(valueField.String()) < h.lenFields[j][1] {
			xi = append(xi, h.lenFields[j][0])
			b = false
		}
		if h.lenFields[j][2] > -1 && len(valueField.String()) > h.lenFields[j][2] {
			xi = append(xi, h.lenFields[j][0])
			b = false
		}
	}
	return b, xi, nil
}

func (mc *ModelController) SaveToDB(m IModel) error {
	h, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in SaveToDB: %s", err)
	}

	b, _, err := mc.Validate(m)
	if err != nil {
		return fmt.Errorf("error with Validate in SaveToDB: %s", err)
	}

	if !b {
		return fmt.Errorf("error with Validate in SaveToDB")
	}

	if m.GetID() != 0 {
		_, err = mc.dbConn.Exec(h.GetQueryUpdateById(), mc.GetModelForUpdateById(m)...)
	} else {
		err = mc.dbConn.QueryRow(h.GetQueryInsert(), mc.GetModelForInsert(m)...).Scan(mc.GetModelID(m))
	}
	if err != nil {
		return fmt.Errorf("error with db.Exec in SaveToDB: %s", err)
	}
	return nil
}

func (mc *ModelController) SetFromDB(m IModel, id string) error {
	_, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in Validate: %s", err)
	}
	return nil
}

func (mc *ModelController) DeleteFromDB(m IModel) error {
	_, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in Validate: %s", err)
	}
	return nil
}

func (mc *ModelController) GetModelID(u interface{}) interface{} {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Addr().Interface()
}

func (mc *ModelController) GetModelForInsert(u interface{}) []interface{} {
    val := reflect.ValueOf(u).Elem()
    v := make([]interface{}, val.NumField()-1)
    for i := 1; i < val.NumField(); i++ {
        valueField := val.Field(i)
        v[i-1] = valueField.Addr().Interface()
    }
    return v
}

func (mc *ModelController) GetModelForUpdateById(u interface{}) []interface{} {
    val := reflect.ValueOf(u).Elem()
    v := make([]interface{}, val.NumField())
    for i := 0; i < val.NumField(); i++ {
        valueField := val.Field(i)
		j := i-1
		if i == 0 {
			j = val.NumField()-1
		}
        v[j] = valueField.Addr().Interface()
    }
    return v
}
