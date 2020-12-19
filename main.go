package main

import (
	"fmt"
	"log"
	"os"
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


	prod := &Product{}
	prod.Name = "Coffee 1kg"
	prod.Description = "Package of very good coffee beans"
	err = SaveMdlToDB(prod, conn)
	if err != nil {
		log.Fatal(err)
	}
}
