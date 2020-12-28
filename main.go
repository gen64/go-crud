package main

import (
	"fmt"
	"log"
	"os"
	"time"
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

	mc := &ModelController{}
	mc.AttachDBConn(conn)
	mc.SetDBTablePrefix("f0x_")

	models := []interface{}{
		&User{}, &Session{},
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

	// Add data
	user := &User{
		Flags: 1+2+4,
		Email: "admin@sysg.io",
		CreatedAt: time.Now().Unix(),
	}
	err = mc.SaveToDB(user)
	if err != nil {
		log.Printf("Error with SaveToDB on user: %s", err)
	}

	user.Flags = 1+2+4+8
	err = mc.SaveToDB(user)
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
	err = mc.SaveToDB(session)
	if err != nil {
		log.Printf("Error with SaveToDB on session: %s", err)
	}

	/*// Add data
	blogCategoryGeneral := &BlogCategory{}
	blogCategoryGeneral.Name = "General"
	blogCategoryGeneral.BlogPosts = []*BlogPost{
		&BlogPost{
			Title: "Welcome to my site",
			Content: "This is my first post in here",
		},
		&BlogPost{
			Title: "Second post",
			Content: "I'm happy to announce that my blog engine works"
		}
	}
	err = mc.SaveToDB(blogCategoryGeneral)
	if err != nil {
		log.Errorf("Error with SaveToDB on blogCategoryGeneral: %s", err)
	}

	blogCategoryDevops := &BlogCategory{}
	blogCategoryDevops.Name = "Devops"
	err = mc.SaveToDB(blogCategoryDevops)
	if err != nil {
		log.Errorf("Error with SaveToDB on blogCategoryDevops: %s", err)
	}

	blogPostDevops := &BlogPost{}
	blogPostDevops.Title = "Devops post"
	blogPostDevops.Content = "This one is a bit more technical"
	blogPostDevops.BlogCategoryID = blogCategoryDevops.GetID()
	err = mc.SaveToDB(blogPostDevops)
	if err != nil {
		log.Fatal(err)
	}*/
}
