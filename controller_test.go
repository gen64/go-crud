package crud

import (
	_ "bytes"
	_ "context"
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/ory/dockertest/v3"
	_ "io/ioutil"
	"log"
	_ "net/http"
	"testing"
	_ "time"
)

func TestGetModelIDInterface(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	ts.ID = 123
	i := testController.GetModelIDInterface(ts)
	if *(i.(*int64)) != int64(123) {
		log.Fatalf("GetModelIDInterface failed to get ID")
	}
}

func TestGetModelIDValue(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	ts.ID = 123
	v := testController.GetModelIDValue(ts)
	if v != 123 {
		log.Fatalf("GetModelIDValue failed to get ID")
	}
}

func TestGetModelFieldInterfaces(t *testing.T) {
	// TODO
}

func TestResetFields(t *testing.T) {
	// TODO
}

func TestCreateDBTables(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	_ = testController.CreateDBTables(ts)

	n, err := getTableName("gen64_test_structs")
	if n != "gen64_test_structs" {
		t.Fatalf("CreateDBTables failed to create table for a struct")
	}
	if err != nil {
		t.Fatalf("CreateDBTables failed to create table for a struct: %s", err.Error())
	}
}

func TestValidate(t *testing.T) {
	// TODO
}

func TestSaveToDB(t *testing.T) {
	ts := getTestStructWithData()

	err := testController.SaveToDB(ts)
	if err != nil {
		t.Fatalf("SaveToDB failed to insert struct to the table: %s", err.Op)
	}
	id, flags, primaryEmail, emailSecondary, firstName, lastName, age, price, postCode, postCode2, password, createdByUserID, key, err2 := getRow()
	if err2 != nil {
		t.Fatalf("SaveToDB failed to insert struct to the table: %s", err.Error())
	}
	if id == 0 || flags != ts.Flags || primaryEmail != ts.PrimaryEmail || emailSecondary != ts.EmailSecondary || firstName != ts.FirstName || lastName != ts.LastName || age != ts.Age || price != ts.Price || postCode != ts.PostCode || postCode2 != ts.PostCode2 || createdByUserID != ts.CreatedByUserID || key != ts.Key {
		t.Fatalf("SaveToDB failed to insert struct to the table")
	}
	if password == "" {
		t.Fatalf("SaveToDB failed to insert struct to the table and ignore 'nocreate' field")
	}

	ts.Flags = 7
	ts.PrimaryEmail = "primary1@gen64.net"
	ts.EmailSecondary = "secondary2@gen64.net"
	ts.FirstName = "Johnny"
	ts.LastName = "Smithsy"
	ts.Age = 50
	ts.Price = 222
	ts.PostCode = "22-222"
	ts.PostCode2 = "33-333"
	ts.Password = "xxx"
	ts.CreatedByUserID = 7
	ts.Key = "reallyunique"
	_ = testController.SaveToDB(ts)

	flags, primaryEmail, emailSecondary, firstName, lastName, age, price, postCode, postCode2, password, createdByUserID, key, err2 = getRowById(id)
	if err2 != nil {
		t.Fatalf("SaveToDB failed to update struct in the table: %s", err.Error())
	}
	if id == 0 || flags != ts.Flags || primaryEmail != ts.PrimaryEmail || emailSecondary != ts.EmailSecondary || firstName != ts.FirstName || lastName != ts.LastName || age != ts.Age || price != ts.Price || postCode != ts.PostCode || postCode2 != ts.PostCode2 || createdByUserID != ts.CreatedByUserID || key != ts.Key || password != ts.Password {
		t.Fatalf("SaveToDB failed to update struct to the table")
	}
}

func TestSetFromDB(t *testing.T) {
	ts := getTestStructWithData()
	err := testController.SaveToDB(ts)
	if err != nil {
		t.Fatalf("SaveToDB in TestSetFromDB failed to insert struct to the table: %s", err.Op)
	}

	ts2 := testStructNewFunc().(*TestStruct)
	err = testController.SetFromDB(ts2, fmt.Sprintf("%d", ts.ID))
	if err != nil {
		t.Fatalf("SetFromDB failed to get data: %s", err.Op)
	}

	if !areTestStructObjectSame(ts, ts2) {
		t.Fatalf("SetFromDB failed to set struct with data: %s", err.Op)
	}
}

func TestDeleteFromDB(t *testing.T) {
	ts := getTestStructWithData()
	err := testController.SaveToDB(ts)
	if err != nil {
		t.Fatalf("SaveToDB in TestDeleteFromDB failed to insert struct to the table: %s", err.Op)
	}
	err = testController.DeleteFromDB(ts)
	if err != nil {
		t.Fatalf("DeleteFromDB failed to remove: %s", err.Op)
	}

	cnt, err2 := getRowCntById(ts.ID)
	if err2 != nil {
		t.Fatalf("DeleteFromDB failed to delete struct from the table")
	}
	if cnt > 0 {
		t.Fatalf("DeleteFromDB failed to delete struct from the table")
	}
	if ts.ID != 0 {
		t.Fatalf("DeleteFromDB failed to set ID to 0 on the struct")
	}
}

func TestGetFromDB(t *testing.T) {
	for i := 1; i < 51; i++ {
		ts := getTestStructWithData()
		ts.ID = 0
		ts.Age = 30 + i
		testController.SaveToDB(ts)
	}

	testStructs, err := testController.GetFromDB(testStructNewFunc, []string{"Age", "asc", "Price", "asc"}, 10, 20, map[string]interface{}{"Price": 444, "PrimaryEmail": "primary@gen64.net"})
	if err != nil {
		t.Fatalf("GetFromDB failed to return list of objects: %s", err.Op)
	}
	if len(testStructs) != 10 {
		t.Fatalf("GetFromDB failed to return list of objects, want %v, got %v", 10, len(testStructs))
	}
	if testStructs[2].(*TestStruct).Age != 52 {
		t.Fatalf("GetFromDB failed to return correct list of objects, want %v, got %v", 52, testStructs[2].(*TestStruct).Age)
	}
}

/*
func TestHTTPHandlerPutMethodForValidation(t *testing.T) {
}

func TestHTTPHandlerPutMethodForCreating(t *testing.T) {
	createHTTPServer()

	j := `{
		"teststruct1_flags": 4,
		"email": "test@example.com",
		"age": 37,
		"price": 1000,
		"currency_rate": 14432,
		"post_code": "66-112"
	}`
	makePUTInsertRequest(j, t)

	id, flags, email, age, price, rate, code, err := getRow()
	if err != nil || flags != 4 || email != "test@example.com" || age != 37 || price != 1000 || rate != 14432 || code != "66-112" || id == 0 {
		t.Fatalf("PUT method failed to insert struct to the table: %s", err)
	}

	// TODO: Check if response contains JSON with key 'id'

	globalId = id
}

func TestHTTPHandlerPutMethodForUpdating(t *testing.T) {
	j := `{
		"teststruct1_flags": 7,
		"email": "test2@example.com",
		"age": 40,
		"price": 2000,
		"currency_rate": 12222,
		"post_code": "22-112"
	}`
	makePUTUpdateRequest(j, t)

	flags, email, age, price, rate, code, err := getRowById(globalId)
	if err != nil || flags != 7 || email != "test2@example.com" || age != 40 || price != 2000 || rate != 12222 || code != "22-112" {
		t.Fatalf("PUT method failed to update struct in the table: %s", err)
	}
}

func TestHTTPHandlerGetMethodOnExisting(t *testing.T) {
	resp := makeGETRequest(t)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET method returned wrong status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET method failed")
	}
	ts1 := &TestStruct1{}
	err = json.Unmarshal(body, ts1)
	if err != nil {
		t.Fatalf("GET method failed to return unmarshable JSON")
	}
	if ts1.Age != 40 {
		t.Fatalf("GET method returned invalid values")
	}
}

func TestHTTPHandlerDeleteMethod(t *testing.T) {
	makeDELETERequest(t)

	_, _, _, _, _, _, err := getRowById(globalId)
	if err != sql.ErrNoRows {
		t.Fatalf("DELETE handler failed to delete struct from the table")
	}
}

func TestHTTPHandlerGetMethodOnNonExisting(t *testing.T) {
	resp := makeGETRequest(t)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET method returned wrong status code, want %d, got %d", http.StatusNotFound, resp.StatusCode)
	}

	globalId = 0

	cancelHTTPCtx()
}

func TestHTTPHandlerGetMethodWithoutID(t *testing.T) {
}

func TestDropDBTables(t *testing.T) {
	ts1 := &TestStruct1{}
	_ = mc.DropDBTables(ts1)

	n, err := getTableName("f0x_test_struct1s")
	if err != sql.ErrNoRows || n != "" {
		t.Fatalf("DropDBTables failed to drop table for a struct: %s", err)
	}

	removeDocker()
}


func removeDocker() {
	_ = pool.Purge(resource)
}



func getRowById(id int64) (int64, string, int, int, int, string, error) {
	var flags int64
	var email string
	var age int
	var price int
	var currencyRate int
	var postCode string
	err := db.QueryRow("SELECT test_struct1_flags, email, age, price, currency_rate, post_code FROM f0x_test_struct1s WHERE test_struct1_id = $1", int64(id)).Scan(&flags, &email, &age, &price, &currencyRate, &postCode)
	return flags, email, age, price, currencyRate, postCode, err
}

func makePUTInsertRequest(j string, t *testing.T) {
	req, err := http.NewRequest("PUT", "http://localhost:"+httpPort+"/"+httpURI+"/", bytes.NewReader([]byte(j)))
	if err != nil {
		t.Fatalf("PUT method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("PUT method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT method returned wrong status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func makePUTUpdateRequest(j string, t *testing.T) {
	req, err := http.NewRequest("PUT", "http://localhost:"+httpPort+"/"+httpURI+"/"+fmt.Sprintf("%d", globalId), bytes.NewReader([]byte(j)))
	if err != nil {
		t.Fatalf("PUT method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("PUT method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT method returned wrong status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func makeDELETERequest(t *testing.T) {
	req, err := http.NewRequest("DELETE", "http://localhost:"+httpPort+"/"+httpURI+"/"+fmt.Sprintf("%d", globalId), bytes.NewReader([]byte("")))
	if err != nil {
		t.Fatalf("DELETE method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("DELETE method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE method returned wrong status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

func makeGETRequest(t *testing.T) *http.Response {
	req, err := http.NewRequest("GET", "http://localhost:"+httpPort+"/"+httpURI+"/"+fmt.Sprintf("%d", globalId), bytes.NewReader([]byte("")))
	if err != nil {
		t.Fatalf("GET method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("GET method failed on HTTP server with handler from GetHTTPHandler: %s", err)
	}
	return resp
}
*/
