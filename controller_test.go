package crudl

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"io/ioutil"
	"log"
	"net/http"
	"testing"
	"time"
)

type TestStruct1 struct {
	ID           int64  `json:"teststruct1_id"`
	Flags        int64  `json:"teststruct1_flags"`
	Email        string `json:"email" crudl:"req lenmin:10 lenmax:255 email"`
	Age          int    `json:"age" crudl:"req valmin:18 valmax:120"`
	Price        int    `json:"price" crudl:"req valmin:5 valmax:3580"`
	CurrencyRate int    `json:"currency_rate" crudl:"req valmin:10 valmax:50004"`
	PostCode     string `json:"post_code" crudl:"req lenmin:6 regexp:^[0-9]{2}\\-[0-9]{3}$"`
}

const dbUser = "testing"
const dbPass = "secret"
const dbName = "testing"

const httpURI = "test_struct1s"
const httpPort = "32777"

var db *sql.DB
var pool *dockertest.Pool
var resource *dockertest.Resource
var mc *Controller
var globalId int64
var cancelHTTPCtx context.CancelFunc

func TestGetModelIDInterface(t *testing.T) {
	ts1 := &TestStruct1{}
	ts1.ID = 123
	i := mc.GetModelIDInterface(ts1)
	if *(i.(*int64)) != int64(123) {
		log.Fatalf("GetModelIDInterface failed to get ID")
	}
}

func TestGetModelIDValue(t *testing.T) {
	ts1 := &TestStruct1{}
	ts1.ID = 123
	v := mc.GetModelIDValue(ts1)
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
	createDocker()
	createController()

	ts1 := &TestStruct1{}
	_ = mc.CreateDBTables(ts1)

	n, err := getTableName("f0x_test_struct1s")
	if err != nil || n != "f0x_test_struct1s" {
		t.Fatalf("CreateDBTables failed to create table for a struct: %s", err)
	}
}

func TestValidate(t *testing.T) {
}

func TestSaveToDB(t *testing.T) {
	ts1 := &TestStruct1{Flags: 4, Email: "test@example.com", Age: 37, Price: 1000, CurrencyRate: 14432, PostCode: "66-112"}
	_ = mc.SaveToDB(ts1)
	id, flags, email, age, price, rate, code, err := getRow()
	if err != nil || flags != 4 || email != "test@example.com" || age != 37 || price != 1000 || rate != 14432 || code != "66-112" {
		t.Fatalf("SaveToDB failed to insert struct to the table: %s", err)
	}

	ts1.Flags = 7
	ts1.Email = "test2@example.com"
	ts1.Age = 40
	ts1.Price = 2000
	ts1.CurrencyRate = 14411
	ts1.PostCode = "17-112"
	_ = mc.SaveToDB(ts1)

	flags, email, age, price, rate, code, err = getRowById(id)
	if err != nil || flags != 7 || email != "test2@example.com" || age != 40 || price != 2000 || rate != 14411 || code != "17-112" {
		t.Fatalf("SaveToDB failed to insert struct to the table: %s", err)
	}

	globalId = id
}

func TestSetFromDB(t *testing.T) {
	ts1 := &TestStruct1{}
	_ = mc.SetFromDB(ts1, fmt.Sprintf("%d", globalId))

	if ts1.ID == 0 || ts1.Flags != 7 || ts1.Email != "test2@example.com" || ts1.Age != 40 || ts1.Price != 2000 || ts1.CurrencyRate != 14411 || ts1.PostCode != "17-112" {
		t.Fatalf("SetFromDB failed to get struct from the table")
	}
}

func TestDeleteFromDB(t *testing.T) {
	ts1 := &TestStruct1{}
	_ = mc.SetFromDB(ts1, fmt.Sprintf("%d", globalId))
	_ = mc.DeleteFromDB(ts1)

	_, _, _, _, _, _, err := getRowById(globalId)
	if err != sql.ErrNoRows {
		t.Fatalf("DeleteFromDB failed to delete struct from the table")
	}
	if ts1.ID != 0 {
		t.Fatalf("DeleteFromDB failed to set ID to 0 on the struct")
	}
	globalId = 0
}

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

func TestDropDBTables(t *testing.T) {
	ts1 := &TestStruct1{}
	_ = mc.DropDBTables(ts1)

	n, err := getTableName("f0x_test_struct1s")
	if err != sql.ErrNoRows || n != "" {
		t.Fatalf("DropDBTables failed to drop table for a struct: %s", err)
	}

	removeDocker()
}

func createDocker() {
	var err error
	if pool == nil {
		pool, err = dockertest.NewPool("")
		if err != nil {
			log.Fatalf("Could not connect to docker: %s", err)
		}
	}
	if resource == nil {
		resource, err = pool.Run("postgres", "13", []string{"POSTGRES_PASSWORD=" + dbPass, "POSTGRES_USER=" + dbUser, "POSTGRES_DB=" + dbName})
		if err != nil {
			log.Fatalf("Could not start resource: %s", err)
		}
	}

	if db == nil {
		if err = pool.Retry(func() error {
			var err error
			db, err = sql.Open("postgres", fmt.Sprintf("host=localhost user=%s password=%s port=%s dbname=%s sslmode=disable", dbUser, dbPass, resource.GetPort("5432/tcp"), dbName))
			if err != nil {
				return err
			}
			return db.Ping()
		}); err != nil {
			log.Fatalf("Could not connect to docker: %s", err)
		}
	}
}

func removeDocker() {
	_ = pool.Purge(resource)
}

func createController() {
	if mc == nil {
		mc = NewController(db, "f0x_")
	}
}

func getTableName(tblName string) (string, error) {
	n := ""
	err := db.QueryRow("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_schema,table_name").Scan(&n)
	return n, err
}

func getRow() (int64, int64, string, int, int, int, string, error) {
	var id int64
	var flags int64
	var email string
	var age int
	var price int
	var currencyRate int
	var postCode string
	err := db.QueryRow("SELECT * FROM f0x_test_struct1s LIMIT 1").Scan(&id, &flags, &email, &age, &price, &currencyRate, &postCode)
	return id, flags, email, age, price, currencyRate, postCode, err
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

func createHTTPServer() {
	if cancelHTTPCtx == nil {
		var ctx context.Context
		ctx, cancelHTTPCtx = context.WithCancel(context.Background())
		go func(ctx context.Context) {
			go func() {
				http.HandleFunc("/"+httpURI+"/", mc.GetHTTPHandler(func() interface{} {
					return &TestStruct1{}
				}, "/"+httpURI+"/"))
				http.ListenAndServe(":"+httpPort, nil)
			}()
		}(ctx)
		time.Sleep(2 * time.Second)
	}
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
