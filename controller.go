package crud

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// Controller is the main component that gets and saves objects in the database
// and generates CRUD HTTP handler that can be attached to an HTTP server.
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

	b, _, err2 := c.Validate(obj, nil)
	if err2 != nil || !b {
		return &ControllerError{
			Op:  "Validate",
			Err: err2,
		}
	}

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

// GetFromDB runs a select query on the database with specified filters, order,
// limit and offset and returns a list of objects
func (c Controller) GetFromDB(newObjFunc func() interface{}, order map[string]string, limit int, offset int, filters map[string]interface{}) ([]interface{}, *ControllerError) {
	obj := newObjFunc()
	h, err := c.getHelper(obj)
	if err != nil {
		return nil, err
	}

	b, _, err1 := c.Validate(obj, filters)
	if err1 != nil || !b {
		return nil, &ControllerError{
			Op:  "ValidateFilters",
			Err: err1,
		}
	}

	var v []interface{}
	rows, err2 := c.dbConn.Query(h.GetQuerySelect(order, limit, offset, filters), c.GetFiltersInterfaces(filters)...)
	if err2 != nil {
		return nil, &ControllerError{
			Op:  "DBQuery",
			Err: err2,
		}
	}
	defer rows.Close()

	for rows.Next() {
		newObj := newObjFunc()
		err3 := rows.Scan(append(append(make([]interface{}, 0), c.GetModelIDInterface(newObj)), c.GetModelFieldInterfaces(newObj)...)...)
		if err3 != nil {
			return nil, &ControllerError{
				Op:  "DBQueryRowsScan",
				Err: err3,
			}
		}
		v = append(v, newObj)
	}

	return v, nil
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

// GetFiltersInterfaces returns list of interfaces from filters map (used in
// querying)
func (c Controller) GetFiltersInterfaces(mf map[string]interface{}) []interface{} {
	var xi []interface{}
	if mf != nil && len(mf) > 0 {
		for _, v := range mf {
			xi = append(xi, v)
		}
	}
	return xi
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

// GetHTTPHandler returns a CRUD HTTP handler that can be attached to HTTP
// server. It creates a CRUD endpoint for PUT, GET and DELETE methods. Listing
// many records is not yet implemented.
// It's important to pass "uri" argument same as the one that the handler is
// attached to.
func (c Controller) GetHTTPHandler(newObjFunc func() interface{}, uri string) func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		id, b := c.getIDFromURI(r.RequestURI[len(uri):], w)
		if !b {
			return
		}
		if r.Method == http.MethodPut {
			c.handleHTTPPut(w, r, newObjFunc, id)
			return
		}
		if r.Method == http.MethodGet {
			c.handleHTTPGet(w, r, newObjFunc, id)
			return
		}
		if r.Method == http.MethodDelete {
			c.handleHTTPDelete(w, r, newObjFunc, id)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return fn
}

// Validate checks object's fields. It returns result of validation as
// a bool and list of fields with invalid value
func (c Controller) Validate(obj interface{}, filters map[string]interface{}) (bool, []string, error) {
	failedFields := []string{}
	b := true

	h, err := c.getHelper(obj)
	if err != nil {
		return false, failedFields, err
	}

	val := reflect.ValueOf(obj).Elem()

	// TODO: Shorten below code
	if filters == nil {
		for k, _ := range h.fieldsRequired {
			valueField := val.FieldByName(k)
			if valueField.Type().Name() == "string" && valueField.String() == "" {
				failedFields = append(failedFields, k)
				b = false
			}
			if valueField.Type().Name() == "int" && valueField.Int() == 0 {
				failedFields = append(failedFields, k)
				b = false
			}
			if valueField.Type().Name() == "int64" && valueField.Int() == 0 {
				failedFields = append(failedFields, k)
				b = false
			}
		}
	}
	for k, v := range h.fieldsLength {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
			if valueField.Type().Name() != val.FieldByName(k).Type().Name() {
				failedFields = append(failedFields, k)
				b = false
				continue
			}
		} else {
			valueField = val.FieldByName(k)
		}
		if valueField.Type().Name() != "string" {
			continue
		}
		if v[0] > -1 && len(valueField.String()) < v[0] {
			failedFields = append(failedFields, k)
			b = false
		}
		if v[1] > -1 && len(valueField.String()) > v[1] {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	for k, v := range h.fieldsValue {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
			if valueField.Type().Name() != val.FieldByName(k).Type().Name() {
				failedFields = append(failedFields, k)
				b = false
				continue
			}
		} else {
			valueField = val.FieldByName(k)
		}
		if valueField.Type().Name() != "int" && valueField.Type().Name() != "int64" {
			continue
		}
		if v[0] > -1 && valueField.Int() < int64(v[0]) {
			failedFields = append(failedFields, k)
			b = false
		}
		if v[1] > -1 && valueField.Int() > int64(v[1]) {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	for k, _ := range h.fieldsEmail {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
			if valueField.Type().Name() != val.FieldByName(k).Type().Name() {
				failedFields = append(failedFields, k)
				b = false
				continue
			}
		} else {
			valueField = val.FieldByName(k)
		}
		if valueField.Type().Name() != "string" {
			continue
		}
		var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
		if !emailRegex.MatchString(valueField.String()) {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	for k, v := range h.fieldsRegExp {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
			if valueField.Type().Name() != val.FieldByName(k).Type().Name() {
				failedFields = append(failedFields, k)
				b = false
				continue
			}
		} else {
			valueField = val.FieldByName(k)
		}
		if valueField.Type().Name() != "string" {
			continue
		}
		if !v.MatchString(valueField.String()) {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	return b, failedFields, nil
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

func (c Controller) handleHTTPPut(w http.ResponseWriter, r *http.Request, newObjFunc func() interface{}, id string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	objClone := newObjFunc()

	if id != "" {
		err2 := c.SetFromDB(objClone, id)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if c.GetModelIDValue(objClone) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		c.ResetFields(objClone)
	}

	err = json.Unmarshal(body, objClone)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	b, _, err := c.Validate(objClone, nil)
	if !b || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("{\"err\":\"validation failed: %s\"}", err)))
		return
	}

	err2 := c.SaveToDB(objClone)
	if err2 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(c.jsonID(c.GetModelIDValue(objClone)))
	return
}

func (c Controller) handleHTTPGet(w http.ResponseWriter, r *http.Request, newObjFunc func() interface{}, id string) {
	if id == "" {
		obj := newObjFunc()
		params := c.getParamsFromURI(r.RequestURI)

		limit, _ := strconv.Atoi(params["limit"])
		offset, _ := strconv.Atoi(params["offset"])
		if limit < 1 {
			limit = 10
		}
		if offset < 0 {
			offset = 0
		}

		order := make(map[string]string)
		if params["order"] != "" {
			order[params["order"]] = params["order_direction"]
		}

		filters := make(map[string]interface{})
		for k, v := range params {
			if strings.HasPrefix(k, "filter_") {
				k = k[7:]
				fieldName, fieldValue, errF := c.uriFilterToFilter(obj, k, v)
				if errF != nil {
					if errF.Op == "GetHelper" {
						w.WriteHeader(http.StatusInternalServerError)
						return
					} else {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}
				if fieldName != "" {
					filters[fieldName] = fieldValue
				}
			}
		}

		xobj, err1 := c.GetFromDB(newObjFunc, order, limit, offset, filters)
		if err1 != nil {
			if err1.Op == "ValidateFilters" {
				w.WriteHeader(http.StatusBadRequest)
				return
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		u := make(map[string]interface{})
		u["items"] = xobj
		j, err2 := json.Marshal(u)
		if err2 != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(j)
		return
	}

	objClone := newObjFunc()

	err := c.SetFromDB(objClone, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if c.GetModelIDValue(objClone) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	j, err2 := json.Marshal(objClone)
	if err2 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(j)
	return
}

func (c Controller) handleHTTPDelete(w http.ResponseWriter, r *http.Request, newObjFunc func() interface{}, id string) {
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(c.jsonError("id missing"))
		return
	}

	objClone := newObjFunc()

	err := c.SetFromDB(objClone, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c.GetModelIDValue(objClone) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = c.DeleteFromDB(objClone)
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
	}
	matched, err := regexp.Match(`^[0-9]+$`, []byte(xs[0]))
	if err != nil || !matched {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(c.jsonError("invalid id"))
		return "", false
	}
	return xs[0], true
}

func (c Controller) getParamsFromURI(uri string) map[string]string {
	o := make(map[string]string)
	xs := strings.SplitN(uri, "?", 2)
	if len(xs) < 2 || xs[1] == "" {
		return o
	}
	xp := strings.SplitN(xs[1], "&", -1)
	for _, p := range xp {
		pv := strings.SplitN(p, "=", 2)
		matched, err := regexp.Match(`^[0-9a-zA-Z_]+$`, []byte(pv[0]))
		if len(pv) == 1 || err != nil || !matched {
			continue
		}
		unesc, err := url.QueryUnescape(pv[1])
		if err != nil {
			continue
		}
		o[pv[0]] = unesc
	}
	return o
}

func (c Controller) jsonError(e string) []byte {
	return []byte(fmt.Sprintf("{\"err\":\"%s\"}", e))
}

func (c Controller) jsonID(id int64) []byte {
	return []byte(fmt.Sprintf("{\"id\":\"%d\"}", id))
}

func (c Controller) isKeyInMap(k string, m map[string]interface{}) bool {
	for _, key := range reflect.ValueOf(m).MapKeys() {
		if key.String() == k {
			return true
		}
	}
	return false
}

func (c Controller) uriFilterToFilter(obj interface{}, filterName string, filterValue string) (string, interface{}, *ControllerError) {
	h, err := c.getHelper(obj)
	if err != nil {
		return "", nil, &ControllerError{
			Op:  "GetHelper",
			Err: err,
		}
	}

	if h.dbCols[filterName] == "" {
		return "", nil, nil
	}

	val := reflect.ValueOf(obj).Elem()
	valueField := val.FieldByName(h.dbCols[filterName])
	if valueField.Type().Name() == "int" {
		filterInt, err := strconv.Atoi(filterValue)
		if err != nil {
			return "", nil, &ControllerError{
				Op:  "InvalidValue",
				Err: err,
			}
		}
		return h.dbCols[filterName], filterInt, nil
	}
	if valueField.Type().Name() == "int64" {
		filterInt64, err := strconv.ParseInt(filterValue, 10, 64)
		if err != nil {
			return "", nil, &ControllerError{
				Op:  "InvalidValue",
				Err: err,
			}
		}
		return h.dbCols[filterName], filterInt64, nil
	}
	if valueField.Type().Name() == "string" {
		return h.dbCols[filterName], filterValue, nil
	}

	return "", nil, nil
}
