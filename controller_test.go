package crudl

import (
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"log"
	"testing"
	"fmt"
)

type TestStruct1 struct {
	ID           int64  `json:"teststruct_id"`
	Flags        int64  `json:"teststruct_flags"`
	Email        string `json:"email" crudl:"req lenmin:10 lenmax:255 email"`
	Age          int    `json:"age" crudl:"req valmin:18 valmax:120"`
	Price        int    `json:"price" crudl:"req valmin:5 valmax:3580"`
	CurrencyRate int    `json:"currency_rate" crudl:"req valmin:10 valmax:50004"`
	PostCode     string `json:"post_code" crudl:"req lenmin:6 regexp:^[0-9]{2}\\-[0-9]{3}$"`
}

var ts1 = &TestStruct1{}

const dbUser = "postgres"
const dbPass = "secret"
const dbName = "testing"

var db *sql.DB
var pool *dockertest.Pool
var resource *dockertest.Resource

func createDocker() {
	var err error
	if pool == nil {
		pool, err = dockertest.NewPool("")
		if err != nil {
			log.Fatalf("Could not connect to docker: %s", err)
		}
	}
	if resource == nil {
		resource, err = pool.Run("postgres", "13", []string{"POSTGRES_PASSWORD=" + dbPass, "POSTGRES_DB=" + dbName})
		if err != nil {
			log.Fatalf("Could not start resource: %s", err)
		}
	}
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", dbUser, dbPass, resource.GetPort("5432/tcp"), dbName))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		removeDocker();
		log.Fatalf("Could not connect to docker: %s", err)
	}
}

func removeDocker() {
	_ = pool.Purge(resource)
}

func Test2(t *testing.T) {
	createDocker()

	// docker ok, it's time to export this and

	removeDocker();
}
