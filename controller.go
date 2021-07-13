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
	"sort"
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

// Values for CRUD operations
const OpRead = 2
const OpUpdate = 4
const OpCreate = 8
const OpDelete = 16
const OpList = 32

// NewController returns new Controller object
func NewController(dbConn *sql.DB, tblPrefix string) *Controller {
	c := &Controller{
		dbConn:      dbConn,
		dbTblPrefix: tblPrefix,
	}
	c.modelHelpers = make(map[string]*Helper)
	return c
}

// DropDBTables drop tables in the database for specified objects (see
// DropDBTable for a single struct)
func (c Controller) DropDBTables(xobj ...interface{}) *ErrController {
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
func (c Controller) CreateDBTables(xobj ...interface{}) *ErrController {
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
func (c Controller) CreateDBTable(obj interface{}) *ErrController {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}

	_, err2 := c.dbConn.Exec(h.GetQueryCreateTable())
	if err2 != nil {
		return &ErrController{
			Op:  "DBQuery",
			Err: fmt.Errorf("Error executing DB query: %w", err2),
		}
	}
	return nil
}

// DropDBTable drops database table used to store specified type of objects. It
// just takes struct name, converts it to lowercase-with-underscore table name
// and executes "DROP TABLE" query using attached DB connection
func (c Controller) DropDBTable(obj interface{}) *ErrController {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}

	_, err2 := c.dbConn.Exec(h.GetQueryDropTable())
	if err2 != nil {
		return &ErrController{
			Op:  "DBQuery",
			Err: fmt.Errorf("Error executing DB query: %w", err2),
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
func (c Controller) SaveToDB(obj interface{}) *ErrController {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}

	b, invalidFields, err2 := c.Validate(obj, nil)
	if err2 != nil {
		return &ErrController{
			Op:  "Validate",
			Err: fmt.Errorf("Error when trying to validate: %w", err2),
		}
	}

	if !b {
		return &ErrController{
			Op: "Validate",
			Err: &ErrValidation{
				Fields: invalidFields,
			},
		}
	}

	var err3 error
	if c.GetModelIDValue(obj) != 0 {
		_, err3 = c.dbConn.Exec(h.GetQueryUpdateById(), append(c.GetModelFieldInterfaces(obj), c.GetModelIDInterface(obj))...)
	} else {
		err3 = c.dbConn.QueryRow(h.GetQueryInsert(), c.GetModelFieldInterfaces(obj)...).Scan(c.GetModelIDInterface(obj))
	}
	if err3 != nil {
		return &ErrController{
			Op:  "DBQuery",
			Err: fmt.Errorf("Error executing DB query: %w", err3),
		}
	}
	return nil
}

// SetFromDB sets object's fields with values from the database table with a
// specific id. If record does not exist in the database, all field values in
// the struct are zeroed
func (c Controller) SetFromDB(obj interface{}, id string) *ErrController {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return &ErrController{
			Op:  "IDToInt",
			Err: fmt.Errorf("Error converting string to int: %w", err),
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
		return &ErrController{
			Op:  "DBQuery",
			Err: fmt.Errorf("Error executing DB query: %w", err),
		}
	default:
		return nil
	}
}

// DeleteFromDB removes object from the database table and it does that only
// when ID field is set (greater than 0). Once deleted from the DB, all field
// values are zeroed
func (c Controller) DeleteFromDB(obj interface{}) *ErrController {
	h, err := c.getHelper(obj)
	if err != nil {
		return err
	}
	if c.GetModelIDValue(obj) == 0 {
		return nil
	}
	_, err2 := c.dbConn.Exec(h.GetQueryDeleteById(), c.GetModelIDInterface(obj))
	if err2 != nil {
		return &ErrController{
			Op:  "DBQuery",
			Err: fmt.Errorf("Error executing DB query: %w", err2),
		}
	}
	c.ResetFields(obj)
	return nil
}

// GetFromDB runs a select query on the database with specified filters, order,
// limit and offset and returns a list of objects
func (c Controller) GetFromDB(newObjFunc func() interface{}, order []string, limit int, offset int, filters map[string]interface{}) ([]interface{}, *ErrController) {
	obj := newObjFunc()
	h, err := c.getHelper(obj)
	if err != nil {
		return nil, err
	}

	b, invalidFields, err1 := c.Validate(obj, filters)
	if err1 != nil {
		return nil, &ErrController{
			Op:  "ValidateFilters",
			Err: fmt.Errorf("Error when trying to validate filters: %w", err1),
		}
	}

	if !b {
		return nil, &ErrController{
			Op: "ValidateFilters",
			Err: &ErrValidation{
				Fields: invalidFields,
			},
		}
	}

	var v []interface{}
	rows, err2 := c.dbConn.Query(h.GetQuerySelect(order, limit, offset, filters, nil, nil), c.GetFiltersInterfaces(filters)...)
	if err2 != nil {
		return nil, &ErrController{
			Op:  "DBQuery",
			Err: fmt.Errorf("Error executing DB query: %w", err2),
		}
	}
	defer rows.Close()

	for rows.Next() {
		newObj := newObjFunc()
		err3 := rows.Scan(append(append(make([]interface{}, 0), c.GetModelIDInterface(newObj)), c.GetModelFieldInterfaces(newObj)...)...)
		if err3 != nil {
			return nil, &ErrController{
				Op:  "DBQueryRowsScan",
				Err: fmt.Errorf("Error scanning DB query row: %w", err3),
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

	if len(mf) > 0 {
		sorted := []string{}
		for k := range mf {
			sorted = append(sorted, k)
		}
		sort.Strings(sorted)

		for _, v := range sorted {
			xi = append(xi, mf[v])
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
// server. It creates a CRUD endpoint for PUT, GET and DELETE methods.
// It's important to pass "uri" argument same as the one that the handler is
// attached to.
func (c Controller) GetHTTPHandler(uri string, newObjFunc func() interface{}, newObjCreateFunc func() interface{}, newObjReadFunc func() interface{}, newObjUpdateFunc func() interface{}, newObjDeleteFunc func() interface{}, newObjListFunc func() interface{}) func(http.ResponseWriter, *http.Request) {
	c.initHelpersForHTTPHandler(newObjFunc, newObjCreateFunc, newObjReadFunc, newObjUpdateFunc, newObjDeleteFunc, newObjListFunc)

	fn := func(w http.ResponseWriter, r *http.Request) {
		id, b := c.getIDFromURI(r.RequestURI[len(uri):], w)
		if !b {
			return
		}
		if r.Method == http.MethodPut && id == "" {
			c.handleHTTPPut(w, r, newObjCreateFunc, id)
			return
		}
		if r.Method == http.MethodPut && id != "" {
			c.handleHTTPPut(w, r, newObjUpdateFunc, id)
			return
		}
		if r.Method == http.MethodGet && id != "" {
			c.handleHTTPGet(w, r, newObjReadFunc, id)
			return
		}
		if r.Method == http.MethodGet && id == "" {
			c.handleHTTPGet(w, r, newObjListFunc, id)
			return
		}
		if r.Method == http.MethodDelete && id != "" {
			c.handleHTTPDelete(w, r, newObjFunc, id)
			return
		}

		w.WriteHeader(http.StatusBadRequest)
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

	// Check required fields only when we are not validating filters
	if filters == nil {
		for k := range h.fieldsRequired {
			valueField := val.FieldByName(k)
			canBeZero := false
			if len(h.fieldsValueNotNil[k]) == 2 && (h.fieldsValueNotNil[k][0] || h.fieldsValueNotNil[k][1]) {
				canBeZero = true
			}
			if !c.validateFieldRequired(valueField, canBeZero) {
				failedFields = append(failedFields, k)
				b = false
			}
		}
	}
	for k, v := range h.fieldsLength {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		if filters != nil && reflect.ValueOf(filters[k]).Type().Name() != val.FieldByName(k).Type().Name() {
			failedFields = append(failedFields, k)
			b = false
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
		} else {
			valueField = val.FieldByName(k)
		}
		if !c.validateFieldLength(valueField, v) {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	for k, v := range h.fieldsValue {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		if filters != nil && reflect.ValueOf(filters[k]).Type().Name() != val.FieldByName(k).Type().Name() {
			failedFields = append(failedFields, k)
			b = false
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
		} else {
			valueField = val.FieldByName(k)
		}
		minIsZero := false
		maxIsZero := false
		if len(h.fieldsValueNotNil[k]) == 2 {
			minIsZero = h.fieldsValueNotNil[k][0]
			maxIsZero = h.fieldsValueNotNil[k][1]
		}
		if !c.validateFieldValue(valueField, v, minIsZero, maxIsZero) {
			failedFields = append(failedFields, k)
			b = false
		}

	}
	for k := range h.fieldsEmail {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		if filters != nil && reflect.ValueOf(filters[k]).Type().Name() != val.FieldByName(k).Type().Name() {
			failedFields = append(failedFields, k)
			b = false
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
		} else {
			valueField = val.FieldByName(k)
		}
		if !c.validateFieldEmail(valueField) {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	for k, v := range h.fieldsRegExp {
		if filters != nil && !c.isKeyInMap(k, filters) {
			continue
		}
		if filters != nil && reflect.ValueOf(filters[k]).Type().Name() != val.FieldByName(k).Type().Name() {
			failedFields = append(failedFields, k)
			b = false
		}
		var valueField reflect.Value
		if filters != nil {
			valueField = reflect.ValueOf(filters[k])
		} else {
			valueField = val.FieldByName(k)
		}
		if !c.validateFieldRegExp(valueField, v) {
			failedFields = append(failedFields, k)
			b = false
		}
	}
	return b, failedFields, nil
}

// validateFieldRequired checks if field that is required has a value
func (c *Controller) validateFieldRequired(valueField reflect.Value, canBeZero bool) bool {
	if valueField.Type().Name() == "string" && valueField.String() == "" {
		return false
	}
	if valueField.Type().Name() == "int" && valueField.Int() == 0 && !canBeZero {
		return false
	}
	if valueField.Type().Name() == "int64" && valueField.Int() == 0 && !canBeZero {
		return false
	}
	return true
}

// validateFieldLength checks string field's length
func (c *Controller) validateFieldLength(valueField reflect.Value, length [2]int) bool {
	if valueField.Type().Name() != "string" {
		return true
	}
	if length[0] > -1 && len(valueField.String()) < length[0] {
		return false
	}
	if length[1] > -1 && len(valueField.String()) > length[1] {
		return false
	}
	return true
}

// validateFieldValue checks int field's value
func (c *Controller) validateFieldValue(valueField reflect.Value, value [2]int, minIsZero bool, maxIsZero bool) bool {
	if valueField.Type().Name() != "int" && valueField.Type().Name() != "int64" {
		return true
	}
	// Minimal value is 0 only when canBeZero is true; otherwise it's not defined
	if ((minIsZero && value[0] == 0) || value[0] != 0) && valueField.Int() < int64(value[0]) {
		return false
	}
	// Maximal value is 0 only when canBeZero is true; otherwise it's not defined
	if ((maxIsZero && value[1] == 0) || value[1] != 0) && valueField.Int() > int64(value[1]) {
		return false
	}
	return true
}

// validateFieldEmail checks if email field has a valid value
func (c *Controller) validateFieldEmail(valueField reflect.Value) bool {
	if valueField.Type().Name() != "string" {
		return true
	}
	var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegex.MatchString(valueField.String())
}

// validateFieldRegExp checks if string field's value matches the regular expression
func (c *Controller) validateFieldRegExp(valueField reflect.Value, re *regexp.Regexp) bool {
	if valueField.Type().Name() != "string" {
		return true
	}
	if !re.MatchString(valueField.String()) {
		return false
	}
	return true
}

// initHelpers creates all the Helper objects. For HTTP endpoints, it is
// necessary to create these first
func (c *Controller) initHelpersForHTTPHandler(newObjFunc func() interface{}, newObjCreateFunc func() interface{}, newObjReadFunc func() interface{}, newObjUpdateFunc func() interface{}, newObjDeleteFunc func() interface{}, newObjListFunc func() interface{}) *ErrController {
	obj := newObjFunc()
	v := reflect.ValueOf(obj)
	i := reflect.Indirect(v)
	s := i.Type()
	forceName := s.Name()

	h, cErr := c.getHelper(obj)
	if cErr != nil {
		return cErr
	}

	cErr = c.initHelper(newObjCreateFunc, forceName, h)
	if cErr != nil {
		return cErr
	}
	cErr = c.initHelper(newObjReadFunc, forceName, h)
	if cErr != nil {
		return cErr
	}
	cErr = c.initHelper(newObjUpdateFunc, forceName, h)
	if cErr != nil {
		return cErr
	}
	cErr = c.initHelper(newObjDeleteFunc, forceName, h)
	if cErr != nil {
		return cErr
	}
	cErr = c.initHelper(newObjListFunc, forceName, h)
	if cErr != nil {
		return cErr
	}

	return nil
}

func (c *Controller) initHelper(newObjFunc func() interface{}, forceName string, sourceHelper *Helper) *ErrController {
	if newObjFunc == nil {
		return nil
	}

	obj := newObjFunc()
	v := reflect.ValueOf(obj)
	i := reflect.Indirect(v)
	s := i.Type()
	n := s.Name()
	h := NewHelper(obj, c.dbTblPrefix, forceName, sourceHelper)
	if h.Err() != nil {
		return &ErrController{
			Op:  "InitHelperWithForcedName",
			Err: fmt.Errorf("Error initialising Helper with forced name: %w", h.Err()),
		}
	}
	c.modelHelpers[n] = h
	return nil
}

// getHelper returns a special Helper instance which reflects the struct type
// to get SQL queries, validation etc.
func (c *Controller) getHelper(obj interface{}) (*Helper, *ErrController) {
	v := reflect.ValueOf(obj)
	i := reflect.Indirect(v)
	s := i.Type()
	n := s.Name()
	if c.modelHelpers[n] == nil {
		h := NewHelper(obj, c.dbTblPrefix, "", nil)
		if h.Err() != nil {
			return nil, &ErrController{
				Op:  "GetHelper",
				Err: fmt.Errorf("Error getting Helper: %w", h.Err()),
			}
		}
		c.modelHelpers[n] = h
	}
	return c.modelHelpers[n], nil
}

func (c Controller) handleHTTPPut(w http.ResponseWriter, r *http.Request, newObjFunc func() interface{}, id string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.writeErrText(w, http.StatusInternalServerError, "cannot_read_request_body")
		return
	}

	objClone := newObjFunc()

	if id != "" {
		err2 := c.SetFromDB(objClone, id)
		if err2 != nil {
			c.writeErrText(w, http.StatusInternalServerError, "cannot_get_from_db")
			return
		}
		if c.GetModelIDValue(objClone) == 0 {
			c.writeErrText(w, http.StatusNotFound, "not_found_in_db")
			return
		}
	} else {
		c.ResetFields(objClone)
	}

	err = json.Unmarshal(body, objClone)
	if err != nil {
		c.writeErrText(w, http.StatusBadRequest, "invalid_json")
		return
	}

	b, _, err := c.Validate(objClone, nil)
	if !b || err != nil {
		c.writeErrText(w, http.StatusBadRequest, "validation_failed")
		return
	}

	err2 := c.SaveToDB(objClone)
	if err2 != nil {
		c.writeErrText(w, http.StatusInternalServerError, "cannot_save_to_db")
		return
	}

	c.writeOK(w, http.StatusOK, map[string]interface{}{
		"id": c.GetModelIDValue(objClone),
	})
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

		order := []string{}
		if params["order"] != "" {
			order = append(order, params["order"])
			order = append(order, params["order_direction"])
		}

		filters := make(map[string]interface{})
		for k, v := range params {
			if strings.HasPrefix(k, "filter_") {
				k = k[7:]
				fieldName, fieldValue, errF := c.uriFilterToFilter(obj, k, v)
				if errF != nil {
					if errF.Op == "GetHelper" {
						c.writeErrText(w, http.StatusInternalServerError, "get_helper")
						return
					} else {
						c.writeErrText(w, http.StatusBadRequest, "invalid_filter")
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
				c.writeErrText(w, http.StatusBadRequest, "invalid_filter_value")
				return
			} else {
				c.writeErrText(w, http.StatusInternalServerError, "cannot_get_from_db")
				return
			}
		}

		c.writeOK(w, http.StatusOK, map[string]interface{}{
			"items": xobj,
		})

		return
	}

	objClone := newObjFunc()

	err := c.SetFromDB(objClone, id)
	if err != nil {
		c.writeErrText(w, http.StatusInternalServerError, "cannot_get_from_db")
		return
	}

	if c.GetModelIDValue(objClone) == 0 {
		c.writeErrText(w, http.StatusNotFound, "not_found_in_db")
		return
	}

	c.writeOK(w, http.StatusOK, map[string]interface{}{
		"item": objClone,
	})
}

func (c Controller) handleHTTPDelete(w http.ResponseWriter, r *http.Request, newObjFunc func() interface{}, id string) {
	if id == "" {
		c.writeErrText(w, http.StatusBadRequest, "invalid_id")
		return
	}

	objClone := newObjFunc()

	err := c.SetFromDB(objClone, id)
	if err != nil {
		c.writeErrText(w, http.StatusInternalServerError, "cannot_get_from_db")
		return
	}
	if c.GetModelIDValue(objClone) == 0 {
		c.writeErrText(w, http.StatusNotFound, "not_found_in_db")
		return
	}

	err = c.DeleteFromDB(objClone)
	if err != nil {
		c.writeErrText(w, http.StatusInternalServerError, "cannot_delete_from_db")
		return
	}

	c.writeOK(w, http.StatusOK, map[string]interface{}{
		"id": id,
	})
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

func (c Controller) uriFilterToFilter(obj interface{}, filterName string, filterValue string) (string, interface{}, *ErrController) {
	h, err := c.getHelper(obj)
	if err != nil {
		return "", nil, &ErrController{
			Op:  "GetHelper",
			Err: fmt.Errorf("Error getting Helper: %w", err),
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
			return "", nil, &ErrController{
				Op:  "InvalidValue",
				Err: fmt.Errorf("Error converting string to int: %w", err),
			}
		}
		return h.dbCols[filterName], filterInt, nil
	}
	if valueField.Type().Name() == "int64" {
		filterInt64, err := strconv.ParseInt(filterValue, 10, 64)
		if err != nil {
			return "", nil, &ErrController{
				Op:  "InvalidValue",
				Err: fmt.Errorf("Error converting string to int64: %w", err),
			}
		}
		return h.dbCols[filterName], filterInt64, nil
	}
	if valueField.Type().Name() == "string" {
		return h.dbCols[filterName], filterValue, nil
	}

	return "", nil, nil
}

func (c Controller) writeErrText(w http.ResponseWriter, status int, errText string) {
	r := NewHTTPResponse(0, errText)
	j, err := json.Marshal(r)
	w.WriteHeader(status)
	if err == nil {
		w.Write(j)
	}
}

func (c Controller) writeOK(w http.ResponseWriter, status int, data map[string]interface{}) {
	r := NewHTTPResponse(1, "")
	r.Data = data
	j, err := json.Marshal(r)
	w.WriteHeader(status)
	if err == nil {
		w.Write(j)
	}
}
