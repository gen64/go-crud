package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	"github.com/gen64/go-crud"
	"net/http"

	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Syntax: f0x <config.json>\n")
		os.Exit(1)
	}

	cfg := NewConfig(os.Args[1])

	conn, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName))
	if err != nil {
		panic(err)
	}
	conn.SetConnMaxLifetime(time.Minute * 3)
	conn.SetMaxOpenConns(50)
	conn.SetMaxIdleConns(30)

	err = conn.Ping()
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	mc := crud.NewController(conn, "f0x_")

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

	http.HandleFunc("/users/", mc.GetHTTPHandler(func() interface{} {
		return &User{}
	}, "/users/"))
	log.Fatal(http.ListenAndServe(":9001", nil))
}
