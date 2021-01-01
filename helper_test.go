package crudl

import (
	"testing"
)

type TestStruct struct {
	ID           int64  `json:"teststruct_id"`
	Flags        int64  `json:"teststruct_flags"`
	Email        string `json:"email" crudl:"req lenmin:10 lenmax:255 email"`
	Age          int    `json:"age" crudl:"req valmin:18 valmax:120"`
	Price        int    `json:"price" crudl:"req valmin:5 valmax:3580"`
	CurrencyRate int    `json:"currency_rate" crudl:"req valmin:10 valmax:50004"`
	PostCode     string `json:"post_code" crudl:"req lenmin:6 regexp:^[0-9]{2}\\-[0-9]{3}$"`
}

var ts = &TestStruct{}

func TestSQLQueries(t *testing.T) {
	h, _ := NewHelper(ts)

	got := h.GetQueryDropTable()
	want := "DROP TABLE IF EXISTS test_structs"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryCreateTable()
	want = "CREATE TABLE test_structs (test_struct_id SERIAL PRIMARY KEY, test_struct_flags BIGINT, email VARCHAR(255), age BIGINT, price BIGINT, currency_rate BIGINT, post_code VARCHAR(255))"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryInsert()
	want = "INSERT INTO test_structs(test_struct_flags,email,age,price,currency_rate,post_code) VALUES ($1,$2,$3,$4,$5,$6) RETURNING test_struct_id"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryUpdateById()
	want = "UPDATE test_structs SET test_struct_flags=$1,email=$2,age=$3,price=$4,currency_rate=$5,post_code=$6 WHERE test_struct_id = $7"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQuerySelectById()
	want = "SELECT test_struct_id, test_struct_flags, email, age, price, currency_rate, post_code FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}

	got = h.GetQueryDeleteById()
	want = "DELETE FROM test_structs WHERE test_struct_id = $1"
	if got != want {
		t.Fatalf("Want %v, got %v", want, got)
	}
}

func TestPluralName(t *testing.T) {
	type Category struct {}
	type Cross struct {}
	type ProductCategory struct {}
	type UserCart struct {}

	h1, _ := NewHelper(&Category{})
	h2, _ := NewHelper(&Cross{})
	h3, _ := NewHelper(&ProductCategory{})
	h4, _ := NewHelper(&UserCart{})

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
	h, _:= NewHelper(ts)

	got := h.reqFields
	want := []int{2,3,4,5,6}
	if len(got) != len(want) {
		t.Fatalf("Want %v, got %v", len(want), len(got))
	}
	for i, _ := range got {
		if got[i] != want[i] {
			t.Fatalf("Want %v, got %v", want[i], got[i])
		}
	}

	got2 := h.lenFields
	want2 := [][3]int{{2,10,255}, {6,6,-1}}
	if len(got2) != len(want2) {
		t.Fatalf("Want %v, got %v", len(want2), len(got2))
	}
	if len(got2[0]) != len(want2[0]) {
		t.Fatalf("Want %v, got %v", len(want2), len(got2))
	}
	if len(got2[1]) != len(want2[1]) {
		t.Fatalf("Want %v, got %v", len(want2), len(got2))
	}
	for i, _ := range got2 {
		for j:=0; j<3; j++ {
			if got2[i][j] != want2[i][j] {
				t.Fatalf("Want %v, got %v", want2[i][j], got2[i][j])
			}
		}
	}

	got3 := h.emailFields
	want3 := []int{2}
	if len(got3) != len(want3) {
		t.Fatalf("Want %v, got %v", len(want3), len(got3))
	}
	for i, _ := range got3 {
		if got3[i] != want3[i] {
			t.Fatalf("Want %v, got %v", want3[i], got3[i])
		}
	}

	got4 := h.valFields
	want4 := [][3]int{{3,18,120},{4,5,3580},{5,10,50004}}
	if len(got4) != len(want4) {
		t.Fatalf("Want %v, got %v", len(want4), len(got4))
	}
	if len(got4[0]) != len(want4[0]) {
		t.Fatalf("Want %v, got %v", len(want4), len(got4))
	}
	if len(got4[1]) != len(want4[1]) {
		t.Fatalf("Want %v, got %v", len(want4), len(got4))
	}
	if len(got4[2]) != len(want4[2]) {
		t.Fatalf("Want %v, got %v", len(want4), len(got4))
	}
	for i, _ := range got4 {
		for j:=0; j<3; j++ {
			if got4[i][j] != want4[i][j] {
				t.Fatalf("Want %v, got %v", want4[i][j], got4[i][j])
			}
		}
	}

	if h.regexpFields == nil {
		t.Fatalf("Got empty regexpFields")
	}
	if h.regexpFields[6] == nil {
		t.Fatalf("Missing entry in regexpFields")
	}
	if h.regexpFields[6].String() != "^[0-9]{2}\\-[0-9]{3}$" {
		t.Fatalf("Want ^[0-9]{2}\\-[0-9]{3}$, got %v", h.regexpFields[6].String())
	}
	// TODO: Cover all the fields ending with Fields
}
