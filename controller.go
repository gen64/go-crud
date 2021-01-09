package crudl

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	_ "log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"log"
)

// Controller is main component that manages database storage for structs and generates CRUDL HTTP handlers
type Controller struct {
	dbConn        *sql.DB
	dbTablePrefix string
	modelHelpers  map[string]*Helper
}

// NewController returns new Controller struct
func NewController(db *sql.DB, p string) *Controller {
	c := &Controller{
		dbConn:        db,
		dbTablePrefix: p,
	}
	return c
}

// DropDBTables drop tables in the database for specified structs
func (mc *Controller) DropDBTables(xm ...interface{}) *ControllerError {
	for _, m := range xm {
		err := mc.DropDBTable(m)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDBTables creates tables in the database for specified structs
func (mc *Controller) CreateDBTables(xm ...interface{}) *ControllerError {
	for _, m := range xm {
		err := mc.CreateDBTable(m)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDBTable creates database table for specified struct
func (mc *Controller) CreateDBTable(m interface{}) *ControllerError {
	h, err := mc.getHelper(m)
	if err != nil {
		return err
	}

	_, err2 := mc.dbConn.Exec(h.GetQueryCreateTable())
	if err2 != nil {
		return &ControllerError{
			Op: "DBQuery",
			Err: err2,
		}
	}
	return nil
}

// DropDBTable drops database table for specified struct
func (mc *Controller) DropDBTable(m interface{}) *ControllerError {
	h, err := mc.getHelper(m)
	if err != nil {
		return err
	}

	_, err2 := mc.dbConn.Exec(h.GetQueryDropTable())
	if err2 != nil {
		return &ControllerError{
			Op: "DBQuery",
			Err: err2,
		}
	}
	return nil
}

// SaveToDB takes struct and saves it to database; calls either INSERT or UPDATE
func (mc *Controller) SaveToDB(m interface{}) *ControllerError {
	h, err := mc.getHelper(m)
	if err != nil {
		return err
	}

	b, _, err2 := mc.Validate(m)
	if err2 != nil || !b {
		return &ControllerError{
			Op: "Validate",
			Err: err2,
		}
	}

	mc.populateLinks(m)

	var err3 error
	if mc.GetModelIDValue(m) != 0 {
		_, err3 = mc.dbConn.Exec(h.GetQueryUpdateById(), append(mc.GetModelFieldInterfaces(m), mc.GetModelIDInterface(m))...)
	} else {
		err3 = mc.dbConn.QueryRow(h.GetQueryInsert(), mc.GetModelFieldInterfaces(m)...).Scan(mc.GetModelIDInterface(m))
	}
	if err3 != nil {
		return &ControllerError{
			Op: "DBQuery",
			Err: err3,
		}
	}
	return nil
}

// SetFromDB sets struct fields from database record
func (mc *Controller) SetFromDB(m interface{}, id string) *ControllerError {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return &ControllerError{
			Op: "IDToInt",
			Err: err,
		}
	}

	h, err2 := mc.getHelper(m)
	if err2 != nil {
		return err2
	}

	err = mc.dbConn.QueryRow(h.GetQuerySelectById(), int64(idInt)).Scan(append(append(make([]interface{}, 0), mc.GetModelIDInterface(m)), mc.GetModelFieldInterfaces(m)...)...)
	switch {
	case err == sql.ErrNoRows:
		mc.ResetFields(m)
		return nil
	case err != nil:
		return &ControllerError{
			Op: "DBQuery",
			Err: err,
		}
	default:
		return nil
	}
	return nil
}

// DeleteFromDB removes struct from the database storage
func (mc *Controller) DeleteFromDB(m interface{}) *ControllerError {
	h, err := mc.getHelper(m)
	if err != nil {
		return err
	}
	if mc.GetModelIDValue(m) == 0 {
		return nil
	}
	_, err2 := mc.dbConn.Exec(h.GetQueryDeleteById(), mc.GetModelIDInterface(m))
	if err2 != nil {
		return &ControllerError{
			Op: "DBQuery",
			Err: err2,
		}
	}
	mc.ResetFields(m)
	return nil
}

// GetModelIDInterface returns interface to ID field of a specified struct
func (mc *Controller) GetModelIDInterface(u interface{}) interface{} {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Addr().Interface()
}

// GetModelIDValue returns value of ID field (int64) of a specified struct
func (mc *Controller) GetModelIDValue(u interface{}) int64 {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Int()
}

// GetModelFieldInterfaces returns list of interfaces for struct fields, excluding the ID field
func (mc *Controller) GetModelFieldInterfaces(u interface{}) []interface{} {
	val := reflect.ValueOf(u).Elem()
	var v []interface{}
	for i := 1; i < val.NumField(); i++ {
		valueField := val.Field(i)
		if valueField.Kind() != reflect.Int64 && valueField.Kind() != reflect.Int && valueField.Kind() != reflect.String {
			continue
		}
		v = append(v, valueField.Addr().Interface())
	}
	return v
}

// ResetFields zeroes struct field values
func (mc *Controller) ResetFields(u interface{}) {
	val := reflect.ValueOf(u).Elem()
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		if valueField.Kind() == reflect.Ptr {
			valueField.Set(reflect.Zero(valueField.Type()))
		}
		if valueField.Kind() == reflect.Int64 {
			valueField.SetInt(0)
		}
		if valueField.Kind() == reflect.Int {
			valueField.SetInt(0)
		}
		if valueField.Kind() == reflect.String {
			valueField.SetString("")
		}
	}
}

// GetHTTPHandler returns HTTP handler (func) that can be attached to HTTP server which creates a CRUDL endpoint for a specific struct
func (mc *Controller) GetHTTPHandler(u interface{}, uri string) func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id, b := mc.getIDFromURI(r.RequestURI[len(uri):], w)
		if !b {
			return
		}
		if r.Method == http.MethodPut {
			mc.handleHTTPPut(w, r, u, id)
			return
		}
		if r.Method == http.MethodGet {
			mc.handleHTTPGet(w, r, u, id)
			return
		}
		if r.Method == http.MethodDelete {
			mc.handleHTTPDelete(w, r, u, id)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return fn
}

// Validate checks values of struct fields; returns bool if they are valid and list of names of invalid fields
func (mc *Controller) Validate(m interface{}) (bool, []int, error) {
	xi := []int{}
	b := true

	h, err := mc.getHelper(m)
	if err != nil {
		return false, xi, err
	}

	val := reflect.ValueOf(m).Elem()
	for j := 0; j < len(h.reqFields); j++ {
		valueField := val.Field(h.reqFields[j])
		if valueField.Type().Name() == "string" && valueField.String() == "" {
			xi = append(xi, h.reqFields[j])
			b = false
		}
		if valueField.Type().Name() == "int" && valueField.Int() == 0 {
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
	for j := 0; j < len(h.valFields); j++ {
		valueField := val.Field(h.valFields[j][0])
		if valueField.Type().Name() != "int" && valueField.Type().Name() != "int64" {
			continue
		}
		if h.valFields[j][1] > -1 && valueField.Int() < int64(h.valFields[j][1]) {
			xi = append(xi, h.valFields[j][0])
			b = false
		}
		if h.valFields[j][2] > -1 && valueField.Int() > int64(h.valFields[j][2]) {
			xi = append(xi, h.valFields[j][0])
			b = false
		}
	}
	for j := 0; j < len(h.emailFields); j++ {
		valueField := val.Field(h.emailFields[j])
		if valueField.Type().Name() != "string" {
			continue
		}
		var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
		if !emailRegex.MatchString(valueField.String()) {
			xi = append(xi, h.emailFields[j])
			b = false
		}
	}
	for k, v := range h.regexpFields {
		valueField := val.Field(k)
		if valueField.Type().Name() != "string" {
			continue
		}
		if !v.MatchString(valueField.String()) {
			xi = append(xi, k)
			b = false
		}
	}
	return b, xi, nil
}

// getHelper returns a special Helper struct that is used to parse fields' tags
// and reflect struct fields to generate SQL queries and validation
func (mc *Controller) getHelper(m interface{}) (*Helper, *ControllerError) {
	v := reflect.ValueOf(m)
	i := reflect.Indirect(v)
	s := i.Type()
	n := s.Name()

	if mc.modelHelpers == nil {
		mc.modelHelpers = make(map[string]*Helper)
	}
	if mc.modelHelpers[n] == nil {
		h := NewHelper(m, mc.dbTablePrefix)
		if h.Err() != nil {
			return nil, &ControllerError{
				Op: "GetHelper",
				Err: h.Err(),
			}
		}
		mc.modelHelpers[n] = h
	}
	return mc.modelHelpers[n], nil
}

// populateLinks gets ID of linked struct and sets it to apriopriate ID field (int64) of original struct
func (mc *Controller) populateLinks(m interface{}) {
	h, err := mc.getHelper(m)
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

func (mc *Controller) handleHTTPPut(w http.ResponseWriter, r *http.Request, u interface{}, id string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if id != "" {
		err2 := mc.SetFromDB(u, id)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if mc.GetModelIDValue(u) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		mc.ResetFields(u)
	}

	err = json.Unmarshal(body, u)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b, _, err := mc.Validate(u)
	if !b || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"err\":\"validation failed: %s\"}", err)))
		return
	}

	err2 := mc.SaveToDB(u)
	if err2 != nil {
		log.Print(err2)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(mc.jsonID(mc.GetModelIDValue(u)))
	return
}

func (mc *Controller) handleHTTPGet(w http.ResponseWriter, r *http.Request, u interface{}, id string) {
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(mc.jsonError("id missing"))
		return
	}

	err := mc.SetFromDB(u, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if mc.GetModelIDValue(u) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	j, err2 := json.Marshal(u)
	if err2 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(j)
	return
}

func (mc *Controller) handleHTTPDelete(w http.ResponseWriter, r *http.Request, u interface{}, id string) {
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(mc.jsonError("id missing"))
		return
	}

	err := mc.SetFromDB(u, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if mc.GetModelIDValue(u) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = mc.DeleteFromDB(u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (mc *Controller) getIDFromURI(uri string, w http.ResponseWriter) (string, bool) {
	xs := strings.SplitN(uri, "?", 2)
	if xs[0] == "" {
		return "", true
	} else {
		matched, err := regexp.Match(`^[0-9]+$`, []byte(xs[0]))
		if err != nil || !matched {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(mc.jsonError("invalid id"))
			return "", false
		}
		return xs[0], true
	}
}

func (mc *Controller) jsonError(e string) []byte {
	return []byte(fmt.Sprintf("{\"err\":\"%s\"}", e))
}

func (mc *Controller) jsonID(id int64) []byte {
	return []byte(fmt.Sprintf("{\"id\":\"%d\"}", id))
}
