# go-crud

[![Build Status](https://travis-ci.com/gen64/go-crud.svg?branch=main)](https://travis-ci.com/gen64/go-crud)

Package CRUD is meant to make two things: map structs to PostgreSQL tables
(like ORM) and create CRUD HTTP endpoint for simple data management.


## Example usage
### Structs (models)
Models are defined with structs as follows (take a closer look at the tags):

```
type User struct {
	ID                 int    `json:"user_id"`
	Flags              int    `json:"flags"`
	Name               string `json:"name" crud:"req lenmin:2 lenmax:50"`
	Email              string `json:"email" crud:"req"`
	Password           string `json:"password"`
	EmailActivationKey string `json:"email_activation_key" crud:""`
	CreatedAt          int    `json:"created_at"`
	CreatedByUserID    int    `json:"created_by_user_id"`
}

type Session struct {
	ID                 int    `json:"session_id"`
	Flags              int    `json:"flags"`
	Key                string `json:"key" crud:"uniq lenmin:32 lenmax:50"`
	ExpiresAt          int    `json:"expires_at"`
	UserID             int    `json:"user_id" crud:"req"`
}

type Something struct {
	ID                 int    `json:"something_id"`
	Flags              int    `json:"flags"`
	Email              string `json:"email" crud:"req"`
	Age                int    `json:"age" crud:"req valmin:18 valmax:130 val:18"`
	Price              int    `json:"price" crud:"req valmin:0 valmax:9900 val:100"`
	CurrencyRate       int    `json:"currency_rate" crud:"req valmin:40000 valmax:61234 val:10000"`
	PostCode           string `json:"post_code" crud:"req val:32-600"`
}
```


#### Field tags
Struct tags define ORM behaviour. `go-crud` parses tags such as `crud`, `http`
and various tags starting with `crud_`. Apart from the last one, a tag define
many properties which are separated with space char, and if they contain
a value other than bool (true, false), it is added after semicolon char.
See below list of all the tags with examples.

Tag | Example | Explanation
--- | --- | ---
`crud` | `crud:"req valmin:0 valmax:130 val:18"` | Struct field properties defining its valid value for model. See CRUD Field Properties for more info
`crud_val` | `crud_val:"Default value"` | Struct field default value
`crud_regexp` | `crud_regexp:"^[0-9]{2}\\-[0-9]{3}$"` | Regular expression that struct field must match
`crud_testvalpattern` | `crud_testvalpattern:DD-DDD` | Very simple pattern for generating valid test value (used for tests). In the string, `D` is replaced with a digit


##### CRUD Field Properties
Property | Explanation
--- | ---
`req` | Field is required
`uniq` | Field has to be unique (like `UNIQUE` on the database column)
`valmin` | If field is numeric, this is minimal value for the field
`valmax` | If field is numeric, this is maximal value for the field
`val` | Default value for the field. If the value is not a simple, short alphanumeric, use the `crud_val` tag for it
`lenmin` | If field is string, this is a minimal length of the field value
`lenmax` | If field is string, this is a maximal length of the field value


### Database storage
Currently, `go-crud` supports only PostgreSQL as a storage for objects. 

#### Controller
To perform model database actions, a `Controller` object must be created. See
below example that modify object(s) in the database.

```
// Create connection with sql
conn, _ := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPass, dbName))
defer conn.Close()

// Create CRUD controller and an instance of a struct
c := crud.NewController(conn, "app1_")
user := &User{}

err = c.CreateDBTable(user) // Run 'CREATE TABLE'

user.Email = "test@example.com"
user.Name = "Nicholas"
user.CreatedAt = time.Now().Unix()
err = c.SaveToDB(user) // Insert object to database table

user.Email = "newemail@example.com"
err = c.SaveToDB() // Update object in the database table

err = c.DeleteFromDB() // Delete object from the database table

err = c.DropDBTable(user) // Run 'DROP TABLE'
```

### HTTP Endpoints
With `go-crud`, HTTP endpoints can be created to manage objects stored in the
database.

If User struct is used for HTTP endpoint, fields such as `Password` will be
present when listing users. Therefore, it's necessary to create new
structs to define CRUD endpoints' input and/or output. These structs should
start with `User_` so that `go-crud` knows its validation is already defined
within `User` struct (which is the main complete object).

In below example, `User_Create` defines input fields when creating a User,
`User_Update` defines fields that are meant to change when permorming update,
`User_UpdatePassword` is an additional struct just for updating User password,
and finally - fields in `User_List` will be visible when listing users or
reading one user. (You can define these as you like).
```
type User_Create {
	ID       int    `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type User_Update {
	ID       int    `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}
type User_UpdatePassword {
	ID       int `json:"user_id"`
	Password string `json:"password"`
}
type User_List {
	ID       int    `json:"user_id"`
	Name     string `json:"name"
}
```

```
var parentFunc = func() interface{} { return &User; }
var createFunc = func() interface{} { return &User_Create; }
var readFunc   = func() interface{} { return &User_List; }
var updateFunc = func() interface{} { return &User_Update; }
var listFunc   = func() interface{} { return &User_List; }

var updatePasswordFunc = func() interface{} { return &User_UpdatePassword; }

http.HandleFunc("/users/", c.GetHTTPHandler("/users/", parentFunc, createFunc, readFunc, updateFunc, parentFunc, listFunc))
http.HandleFunc("/users/password/", c.GetHTTPHandler("/users/password/", parentFunc, nil, nil, updatePasswordFunc, nil, nil))
log.Fatal(http.ListenAndServe(":9001", nil))
```

In the example, `/users/` CRUDL endpoint is created and it allows to:
* create new User by sending JSON payload using PUT method
* update existing User by sending JSON payload to `/users/:id` with PUT method
* get existing User details with making GET request to `/users/:id`
* delete existing User with DELETE request to `/users/:id`
* get list of Users with making GET request to `/users/` with optional query parameters such as `limit`, `offset` to slice the returned list and `filter_` params (eg. `filter_email`) to filter out records with by specific fields

When creating or updating an object, JSON payload with object details is
required. It should match the struct used for Create and Update operations.
In this case, `User_Create` and `User_Update`.

```
{
	"email": "test@example.com",
	"name": "Nicholas",
	...
}
```

Output from the endpoint is in JSON format as well.
