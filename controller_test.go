package crudl

import (
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"log"
	"testing"
	"fmt"
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

var db *sql.DB
var pool *dockertest.Pool
var resource *dockertest.Resource
var mc *Controller
var globalId int64

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
		time.Sleep(10*time.Second)
		db, err = sql.Open("postgres", fmt.Sprintf("host=localhost user=%s password=%s port=%s dbname=%s sslmode=disable", dbUser, dbPass, resource.GetPort("5432/tcp"), dbName))
		if err != nil {
			log.Fatalf("Could not connect to postgres docker: %s", err)
		}

		err = db.Ping()
		if err != nil {
			log.Fatalf("Could not start postgres docker: %s", err)
		}
	}
}

func removeDocker() {
	_ = pool.Purge(resource)
}

func createController() {
	if mc == nil {
		mc = NewController()
		mc.AttachDBConn(db)
		mc.SetDBTablePrefix("f0x_")
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

func TestSaveToDB(t *testing.T) {
	ts1 := &TestStruct1{}
	ts1.Flags = 4
	ts1.Email = "test@example.com"
	ts1.Age = 37
	ts1.Price = 1000
	ts1.CurrencyRate = 14432
	ts1.PostCode = "66-112"
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

func TestDropDBTables(t *testing.T) {
	ts1 := &TestStruct1{}
	_ = mc.DropDBTables(ts1)

	n, err := getTableName("f0x_test_struct1s")
	if err != sql.ErrNoRows || n != "" {
		t.Fatalf("DropDBTables failed to drop table for a struct: %s", err)
	}

	removeDocker();
}
