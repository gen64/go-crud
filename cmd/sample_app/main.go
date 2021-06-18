package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "os"
	_ "time"

	"github.com/gen64/go-crud"

	"github.com/ory/dockertest/v3"

	_ "github.com/lib/pq"
)

const dbUser = "testing"
const dbPass = "secret"
const dbName = "testing"

const httpURI = "test_struct1s"
const httpPort = "32777"

var db *sql.DB
var pool *dockertest.Pool
var resource *dockertest.Resource

func main() {

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

	mc := crud.NewController(db, "dupa_")

	user := &User{}
	session := &Session{}

	models := []interface{}{
		user, session,
	}

	// Drop all structure
	err = mc.DropDBTables(models...)
	if err != nil {
		log.Printf("Error with DropAllDBTables: %s", err)
	}

	// Create structure
	err = mc.CreateDBTables(models...)
	if err != nil {
		log.Printf("Error with CreateTables: %s", err)
	}
	/*
		http.HandleFunc("/users/", mc.GetHTTPHandler(func() interface{} {
			return &User{}
		}, "/users/"))
		log.Fatal(http.ListenAndServe(":9001", nil))
	*/
}
