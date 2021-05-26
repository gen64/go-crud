# go-crud

[![Build Status](https://travis-ci.com/gen64/go-crud.svg?branch=main)](https://travis-ci.com/gen64/go-crud)

Package CRUD is meant to make two things: map structs to PostgreSQL tables
(like ORM) and create CRUD HTTP endpoint for simple data management.


## Example usage
### Structs (models)
For example, you can define structs (later called models) as follows (note the tags):

```
type User struct {
	ID                 int    `json:"user_id" http_endpoint:"noread noupdate nocreate nodelete nolist"`
	Flags              int    `json:"flags"`
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
Here is an example, creating table based on struct, adding record, updating
it and deleting.

```
conn, _ := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost,Port, dbUser, dbPass, dbName))
defer conn.Close()

c := crud.NewController(conn, "app1_")
user := &User{}
err = c.CreateDBTable(user) // runs CREATE TABLE

user.Email = "test@example.com"
user.Name = "Nicholas"
user.CreatedAt = time.Now().Unix()
err = c.SaveToDB(user) // runs INSERT

user.Email = "newemail@example.com"
err = c.SaveToDB() // runs UPDATE

err = c.DeleteFromDB() // runs DELETE

err = c.DropDBTable(user) // runs DROP TABLE
```

### HTTP Handler
Finally, here is an example of creating CRUD HTTP endpoint.

```
http.HandleFunc("/users/", c.GetHTTPHandler(user, "/users/"))
log.Fatal(http.ListenAndServe(":9001", nil))
```

With above, you can send a JSON payload using PUT method to `/users/`
endpoint to create a new record.
For already existing record, use `/users/:id` with PUT, GET or DELETE method to
update, get or delete the record.
Here is how JSON input would look like for previously shown User struct.

```
{
	"email": "test@example.com",
	"name": "James",
	"created_at": "1610356241"
}
```
