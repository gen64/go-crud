package main

import (
	"database/sql"
	"fmt"
	"reflect"
)

type ModelController struct {
	dbConn *sql.DB
	dbTablePrefix string
	modelHelpers map[string]*ModelHelper
}

func (mc *ModelController) SetDBTablePrefix(p string) {
	mc.dbTablePrefix = p
}

func (mc *ModelController) AttachDBConn(db *sql.DB) {
	mc.dbConn = db
}

func (mc *ModelController) GetHelper(m interface{}) (*ModelHelper, error) {
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

func (mc *ModelController) DropDBTables(xm ...interface{}) error {
	for _, m := range xm {
		err := mc.DropDBTable(m)
		if err != nil {
			return fmt.Errorf("error with DropDBTable: %s", err)
		}
	}
	return nil
}

func (mc *ModelController) CreateDBTables(xm ...interface{}) error {
	for _, m := range xm {
		err := mc.CreateDBTable(m)
		if err != nil {
			return fmt.Errorf("error with CreateDBTable: %s", err)
		}
	}
	return nil
}

func (mc *ModelController) Validate(m interface{}) (bool, []int, error) {
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
		if valueField.Type().Name() == "int64" && valueField.Int() == 0 {
			bf := true
			// Check if field is not a link
			for l := 0; l < len(h.linkFields); l++ {
				if h.linkFields[l][0] == h.reqFields[j] {
					valueLinkField := val.Field(h.linkFields[l][1])
					// If linked field is nil or linked object ID is 0
					if valueLinkField.IsNil() {
						bf = false
					} else {
						linkedId := mc.GetModelIDValue(reflect.Indirect(valueLinkField).Addr().Interface())
						if linkedId == 0 {
							bf = false
						}
					}
					break
				}
			}
			if !bf {
				xi = append(xi, h.reqFields[j])
				b = false
			}
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

func (mc *ModelController) PopulateLinks(m interface{}) {
	h, err := mc.GetHelper(m)
	if err != nil {
		return
	}

	val := reflect.ValueOf(m).Elem()
	for l := 0; l < len(h.linkFields); l++ {
		valueTargetField := val.Field(h.linkFields[l][0])
		valueSourceField := val.Field(h.linkFields[l][1])
		if !valueSourceField.IsNil() {
			linkedId := mc.GetModelIDValue(reflect.Indirect(valueSourceField).Addr().Interface())
			if linkedId != 0 {
				valueTargetField.SetInt(linkedId)
			}
		}
	}
}

func (mc *ModelController) CreateDBTable(m interface{}) error {
	h, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in CreateDBTable: %s", err)
	}

	_, err = mc.dbConn.Exec(h.GetQueryCreateTable())
	if err != nil {
		return fmt.Errorf("error with db.Exec in CreateDBTable: %s", err)
	}
	return nil
}

func (mc *ModelController) DropDBTable(m interface{}) error {
	h, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in DropDBTable: %s", err)
	}

	_, err = mc.dbConn.Exec(h.GetQueryDropTable())
	if err != nil {
		return fmt.Errorf("error with db.Exec in DropDBTable: %s", err)
	}
	return nil
}

func (mc *ModelController) SaveToDB(m interface{}) error {
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

	mc.PopulateLinks(m)

	if mc.GetModelIDValue(m) != 0 {
		_, err = mc.dbConn.Exec(h.GetQueryUpdateById(), append(mc.GetModelFieldInterfaces(m), mc.GetModelIDInterface(m))...)
	} else {
		err = mc.dbConn.QueryRow(h.GetQueryInsert(), mc.GetModelFieldInterfaces(m)...).Scan(mc.GetModelIDInterface(m))
	}
	if err != nil {
		return fmt.Errorf("error with db.Exec in SaveToDB: %s", err)
	}
	return nil
}

func (mc *ModelController) SetFromDB(m interface{}, id string) error {
	_, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in Validate: %s", err)
	}
	return nil
}

func (mc *ModelController) DeleteFromDB(m interface{}) error {
	_, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in Validate: %s", err)
	}
	return nil
}

func (mc *ModelController) GetModelIDInterface(u interface{}) interface{} {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Addr().Interface()
}

func (mc *ModelController) GetModelIDValue(u interface{}) int64 {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Int()
}

func (mc *ModelController) GetModelFieldInterfaces(u interface{}) []interface{} {
    val := reflect.ValueOf(u).Elem()
    var v []interface{}
    for i := 1; i < val.NumField(); i++ {
        valueField := val.Field(i)
		if valueField.Kind() != reflect.Int64 && valueField.Kind() != reflect.String {
			continue
		}
        v = append(v, valueField.Addr().Interface())
    }
    return v
}
