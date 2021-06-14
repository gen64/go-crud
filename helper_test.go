package crud

import (
	"testing"
)

func TestSQLQueries(t *testing.T) {
	h := NewHelper(testStructObj, "")

	got := h.GetQueryDropTable()
	want := "DROP TABLE IF EXISTS test_structs"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryCreateTable()
	want = "CREATE TABLE test_structs (test_struct_id SERIAL PRIMARY KEY,test_struct_flags BIGINT DEFAULT 0,primary_email VARCHAR(255) DEFAULT '',email_secondary VARCHAR(255) DEFAULT '',first_name VARCHAR(255) DEFAULT '',last_name VARCHAR(255) DEFAULT '',age BIGINT DEFAULT 0,price BIGINT DEFAULT 0,post_code VARCHAR(255) DEFAULT '',post_code2 VARCHAR(255) DEFAULT '',password VARCHAR(255) DEFAULT '',created_by_user_id BIGINT DEFAULT 0,key VARCHAR(255) DEFAULT '' UNIQUE)"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLInsertQueries(t *testing.T) {
	h := NewHelper(testStructObj, "")

	got := h.GetQueryInsert()
	want := "INSERT INTO test_structs(test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING test_struct_id"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLUpdateQueries(t *testing.T) {
	h := NewHelper(testStructObj, "")

	got := h.GetQueryUpdateById()
	want := "UPDATE test_structs SET test_struct_flags=$1,primary_email=$2,email_secondary=$3,first_name=$4,last_name=$5,age=$6,price=$7,post_code=$8,post_code2=$9,password=$10,created_by_user_id=$11,key=$12 WHERE test_struct_id = $13"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLDeleteQueries(t *testing.T) {
	h := NewHelper(testStructObj, "")

	got := h.GetQueryDeleteById()
	want := "DELETE FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLSelectQueries(t *testing.T) {
	h := NewHelper(testStructObj, "")

	got := h.GetQuerySelectById()
	want := "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQuerySelect(nil, 67, 13, nil, nil, nil)
	want = "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{"EmailSecondary", "desc", "Age", "asc"}, 67, 13, map[string]interface{}{"Price": 4444, "PostCode2": "11-111"}, nil, nil)
	want = "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs WHERE post_code2=$1 AND price=$2 ORDER BY email_secondary DESC,age ASC LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{"EmailSecondary", "desc", "Age", "asc"}, 67, 13, map[string]interface{}{"Price": 4444, "PostCode2": "11-111"}, map[string]bool{"EmailSecondary": true}, nil)
	want = "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs WHERE post_code2=$1 AND price=$2 ORDER BY email_secondary DESC LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{"EmailSecondary", "desc", "Age", "asc"}, 67, 13, map[string]interface{}{"Price": 4444, "PostCode2": "11-111"}, map[string]bool{"EmailSecondary": true}, map[string]bool{"Price": true})
	want = "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs WHERE price=$1 ORDER BY email_secondary DESC LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestPluralName(t *testing.T) {
	type Category struct{}
	type Cross struct{}
	type ProductCategory struct{}
	type UserCart struct{}

	h1 := NewHelper(&Category{}, "")
	h2 := NewHelper(&Cross{}, "")
	h3 := NewHelper(&ProductCategory{}, "")
	h4 := NewHelper(&UserCart{}, "")

	got := h1.GetQueryDropTable()
	want := "DROP TABLE IF EXISTS categories"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h2.GetQueryDropTable()
	want = "DROP TABLE IF EXISTS crosses"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h3.GetQueryDropTable()
	want = "DROP TABLE IF EXISTS product_categories"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h4.GetQueryDropTable()
	want = "DROP TABLE IF EXISTS user_carts"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestValidationFields(t *testing.T) {
	h := NewHelper(testStructObj, "")

	got := h.fieldsRequired
	want := map[string]bool{"PrimaryEmail": true, "EmailSecondary": true, "FirstName": true, "LastName": true, "Age": true, "PostCode": true, "Key": true}
	if len(got) != len(want) {
		t.Fatalf("Required fields: want %v, got %v", len(want), len(got))
	}
	for i, _ := range got {
		if got[i] != want[i] {
			t.Fatalf("Required fields: want %v, got %v", want[i], got[i])
		}
	}

	got2 := h.fieldsLength
	want2 := map[string][2]int{"FirstName": [2]int{2, 30}, "LastName": [2]int{0, 255}, "PostCode": [2]int{6, 0}, "PostCode2": [2]int{6, 0}, "Key": [2]int{30, 255}}
	/*if len(got2) != len(want2) {
		t.Fatalf("Field lengths: want %v, got %v", len(want2), len(got2))
	}*/
	for _, v := range []string{"FirstName", "LastName", "PostCode", "PostCode2", "Key"} {
		if len(got2[v]) != len(want2[v]) {
			t.Fatalf("Field lengths: want %v, got %v", len(want2[v]), len(got2[v]))
		}
		for j := 0; j < 2; j++ {
			if got2[v][j] != want2[v][j] {
				t.Fatalf("Field lengths: want %v, got %v", want2[v][j], got2[v][j])
			}
		}
	}

	got3 := h.fieldsEmail
	want3 := map[string]bool{"PrimaryEmail": true, "EmailSecondary": true}
	if len(got3) != len(want3) {
		t.Fatalf("Email fields: want %v, got %v", len(want3), len(got3))
	}
	for i, _ := range got3 {
		if got3[i] != want3[i] {
			t.Fatalf("Email fields: want %v, got %v", want3[i], got3[i])
		}
	}

	got4 := h.fieldsValue
	want4 := map[string][2]int{"Age": [2]int{0, 120}, "Price": [2]int{0, 999}}
	/*if len(got4) != len(want4) {
		t.Fatalf("Field values: want %v, got %v", len(want4), len(got4))
	}*/
	for _, v := range []string{"Age", "Price"} {
		if len(got4[v]) != len(want4[v]) {
			t.Fatalf("Field values: want %v, got %v", len(want4[v]), len(got4[v]))
		}
		for j := 0; j < 2; j++ {
			if got4[v][j] != want4[v][j] {
				t.Fatalf("Field values: want %v, got %v", want4[v][j], got4[v][j])
			}
		}
	}

	got5 := h.fieldsValueNotNil
	want5 := map[string][2]bool{"Age": [2]bool{false, false}, "Price": [2]bool{true, false}}
	/*if len(got5) != len(want5) {
		t.Fatalf("Field value not nils: want %v, got %v", len(want5), len(got5))
	}*/
	for _, v := range []string{"Age", "Price"} {
		if len(got5[v]) != len(want5[v]) {
			t.Fatalf("Field value not nils: want %v, got %v", len(want5[v]), len(got5[v]))
		}
		for j := 0; j < 2; j++ {
			if got5[v][j] != want5[v][j] {
				t.Fatalf("Field value not nils: want %v, got %v", want5[v][j], got5[v][j])
			}
		}
	}

	if h.fieldsRegExp == nil {
		t.Fatalf("Got empty fieldsRegExp")
	}
	if h.fieldsRegExp["PostCode"] == nil || h.fieldsRegExp["PostCode2"] == nil {
		t.Fatalf("Missing entry in fieldsRegExp")
	}
	if h.fieldsRegExp["PostCode"].String() != "^[0-9]{2}\\-[0-9]{3}$" {
		t.Fatalf("Want ^[0-9]{2}\\-[0-9]{3}$, got %v", h.fieldsRegExp["PostCode"].String())
	}
	if h.fieldsRegExp["PostCode2"].String() != "^[0-9]{2}\\-[0-9]{3}$" {
		t.Fatalf("Want ^[0-9]{2}\\-[0-9]{3}$, got %v", h.fieldsRegExp["PostCode2"].String())
	}
}
