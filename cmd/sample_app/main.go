package main

import (
	"fmt"
	"log"
	"os"
	// "time"
	"net/http"
	"github.com/gen64/go-crudl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Syntax: f0x <config.json>\n")
		os.Exit(1)
	}

	cfg := NewConfig(os.Args[1])
	oDB := NewDB(cfg)

	conn, err := oDB.GetConn()
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	mc := &crudl.Controller{}
	mc.AttachDBConn(conn)
	mc.SetDBTablePrefix("f0x_")

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
	// Add data
	user := &User{
		Flags: 1+2+4,
		Email: "admin@sysg.io",
		CreatedAt: time.Now().Unix(),
	}
	_, err = mc.SaveToDB(user)
	if err != nil {
		log.Printf("Error with SaveToDB on user: %s", err)
	}

	user.Flags = 1+2+4+8
	_, err = mc.SaveToDB(user)
	if err != nil {
		log.Printf("Error with SaveToDB on user: %s", err)
	}

	session := &Session {
		Flags: 1+2,
		Key: "key",
		ExpiresAt: time.Now().Add(time.Duration(30) * time.Minute).Unix(),
		UserID: 0,
		User: user,
	}
	_, err = mc.SaveToDB(session)
	if err != nil {
		log.Printf("Error with SaveToDB on session: %s", err)
	}

	sessionCopy := &Session {}
	err = mc.SetFromDB(sessionCopy, "1")
	if err != nil {
		log.Printf("Error with SetFromDB on sessionCopy: %s", err)
	}

	err = mc.DeleteFromDB(session)
	if err != nil {
		log.Printf("Error with DeleteFromDB on session: %s", err)
	}

	err = mc.DeleteFromDB(user)
	if err != nil {
		log.Printf("Error with DeleteFromDB on user: %s", err)
	}

	sth := &Something {
		Email: "mg@gen64.pl",
		Age: 19,
		Price: 5,
		CurrencyRate: 20,
		PostCode: "32-600",
	}
	b, fields, err := mc.Validate(sth)
	log.Print(b)
	log.Print(fields)*/

	http.HandleFunc("/users/", mc.GetHTTPHandler(user, "/users/"))
	log.Fatal(http.ListenAndServe(":9001", nil))
}
