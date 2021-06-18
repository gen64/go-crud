package crud

import (
	"log"
	"testing"
)

func TestGetModelIDInterface(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	ts.ID = 123
	i := testController.GetModelIDInterface(ts)
	if *(i.(*int64)) != int64(123) {
		log.Fatalf("GetModelIDInterface failed to get ID")
	}
}

func TestGetModelIDValue(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	ts.ID = 123
	v := testController.GetModelIDValue(ts)
	if v != 123 {
		log.Fatalf("GetModelIDValue failed to get ID")
	}
}

func TestGetModelFieldInterfaces(t *testing.T) {
	// TODO
}

func TestResetFields(t *testing.T) {
	// TODO
}

func TestCreateDBTables(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	_ = testController.CreateDBTables(ts)

	cnt, err2 := getTableNameCnt("gen64_test_structs")
	if err2 != nil {
		t.Fatalf("CreateDBTables failed to create table for a struct: %s", err2.Error())
	}
	if cnt == 0 {
		t.Fatalf("CreateDBTables failed to create the table")
	}
}

func TestValidate(t *testing.T) {
	ts := getTestStructWithData()
	ts.PrimaryEmail = "primary@gen64.net"
	ts.EmailSecondary = "secondary@gen64.net"
	ts.PostCode = "00-000"
	b, failedFields, err := testController.Validate(ts, nil)
	log.Print(failedFields)
	log.Print(b)
	log.Print(err)
}

/*
func TestSaveToDB(t *testing.T) {
	ts := getTestStructWithData()

	err := testController.SaveToDB(ts)
	if err != nil {
		t.Fatalf("SaveToDB failed to insert struct to the table: %s", err.Op)
	}
	id, flags, primaryEmail, emailSecondary, firstName, lastName, age, price, postCode, postCode2, password, createdByUserID, key, err2 := getRow()
	if err2 != nil {
		t.Fatalf("SaveToDB failed to insert struct to the table: %s", err.Error())
	}
	if id == 0 || flags != ts.Flags || primaryEmail != ts.PrimaryEmail || emailSecondary != ts.EmailSecondary || firstName != ts.FirstName || lastName != ts.LastName || age != ts.Age || price != ts.Price || postCode != ts.PostCode || postCode2 != ts.PostCode2 || createdByUserID != ts.CreatedByUserID || key != ts.Key || password != ts.Password {
		t.Fatalf("SaveToDB failed to insert struct to the table")
	}

	ts.Flags = 7
	ts.PrimaryEmail = "primary1@gen64.net"
	ts.EmailSecondary = "secondary2@gen64.net"
	ts.FirstName = "Johnny"
	ts.LastName = "Smithsy"
	ts.Age = 50
	ts.Price = 222
	ts.PostCode = "22-222"
	ts.PostCode2 = "33-333"
	ts.Password = "xxx"
	ts.CreatedByUserID = 7
	ts.Key = "reallyunique"
	_ = testController.SaveToDB(ts)

	flags, primaryEmail, emailSecondary, firstName, lastName, age, price, postCode, postCode2, password, createdByUserID, key, err2 = getRowById(id)
	if err2 != nil {
		t.Fatalf("SaveToDB failed to update struct in the table: %s", err.Error())
	}
	if id == 0 || flags != ts.Flags || primaryEmail != ts.PrimaryEmail || emailSecondary != ts.EmailSecondary || firstName != ts.FirstName || lastName != ts.LastName || age != ts.Age || price != ts.Price || postCode != ts.PostCode || postCode2 != ts.PostCode2 || createdByUserID != ts.CreatedByUserID || key != ts.Key || password != ts.Password {
		t.Fatalf("SaveToDB failed to update struct to the table")
	}
}

func TestSetFromDB(t *testing.T) {
	ts := getTestStructWithData()
	err := testController.SaveToDB(ts)
	if err != nil {
		t.Fatalf("SaveToDB in TestSetFromDB failed to insert struct to the table: %s", err.Op)
	}

	ts2 := testStructNewFunc().(*TestStruct)
	err = testController.SetFromDB(ts2, fmt.Sprintf("%d", ts.ID))
	if err != nil {
		t.Fatalf("SetFromDB failed to get data: %s", err.Op)
	}

	if !areTestStructObjectSame(ts, ts2) {
		t.Fatalf("SetFromDB failed to set struct with data: %s", err.Op)
	}
}

func TestDeleteFromDB(t *testing.T) {
	ts := getTestStructWithData()
	err := testController.SaveToDB(ts)
	if err != nil {
		t.Fatalf("SaveToDB in TestDeleteFromDB failed to insert struct to the table: %s", err.Op)
	}
	err = testController.DeleteFromDB(ts)
	if err != nil {
		t.Fatalf("DeleteFromDB failed to remove: %s", err.Op)
	}

	cnt, err2 := getRowCntById(ts.ID)
	if err2 != nil {
		t.Fatalf("DeleteFromDB failed to delete struct from the table")
	}
	if cnt > 0 {
		t.Fatalf("DeleteFromDB failed to delete struct from the table")
	}
	if ts.ID != 0 {
		t.Fatalf("DeleteFromDB failed to set ID to 0 on the struct")
	}
}

func TestGetFromDB(t *testing.T) {
	for i := 1; i < 51; i++ {
		ts := getTestStructWithData()
		ts.ID = 0
		ts.Age = 30 + i
		testController.SaveToDB(ts)
	}

	testStructs, err := testController.GetFromDB(testStructNewFunc, []string{"Age", "asc", "Price", "asc"}, 10, 20, map[string]interface{}{"Price": 444, "PrimaryEmail": "primary@gen64.net"})
	if err != nil {
		t.Fatalf("GetFromDB failed to return list of objects: %s", err.Op)
	}
	if len(testStructs) != 10 {
		t.Fatalf("GetFromDB failed to return list of objects, want %v, got %v", 10, len(testStructs))
	}
	if testStructs[2].(*TestStruct).Age != 52 {
		t.Fatalf("GetFromDB failed to return correct list of objects, want %v, got %v", 52, testStructs[2].(*TestStruct).Age)
	}
}

func TestHTTPHandlerPutMethodForValidation(t *testing.T) {
	// TODO
}

func TestHTTPHandlerPutMethodForCreating(t *testing.T) {
	j := `{
		"email": "test@example.com",
		"first_name": "John",
		"last_name": "Smith",
		"key": "uniquekey1"
	}`
	_ = makePUTInsertRequest(j, t)

	id, flags, primaryEmail, emailSecondary, firstName, lastName, age, price, postCode, postCode2, password, createdByUserID, key, err := getRow()
	if err != nil {
		t.Fatalf("PUT method failed to insert struct to the table: %s", err.Error())
	}
	if id == 0 || flags != 0 || primaryEmail != "test@example.com" || emailSecondary != "" || firstName != "John" || lastName != "Smith" || age != 0 || price != 0 || postCode != "" || postCode2 != "" || createdByUserID != 0 || key != "uniquekey1" || password != "" {
		t.Fatalf("PUT method failed to insert struct to the table")
	}

	// TODO: Check if response contains JSON with key 'id'
}

func TestHTTPHandlerPutMethodForUpdating(t *testing.T) {
	j := `{
		"test_struct_flags": 8,
		"email": "test11@example.com",
		"email2": "test22@example.com",
		"first_name": "John2",
		"last_name": "Smith2",
		"age": 39,
		"price": 1002,
		"post_code": "22-222",
		"post_code2": "33-333",
		"password": "password123updated",
		"created_by_user_id": 12,
		"key": "uniquekey2"
	}`
	_ = makePUTUpdateRequest(j, 54, t)

	id, flags, primaryEmail, emailSecondary, firstName, lastName, age, price, postCode, postCode2, password, createdByUserID, key, err := getRow()
	if err != nil {
		t.Fatalf("PUT method failed to update struct to the table: %s", err.Error())
	}
	// Only 2 fields should be updated: FirstName and LastName. Check the TestStruct_Update struct
	if id == 0 || flags != 0 || primaryEmail != "test@example.com" || emailSecondary != "" || firstName != "John2" || lastName != "Smith2" || age != 0 || price != 0 || postCode != "" || postCode2 != "" || createdByUserID != 0 || key != "uniquekey1" || password != "" {
		t.Fatalf("PUT method failed to insert struct to the table")
	}
}

func TestHTTPHandlerGetMethodOnExisting(t *testing.T) {
	resp := makeGETReadRequest(54, t)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET method returned wrong status code, want %d, got %d", http.StatusOK, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("GET method failed")
	}

	ts := testStructNewFunc().(*TestStruct)
	err = json.Unmarshal(body, ts)
	if err != nil {
		t.Fatalf("GET method failed to return unmarshable JSON")
	}
	if ts.Age != 0 {
		t.Fatalf("GET method returned invalid values")
	}
	if strings.Contains(string(body), "email2") {
		t.Fatalf("GET method returned output with field that should have been hidden")
	}
	if strings.Contains(string(body), "post_code2") {
		t.Fatalf("GET method returned output with field that should have been hidden")
	}
}

func TestHTTPHandlerDeleteMethod(t *testing.T) {
	makeDELETERequest(54, t)

	cnt, err2 := getRowCntById(54)
	if err2 != nil {
		t.Fatalf("DELETE handler failed to delete struct from the table")
	}
	if cnt > 0 {
		t.Fatalf("DELETE handler failed to delete struct from the table")
	}
}

func TestHTTPHandlerGetMethodOnNonExisting(t *testing.T) {
	resp := makeGETReadRequest(54, t)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET method returned wrong status code, want %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

func TestHTTPHandlerGetMethodWithoutID(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	for i := 1; i <= 55; i++ {
		testController.ResetFields(ts)
		ts.ID = 0
		testController.SetFromDB(ts, fmt.Sprintf("%d", i))
		if ts.ID != 0 {
			ts.Password = "abcdefghijklopqrwwe"
			testController.SaveToDB(ts)
		}
	}
	b := makeGETListRequest(map[string]string{
		"limit":                "10",
		"offset":               "20",
		"order":                "age",
		"order_direction":      "asc",
		"filter_price":         "444",
		"filter_primary_email": "primary@gen64.net",
	}, t)

	o := struct {
		Items []map[string]interface{} `json:"items"`
	}{
		Items: []map[string]interface{}{},
	}
	err := json.Unmarshal(b, &o)
	if err != nil {
		t.Fatalf("GET method returned wrong json output, error marshaling: %s", err.Error())
	}

	if len(o.Items) != 10 {
		t.Fatalf("GET method returned invalid number of rows, want %d got %d", 10, len(o.Items))
	}

	if o.Items[2]["age"].(float64) != 52 {
		t.Fatalf("GET method returned invalid row, want %d got %f", 52, o.Items[2]["age"].(float64))
	}

	if strings.Contains(string(b), "email2") {
		t.Fatalf("GET method returned output with field that should have been hidden")
	}
	if strings.Contains(string(b), "post_code2") {
		t.Fatalf("GET method returned output with field that should have been hidden")
	}
}

func TestDropDBTables(t *testing.T) {
	ts := testStructNewFunc().(*TestStruct)
	err := testController.DropDBTables(ts)
	if err != nil {
		t.Fatalf("DropDBTables failed to drop table for a struct: %s", err.Op)
	}

	cnt, err2 := getTableNameCnt("gen64_test_structs")
	if err2 != nil {
		t.Fatalf("DropDBTables failed to drop table for a struct: %s", err2.Error())
	}
	if cnt > 0 {
		t.Fatalf("DropDBTables failed to drop the table")
	}
}
*/
