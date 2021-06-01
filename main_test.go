package crud

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


// Global vars used across all the tests
var dbUser = "gocrudtest"
var dbPass = "secret"
var dbName = "gocrud"
var dbConn *sql.DB

var dockerPool *dockertest.Pool
var dockerResource *dockertest.Resource

var httpPort = "32777"
var httpCancelCtx context.CancelFunc
var httpURI = "/v1/testobjects/"

var testController *Controller

var testStructNewFunc func() interface{}
var testStructObj *TestStruct


// Test struct for all the tests
type TestStruct struct {
	ID              int64  `json:"teststruct1_id"`
	Flags           int64  `json:"flags"`

	// Test email validation
	PrimaryEmail    string `json:"email" crud:"req"`
	EmailSecondary  string `json:"email3" crud:"req email"`
	
	// Test length validation
	FirstName       string `json:"first_name" crud:"req lenmin:2 lenmax:30"`
	LastName        string `json:"last_name" crud:"req lenminzero lenmin:0 lenmax:255"`
	
	// Test int value validation
	Age             int    `json:"age" crud:"req valmin:18 valmax:120"`
	Price           int    `json:"price" crud:"valminzero valmin:0 valmax:999"`
	
	// Test regular expression
	PostCode        string `json:"post_code" crud:"req lenmin:6 regexp:^[0-9]{2}\\-[0-9]{3}$"`
	PostCode2       string `json:"post_code2" crud:"lenmin:6" crud_regexp:"^[0-9]{2}\\-[0-9]{3}$"`

	// Test HTTP endpoint tags
	Password        string `json:"password" crud:"noread noupdate nocreate nolist"`
	CreatedByUserID int64  `json:"created_by_user_id" crud:"nocreate" crud_val:55`

	// Test unique tag
	Key             string `json:"key" crud:"req uniq lenmin:30 lenmax:255"`
}


func TestMain(m *testing.M) {
	createDocker()
	createController()
	createHTTPServer()
	os.Exit(m.Run())
}

func createDocker() {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err = pool.Run("postgres", "13", []string{"POSTGRES_PASSWORD=" + dbPass, "POSTGRES_USER=" + dbUser, "POSTGRES_DB=" + dbName})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
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

func createController() {
	testController = NewController(db, "gen64_")
	testStructNewFunc = func() interface{} {
		return &TestStruct{}
	}
	testStructObj = testStructNewFunc()
}

func createHTTPServer() {
	var ctx context.Context
	ctx, httpCancelCtx = context.WithCancel(context.Background())
	go func(ctx context.Context) {
		go func() {
			http.HandleFunc("/"+httpURI+"/", testController.GetHTTPHandler(testStructNewFunc, "/"+httpURI+"/"))
			http.ListenAndServe(":"+httpPort, nil)
		}()
	}(ctx)
	time.Sleep(2 * time.Second)
}
