# go-crud

[![Build Status](https://travis-ci.com/gen64/go-crud.svg?branch=main)](https://travis-ci.com/gen64/go-crud)

Package CRUD is meant to make two things: map structs to PostgreSQL tables
(like ORM) and create CRUD HTTP endpoint for simple data management.


## Example usage
### Structs (models)
Models are defined with structs as follows (take a closer look at the tags):

```
type User struct {
	ID                 int    `json:"user_id" http_endpoint:"noread noupdate nocreate nodelete nolist"`
	Flags              int    `json:"flags"`
	Name               string `json:"name" crud:"req lenmin:2 lenmax:50"`
	Email              string `json:"email" crud:"req"`
	Password           string `json:"password" crud:"" http:"noread noupdate nocreate nolist"`
	EmailActivationKey string `json:"email_activation_key" crud:""`
	CreatedAt          int    `json:"created_at"`
	CreatedByUserID    int    `json:"created_by_user_id"`
}

type Session struct {
	ID                 int    `json:"session_id"`
	Flags              int    `json:"flags"`
	Key                string `json:"key" crud:"uniq lenmin:32 lenmax:50 genSetBy genUpdateBy"`
	ExpiresAt          int    `json:"expires_at"`
	UserID             int    `json:"user_id" crud:"req"`
}

type Something struct {
	ID                 int    `json:"something_id"`
	Flags              int    `json:"flags"`
	Email              string `json:"email" crud:"req"`
	Age                int    `json:"age" crud:"req valmin:18 valmax:130 val:18"`
	Price              int    `json:"price" crud:"req valmin:0 valminzero valmax:9900 val:100"`
	CurrencyRate       int    `json:"currency_rate" crud:"req valmin:40000 valmax:61234 val:10000"`
	PostCode           string `json:"post_code" crud:"req val:32-600 testval:32-600 testvalpattern:DD-DDD" crud_regexp:"^[0-9]{2}\\-[0-9]{3}$"`
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
`crud` | `crud:"req valmin:0 valminzero valmax:130 val:18"` | Struct field properties defining its valid value for model. See CRUD Field Properties for more info
`http` | `http:"noread noupdate nocreate nolist"` | Struct field configuration for model's HTTP endpoint. See HTTP Field Properties for more info
`crud_val` | `crud_val:"Default value"` | Struct field default value
`crud_regexp` | `crud_regexp:"^[0-9]{2}\\-[0-9]{3}$"` | Regular expression that struct field must match
`crud_testvalpattern` | `crud_testvalpattern:DD-DDD` | Very simple pattern for generating valid test value (used for tests). In the string, `D` is replaced with a digit


##### CRUD Field Properties
Property | Explanation
--- | ---
`req` | Field is required
`uniq` | Field has to be unique (like `UNIQUE` on the database column)
`valmin` | If field is numeric, this is minimal value for the field. If it needs to be set to `0`, then additional `valminzero` property must be added (in Go, numeric field cannot be nil)
`valminzero` | Add this if `valmin` is to equal `0`
`valmax` | If field is numeric, this is maximal value for the field
`val` | Default value for the field. If the value is not a simple, short alphanumeric, use the `crud_val` tag for it
`lenmin` | If field is string, this is a minimal length of the field value. If it needs to be set to `0`, then additional `lenminzero` property must be added (in Go, numeric field cannot be nil)
`lenminzero` | Add this if `lenmin` is to equal `0`
`lenmax` | If field is string, this is a maximal length of the field value
`testval` | Valid test value (used for tests)
`testvalpattern` | Same as `crud_testvalpattern`


##### HTTP Field Properties
Below properties configure field presence in model's HTTP endpoint.

Property | Explanation
--- | ---
`noread` | Field will not be present in the output when getting single object
`noupdate` | Field will not be updated (it will be ignored in the payload)
`nocreate` | When creating a new object, field will have its default value (it will be ignored in the payload)
`nolist` | Field will not be present in the output when getting any list of objects


#### Model tags
It is possible to define one additional tag `http_endpoint` in the first
struct field. It's used to configure HTTP endpoint for the whole model (for 
example User, in above case). Within the tag, you can define properties such
as:

* `noread` - HTTP endpoint will not allow to read (get) an object
* `noupdate` - HTTP endpoint will not allow updating an object
* `nocreate` - HTTP endpoint will not allow creating a new object
* `nodelete` - HTTP endpoint will not allow delete an object
* `nolist` - HTTP endpoint not allow listing objects


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
database. See below example that returns HTTP Handler func which can be 
attached to Golang's HTTP server.

```
http.HandleFunc("/users/", c.GetHTTPHandler(func() interface{} {
	return &User{}
}, "/users/"))
log.Fatal(http.ListenAndServe(":9001", nil))
```

In the example, `/users/` CRUDL endpoint is created and it allows to:
* create new User by sending JSON payload using PUT method
* update existing User by sending JSON payload to `/users/:id` with PUT method
* get existing User details with making GET request to `/users/:id`
* delete existing User with DELETE request to `/users/:id`
* get list of Users with making GET request to `/users/` with optional query parameters such as `limit`, `offset` to slice the returned list and `filter_` params (eg. `filter_email`) to filter out records with by specific fields

When creating or updating an object, JSON payload with object details is
required, as on the example below:
```
{
	"email": "test@example.com",
	"name": "Nicholas",
	"created_at": "1610356241",
	...
}
```

Output from the endpoint is in JSON format as well.

#### Custom HTTP Endpoints
It's possible to create custom CRUD endpoints that will operate only on 
specific model fields. For example, you might create endpoint that lists Users
but shows only it's ID and Name, or an endpoint that updates only User
password. Check below code.
```
// List users
http.HandleFunc("/users/list_emails/", c.GetCustomHTTPHandler(func() interface{} {
	return &User{}
}, "/users/shortlist/", crud.OpList, ["Email"]))

// Allow to update only the Password field
http.HandleFunc("/users/update_password/", c.GetCustomHTTPHandler(func() interface{} {
	return &User{}
}, "/users/password/", crud.OpUpdate, ["Password"]))

// Allow to create new User (only with Email and Name fields), and to read one
// element via /users/create_and_read/:id - which will return Email and Name only
http.HandleFunc("/users/create_and_read/", c.GetCustomHTTPHandler(func() interface{} {
	return &User{}
}, "/users/create/", crud.OpCreate | crud.OpRead, ["Email", "Name"]) 
```

Delete is unavailable with custom HTTP endpoints.