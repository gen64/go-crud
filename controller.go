package crudl

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	_ "log"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Controller is the main component that gets and saves objects in the database
// and generates CRUDL HTTP handler that can be attached to an HTTP server.
type Controller struct {
	dbConn       *sql.DB
	dbTblPrefix  string
	modelHelpers map[string]*Helper
}

func NewController(dbConn *sql.DB, tblPrefix string) *Controller {
	c := &Controller{
		dbConn:      dbConn,
		dbTblPrefix: tblPrefix,
	}
	return c
}

// DropDBTables drop tables in the database for specified objects (see
// DropDBTable for a single struct)
func (c Controller) DropDBTables(xobj ...interface{}) *ControllerError {
	for _, obj := range xobj {
		err := c.DropDBTable(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDBTables creates tables in the database for specified objects (see
// CreateDBTable for a single struct)
func (c Controller) CreateDBTables(xobj ...interface{}) *ControllerError {
	for _, obj := range xobj {
		err := c.CreateDBTable(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateDBTable creates database table to store specified type of objects. It
// takes struct name and its fields, converts them into table and columns names
// (all lowercase with underscore), assigns column type based on the field type,
// and then executes "CREATE TABLE" query on attached DB connection
func (c Controller) CreateDBTable(obj interface{}) *ControllerError {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}

	_, err2 := c.dbConn.Exec(h.GetQueryCreateTable())
	if err2 != nil {
		return &ControllerError{
			Op:  "DBQuery",
			Err: err2,
		}
	}
	return nil
}

// DropDBTable drops database table used to store specified type of objects. It
// just takes struct name, converts it to lowercase-with-underscore table name
// and executes "DROP TABLE" query using attached DB connection
func (c Controller) DropDBTable(obj interface{}) *ControllerError {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}

	_, err2 := c.dbConn.Exec(h.GetQueryDropTable())
	if err2 != nil {
		return &ControllerError{
			Op:  "DBQuery",
			Err: err2,
		}
	}
	return nil
}

// SaveToDB takes object, validates its field values and saves it in the
// database.
// If ID field is already set (it's greater than 0) then the function assumes
// that record with such ID already exists in the database and the function with
// execute an "UPDATE" query. Otherwise it will be "INSERT". After inserting,
// new record ID is set to struct's ID field
func (c Controller) SaveToDB(obj interface{}) *ControllerError {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}

	b, _, err2 := c.Validate(obj)
	if err2 != nil || !b {
		return &ControllerError{
			Op:  "Validate",
			Err: err2,
		}
	}

	c.populateLinks(obj)

	var err3 error
	if c.GetModelIDValue(obj) != 0 {
		_, err3 = c.dbConn.Exec(h.GetQueryUpdateById(), append(c.GetModelFieldInterfaces(obj), c.GetModelIDInterface(obj))...)
	} else {
		err3 = c.dbConn.QueryRow(h.GetQueryInsert(), c.GetModelFieldInterfaces(obj)...).Scan(c.GetModelIDInterface(obj))
	}
	if err3 != nil {
		return &ControllerError{
			Op:  "DBQuery",
			Err: err3,
		}
	}
	return nil
}

// SetFromDB sets object's fields with values from the database table with a
// specific id. If record does not exist in the database, all field values in
// the struct are zeroed
func (c Controller) SetFromDB(obj interface{}, id string) *ControllerError {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return &ControllerError{
			Op:  "IDToInt",
			Err: err,
		}
	}

	h, err2 := c.getHelper(obj)
	if err2 != nil {
		return err2
	}

	err3 := c.dbConn.QueryRow(h.GetQuerySelectById(), int64(idInt)).Scan(append(append(make([]interface{}, 0), c.GetModelIDInterface(obj)), c.GetModelFieldInterfaces(obj)...)...)
	switch {
	case err3 == sql.ErrNoRows:
		c.ResetFields(obj)
		return nil
	case err3 != nil:
		return &ControllerError{
			Op:  "DBQuery",
			Err: err,
		}
	default:
		return nil
	}
	return nil
}

// DeleteFromDB removes object from the database table and it does that only
// when ID field is set (greater than 0). Once deleted from the DB, all field
// values are zeroed
func (c Controller) DeleteFromDB(obj interface{}) *ControllerError {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}
	if c.GetModelIDValue(obj) == 0 {
		return nil
	}
	_, err2 := c.dbConn.Exec(h.GetQueryDeleteById(), c.GetModelIDInterface(obj))
	if err2 != nil {
		return &ControllerError{
			Op:  "DBQuery",
			Err: err2,
		}
	}
	c.ResetFields(obj)
	return nil
}

// GetModelIDInterface returns an interface{} to ID field of an object
func (c *Controller) GetModelIDInterface(obj interface{}) interface{} {
	return reflect.ValueOf(obj).Elem().FieldByName("ID").Addr().Interface()
}

// GetModelIDValue returns value of ID field (int64) of an object
func (c *Controller) GetModelIDValue(obj interface{}) int64 {
	return reflect.ValueOf(obj).Elem().FieldByName("ID").Int()
}

// GetModelFieldInterfaces returns list of interfaces to object's fields without
// the ID field
func (c Controller) GetModelFieldInterfaces(obj interface{}) []interface{} {
	val := reflect.ValueOf(obj).Elem()
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

// ResetFields zeroes object's field values
func (c Controller) ResetFields(obj interface{}) {
	val := reflect.ValueOf(obj).Elem()
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

// GetHTTPHandler returns a CRUDL HTTP handler that can be attached to HTTP
// server. It creates a CRUDL endpoint for PUT, GET and DELETE methods. Listing
// many records is not yet implemented.
// It's important to pass "uri" argument same as the one that the handler is
// attached to.
func (c Controller) GetHTTPHandler(obj interface{}, uri string) func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id, b := c.getIDFromURI(r.RequestURI[len(uri):], w)
		if !b {
			return
		}
		if r.Method == http.MethodPut {
			c.handleHTTPPut(w, r, obj, id)
			return
		}
		if r.Method == http.MethodGet {
			c.handleHTTPGet(w, r, obj, id)
			return
		}
		if r.Method == http.MethodDelete {
			c.handleHTTPDelete(w, r, obj, id)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return fn
}

// Validate checks object's fields. It returns result of validation as
// a bool and list of fields with invalid value
func (c Controller) Validate(obj interface{}) (bool, []int, error) {
	xi := []int{}
	b := true

	h, err := c.getHelper(obj)
	if err != nil {
		return false, xi, err
	}

	val := reflect.ValueOf(obj).Elem()
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
						linkedId := c.GetModelIDValue(reflect.Indirect(valueLinkField).Addr().Interface())
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

// getHelper returns a special Helper instance which reflects the struct type
// to get SQL queries, validation etc.
func (c *Controller) getHelper(obj interface{}) (*Helper, *ControllerError) {
	v := reflect.ValueOf(obj)
	i := reflect.Indirect(v)
	s := i.Type()
	n := s.Name()

	if c.modelHelpers == nil {
		c.modelHelpers = make(map[string]*Helper)
	}
	if c.modelHelpers[n] == nil {
		h := NewHelper(obj, c.dbTblPrefix)
		if h.Err() != nil {
			return nil, &ControllerError{
				Op:  "GetHelper",
				Err: h.Err(),
			}
		}
		c.modelHelpers[n] = h
	}
	return c.modelHelpers[n], nil
}

// populateLinks is used when there is an int64 field (eg. CreatedByID int64)
// which is a link (foreign key) to another object (eg. User), and field of
// struct type which is an instance of that link (eg. CreatedBy User), to copy
// ID from the instance to the int64 field (eg. CreatedBy.ID to CreatedByID).
// However, it only works when the linked object exists in the database
// (see SaveToDB)
func (c Controller) populateLinks(obj interface{}) {
	h, err := c.getHelper(obj)
	if err != nil {
		return
	}

	val := reflect.ValueOf(obj).Elem()
	for l := 0; l < len(h.linkFields); l++ {
		valueTargetField := val.Field(h.linkFields[l][0])
		valueSourceField := val.Field(h.linkFields[l][1])
		if !valueSourceField.IsNil() {
			linkedId := c.GetModelIDValue(reflect.Indirect(valueSourceField).Addr().Interface())
			if linkedId != 0 {
				valueTargetField.SetInt(linkedId)
			}
		}
	}
}

func (c Controller) handleHTTPPut(w http.ResponseWriter, r *http.Request, obj interface{}, id string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if id != "" {
		err2 := c.SetFromDB(obj, id)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if c.GetModelIDValue(obj) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		c.ResetFields(obj)
	}

	err = json.Unmarshal(body, obj)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b, _, err := c.Validate(obj)
	if !b || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"err\":\"validation failed: %s\"}", err)))
		return
	}

	err2 := c.SaveToDB(obj)
	if err2 != nil {
		log.Print(err2)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(c.jsonID(c.GetModelIDValue(obj)))
	return
}

func (c Controller) handleHTTPGet(w http.ResponseWriter, r *http.Request, obj interface{}, id string) {
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(c.jsonError("id missing"))
		return
	}

	err := c.SetFromDB(obj, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if c.GetModelIDValue(obj) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	j, err2 := json.Marshal(obj)
	if err2 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(j)
	return
}

func (c Controller) handleHTTPDelete(w http.ResponseWriter, r *http.Request, obj interface{}, id string) {
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(c.jsonError("id missing"))
		return
	}

	err := c.SetFromDB(obj, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c.GetModelIDValue(obj) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = c.DeleteFromDB(obj)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (c Controller) getIDFromURI(uri string, w http.ResponseWriter) (string, bool) {
	xs := strings.SplitN(uri, "?", 2)
	if xs[0] == "" {
		return "", true
	} else {
		matched, err := regexp.Match(`^[0-9]+$`, []byte(xs[0]))
		if err != nil || !matched {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(c.jsonError("invalid id"))
			return "", false
		}
		return xs[0], true
	}
}

func (c Controller) jsonError(e string) []byte {
	return []byte(fmt.Sprintf("{\"err\":\"%s\"}", e))
}

func (c Controller) jsonID(id int64) []byte {
	return []byte(fmt.Sprintf("{\"id\":\"%d\"}", id))
}
