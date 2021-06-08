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
	want = "CREATE TABLE test_structs (test_struct_id SERIAL PRIMARY KEY,test_struct_flags BIGINT,primary_email VARCHAR(255),email_secondary VARCHAR(255),first_name VARCHAR(255),last_name VARCHAR(255),age BIGINT,price BIGINT,post_code VARCHAR(255),post_code2 VARCHAR(255),password VARCHAR(255),created_by_user_id BIGINT,key VARCHAR(255))"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLInsertQueries(t *testing.T) {
	got = h.GetQueryInsert([]string{})
	want = "INSERT INTO test_structs(test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING test_struct_id"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryInsert([]string{"Flags","EmailSecondary","LastName"})
	want = "INSERT INTO test_structs(test_struct_flags,email_secondary,last_name) VALUES ($1,$2,$3) RETURNING test_struct_id"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLUpdateQueries(t *testing.T) {
	got = h.GetQueryUpdateById([]string{})
	want = "UPDATE test_structs SET test_struct_flags=$1,primary_email=$2,email_secondary=$3,first_name=$4,last_name=$5,age=$6,price=$7,post_code=$8,post_code2=$9,password=$10,created_by_user_id=$11,key=$12 WHERE test_struct_id = $13"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryUpdateById([]string{"Flags","EmailSecondary","LastName"})
	want = "UPDATE test_structs SET test_struct_flags=$1,email_secondary=$2,last_name=$3 WHERE test_struct_id = $4"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLDeleteQueries(t *testing.T) {
	got = h.GetQueryDeleteById()
	want = "DELETE FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestSQLSelectQueries(t *testing.T) {
	got = h.GetQuerySelectById([]string{})
	want = "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQuerySelectById([]string{"Flags","EmailSecondary","LastName"})
	want = "SELECT test_struct_flags,email_secondary,last_name FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{}, nil, 67, 13, nil)
	want = "SELECT test_struct_id,test_struct_flags,primary_email,email_secondary,first_name,last_name,age,price,post_code,post_code2,password,created_by_user_id,key FROM test_structs LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{"EmailSecondary","Age"}, nil, 67, 13, nil)
	want = "SELECT email_secondary,age FROM test_structs LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{"Age"}, map[string]string{"EmailSecondary":"desc","Age":"asc"}, 67, 13, nil)
	want = "SELECT age FROM test_structs ORDER BY email_secondary DESC,age ASC LIMIT 67 OFFSET 13"
	if got != want {
		t.Fatalf("want %v, got %v", want, got)
	}

	got = h.GetQuerySelect([]string{"Age"}, map[string]string{"EmailSecondary":"desc","Age":"asc"}, 67, 13, map[string]interface{}{"Price":4444,"PostCode2":"11-111"})
	want = "SELECT age FROM test_structs WHERE price=$1 AND post_code2=$2 ORDER BY email_secondary DESC,age ASC LIMIT 67 OFFSET 13"
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

/*
func TestValidationFields(t *testing.T) {
	h := NewHelper(ts, "")

	got := h.fieldsRequired
	want := map[string]bool{"Email": true, "Age": true, "Price": true, "CurrencyRate": true, "PostCode": true}
	if len(got) != len(want) {
		t.Fatalf("Want %v, got %v", len(want), len(got))
	}
	for i, _ := range got {
		if got[i] != want[i] {
			t.Fatalf("Want %v, got %v", want[i], got[i])
		}
	}

	got2 := h.fieldsLength
	want2 := map[string][2]int{"Email": [2]int{10, 255}, "PostCode": [2]int{6, -1}}
	if len(got2) != len(want2) {
		t.Fatalf("Want %v, got %v", len(want2), len(got2))
	}
	if len(got2["Email"]) != len(want2["Email"]) {
		t.Fatalf("Want %v, got %v", len(want2["Email"]), len(got2["Email"]))
	}
	if len(got2["PostCode"]) != len(want2["PostCode"]) {
		t.Fatalf("Want %v, got %v", len(want2["PostCode"]), len(got2["PostCode"]))
	}
	for k, _ := range got2 {
		for j := 0; j < 2; j++ {
			if got2[k][j] != want2[k][j] {
				t.Fatalf("Want %v, got %v", want2[k][j], got2[k][j])
			}
		}
	}

	got3 := h.fieldsEmail
	want3 := map[string]bool{"Email": true}
	if len(got3) != len(want3) {
		t.Fatalf("Want %v, got %v", len(want3), len(got3))
	}
	for i, _ := range got3 {
		if got3[i] != want3[i] {
			t.Fatalf("Want %v, got %v", want3[i], got3[i])
		}
	}

	got4 := h.fieldsValue
	want4 := map[string][2]int{"Age": [2]int{18, 120}, "Price": [2]int{5, 3580}, "CurrencyRate": [2]int{10, 50004}}
	if len(got4) != len(want4) {
		t.Fatalf("Want %v, got %v", len(want4), len(got4))
	}
	if len(got4["Age"]) != len(want4["Age"]) {
		t.Fatalf("Want %v, got %v", len(want4["Age"]), len(got4["Age"]))
	}
	if len(got4["Price"]) != len(want4["Price"]) {
		t.Fatalf("Want %v, got %v", len(want4["Price"]), len(got4["Price"]))
	}
	if len(got4["CurrencyRate"]) != len(want4["CurrencyRate"]) {
		t.Fatalf("Want %v, got %v", len(want4["CurrencyRate"]), len(got4["CurrencyRate"]))
	}
	for k, _ := range got4 {
		for j := 0; j < 2; j++ {
			if got4[k][j] != want4[k][j] {
				t.Fatalf("Want %v, got %v", want4[k][j], got4[k][j])
			}
		}
	}

	if h.fieldsRegExp == nil {
		t.Fatalf("Got empty fieldsRegExp")
	}
	if h.fieldsRegExp["PostCode"] == nil {
		t.Fatalf("Missing entry in fieldsRegExp")
	}
	if h.fieldsRegExp["PostCode"].String() != "^[0-9]{2}\\-[0-9]{3}$" {
		t.Fatalf("Want ^[0-9]{2}\\-[0-9]{3}$, got %v", h.fieldsRegExp["PostCode"].String())
	}
	// TODO: Cover all the fields ending with Fields
}
*/
