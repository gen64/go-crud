// Package CRUDL is meant to make two things: map structs to PostgreSQL tables
// (like ORM) and create CRUDL HTTP endpoint for simple data management.
//
// For example, a struct can be something as follows (note the tags):
//	type User struct {
//		ID        int64  `json:"user_id"`
//		Email     string `json:"email" crudl:"req email"`
//		Name      string `json:"name" crudl:"lenmax:255"`
//		CreatedAt int64  `json:"created_at"`
//	}
//
// Here is an example, creating table based on struct, adding record, updating
// it and deleting.
//
//	conn, _ := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPass, dbName))
//	defer conn.Close()
//
//	c := crudl.NewController(conn, "app1_")
//	user := &User{}
//	err = c.CreateDBTable(user) // runs CREATE TABLE
//
// 	user.Email = "test@example.com"
// 	user.Name = "Nicholas"
// 	user.CreatedAt = time.Now().Unix()
// 	err = c.SaveToDB(user) // runs INSERT
//
// 	user.Email = "newemail@example.com"
//	err = c.SaveToDB() // runs UPDATE
//
//	err = c.DeleteFromDB() // runs DELETE
//
//	err = c.DropDBTable(user) // runs DROP TABLE
//
// Finally, here is an example of creating CRUDL HTTP endpoint.
//	http.HandleFunc("/users/", c.GetHTTPHandler(user, "/users/"))
//	log.Fatal(http.ListenAndServe(":9001", nil))
// With above, you can send a JSON payload using PUT method to /users/
// endpoint to create a new record.
// For already existing record, use /users/:id with PUT, GET or DELETE method to
// update, get or delete the record.
// Here is how JSON input would look like for previously shown User struct.
//	{ "email": "test@example.com", "name": "James", "created_at": "1610356241" }
package crudl
