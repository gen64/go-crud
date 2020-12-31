package crudl

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"regexp"
	"net/http"
	"strings"
	"io/ioutil"
	"log"
	"encoding/json"
	"runtime"
)

type Controller struct {
	dbConn *sql.DB
	dbTablePrefix string
	modelHelpers map[string]*Helper
}

func (mc *Controller) SetDBTablePrefix(p string) {
	mc.dbTablePrefix = p
}

func (mc *Controller) AttachDBConn(db *sql.DB) {
	mc.dbConn = db
}

func (mc *Controller) GetHelper(m interface{}) (*Helper, error) {
	v := reflect.ValueOf(m)
	i := reflect.Indirect(v)
	s := i.Type()
	n := s.Name()

	if mc.modelHelpers == nil {
		mc.modelHelpers = make(map[string]*Helper)
	}
	if mc.modelHelpers[n] == nil {
		h, err := NewHelper(m)
		if err != nil {
			return nil, fmt.Errorf("error with NewHelper in GetHelper: %s", err)
		}
		mc.modelHelpers[n] = h
	}
	return mc.modelHelpers[n], nil
}

func (mc *Controller) DropDBTables(xm ...interface{}) error {
	for _, m := range xm {
		err := mc.DropDBTable(m)
		if err != nil {
			return fmt.Errorf("error with DropDBTable: %s", err)
		}
	}
	return nil
}

func (mc *Controller) CreateDBTables(xm ...interface{}) error {
	for _, m := range xm {
		err := mc.CreateDBTable(m)
		if err != nil {
			return fmt.Errorf("error with CreateDBTable: %s", err)
		}
	}
	return nil
}

func (mc *Controller) Validate(m interface{}) (bool, []int, error) {
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

func (mc *Controller) PopulateLinks(m interface{}) {
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

func (mc *Controller) CreateDBTable(m interface{}) error {
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

func (mc *Controller) DropDBTable(m interface{}) error {
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

func (mc *Controller) SaveToDB(m interface{}) (error) {
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

func (mc *Controller) SetFromDB(m interface{}, id string) error {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("error with strconv.Atoi in SetFromDB: %s", err)
	}

	h, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in Validate: %s", err)
	}

	err = mc.dbConn.QueryRow(h.GetQuerySelectById(), int64(idInt)).Scan(append(append(make([]interface{}, 0), mc.GetModelIDInterface(m)), mc.GetModelFieldInterfaces(m)...)...)
	switch {
	case err == sql.ErrNoRows:
		mc.ResetFields(m)
		return nil
	case err != nil:
		return fmt.Errorf("error with db.QueryRow in SetFromDB: %s", err)
	default:
		return nil
	}
	return nil
}

func (mc *Controller) DeleteFromDB(m interface{}) error {
	h, err := mc.GetHelper(m)
	if err != nil {
		return fmt.Errorf("error with GetHelper in Validate: %s", err)
	}
	if mc.GetModelIDValue(m) == 0 {
		return nil
	}
	_, err = mc.dbConn.Exec(h.GetQueryDeleteById(), mc.GetModelIDInterface(m))
	if err != nil {
		return fmt.Errorf("error with db.Exec in DeleteFromDB: %s", err)
	}
	mc.ResetFields(m)
	return nil
}

func (mc *Controller) GetModelIDInterface(u interface{}) interface{} {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Addr().Interface()
}

func (mc *Controller) GetModelIDValue(u interface{}) int64 {
	return reflect.ValueOf(u).Elem().FieldByName("ID").Int()
}

func (mc *Controller) GetModelFieldInterfaces(u interface{}) []interface{} {
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
		if valueField.Kind() == reflect.String {
			valueField.SetString("")
		}
	}
}

func (mc *Controller) GetHTTPHandler(u interface{}, uri string) func(http.ResponseWriter, *http.Request) {
	fn := func(w http.ResponseWriter, r *http.Request) {
		PrintMemUsage()
		id, b := mc.getIDFromURI(r.RequestURI[len(uri):], w)
		if !b {
			return
		}
		if r.Method == http.MethodPut {
			mc.HandleHTTPPut(w, r, u, id)
			return
		}
		if r.Method == http.MethodGet {
			mc.HandleHTTPGet(w, r, u, id)
			return
		}
		if r.Method == http.MethodDelete {
			mc.HandleHTTPDelete(w, r, u, id)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	return fn
}

func (mc *Controller) HandleHTTPPut(w http.ResponseWriter, r *http.Request, u interface{}, id string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if id != "" {
		err = mc.SetFromDB(u, id)
		if err != nil {
			log.Print(err)
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

	err = mc.SaveToDB(u)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(mc.jsonID(mc.GetModelIDValue(u)))
	return
}

func (mc *Controller) HandleHTTPGet(w http.ResponseWriter, r *http.Request, u interface{}, id string) {
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

	j, err := json.Marshal(u)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(j)
	return
}

func (mc *Controller) HandleHTTPDelete(w http.ResponseWriter, r *http.Request, u interface{}, id string) {
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

func PrintMemUsage() {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
        fmt.Printf("Alloc = %v B", m.Alloc)
        fmt.Printf("\tTotalAlloc = %v B", m.TotalAlloc)
        fmt.Printf("\tSys = %v B", m.Sys)
        fmt.Printf("\tNumGC = %v", m.NumGC)
		fmt.Printf("\tMallocs = %v", m.Mallocs)
		fmt.Printf("\tFrees = %v", m.Frees)
		fmt.Printf("\tHeapObjects = %v\n", m.HeapObjects)
}
