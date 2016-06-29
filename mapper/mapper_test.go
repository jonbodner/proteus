package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jonbodner/gdb/cmp"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"reflect"
	"testing"
)

func TestExtract(t *testing.T) {
	f := func(in interface{}, path []string, expected int) {
		v, err := Extract(in, path)
		if err != nil {
			t.Errorf("Expected no error, got %s", err)
		}
		if v == nil {
			t.Errorf("Expected back an int, got a nil")
		}
		if i, ok := v.(int); !ok {
			t.Errorf("Expected back an int, got a ", reflect.TypeOf(v).Kind())
		} else {
			if i != expected {
				t.Errorf("Expected back %d, got %d", expected, i)
			}
		}
	}
	// base case int
	f(10, []string{"A"}, 10)

	// ptr case
	a := 20
	f(&a, []string{"A"}, a)

	// struct case
	type Bar struct {
		A int
	}

	type Foo struct {
		B Bar
	}

	f(Foo{
		B: Bar{
			A: 100,
		},
	}, []string{"foo", "B", "A"}, 100)

	// map case
	f(map[string]interface{}{
		"Bar": Bar{
			A: 200,
		},
	}, []string{"m", "Bar", "A"}, 200)
}

func TestExtractFail(t *testing.T) {
	f := func(in interface{}, path []string, msg string) {
		_, err := Extract(in, path)
		if err == nil {
			t.Errorf("Expected an error %s, got none", msg)
		}
		eExp := errors.New(msg)
		if !cmp.Errors(err, eExp) {
			t.Errorf("Expected error %s, got %s", eExp, err)
		}
	}
	//base case no path
	f(10, []string{}, "cannot extract value; no path remaining")

	//path too long
	f(10, []string{"A", "B"}, "cannot extract value; only maps and structs can have contained values")

	//invalid map
	f(map[int]interface{}{10: "Hello"}, []string{"m", "10"}, "cannot extract value; map does not have a string key")

	//no such field case
	type Bar struct {
		A int
	}
	f(Bar{A: 20}, []string{"b", "B"}, "cannot extract value; no such field B")
}

func TestBuild(t *testing.T) {
	//todo
	v, err := Build(nil, reflect.TypeOf(10))
	if v != nil {
		t.Error("Expected nil when passing in nil rows")
	}
	eExp := errors.New("Both rows and sType must be non-nil")
	if !cmp.Errors(err, eExp) {
		t.Errorf("Expected error %s, got %s", eExp, err)
	}
}

func setupDb(db *sql.DB) {
	sqlStmt := `
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into product(id, name, cost) values(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("person%d", i), 1.1*float64(i))
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()
}

func TestBuildSqliteStruct(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sqlite test in short mode")
	}
	os.Remove("./gdb.db")

	db, err := sql.Open("sqlite3", "./gdb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	setupDb(db)

	//struct
	type Product struct {
		Id   int     `gdbf:"id,pk"`
		Name *string `gdbf:"name"`
		Cost float64 `gdbf:"cost"`
	}

	rows, err := db.Query("select id, name, cost from product")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	pType := reflect.TypeOf((*Product)(nil)).Elem()

	for i := 0; i < 100; i++ {
		prod, err := Build(rows, pType)
		if err != nil {
			t.Fatal(err)
		}
		p2, ok := prod.(Product)
		if !ok {
			t.Error("wrong type")
		} else {
			if p2.Id != i {
				t.Errorf("Wrong id, expected %d, got %d", i, p2.Id)
			}
			if *p2.Name != fmt.Sprintf("person%d", i) {
				t.Errorf("Wrong name, expected %s, got %s", fmt.Sprintf("person%d", i), *p2.Name)
			}
			if p2.Cost != 1.1*float64(i) {
				t.Errorf("Wrong cost, expected %f, got %f", 1.1*float64(i), p2.Cost)
			}
		}
		//fmt.Println(p2)
	}
	err = rows.Err()
	if err != nil {
		t.Error(err)
	}
	if rows.Next() {
		t.Error("Expected no more rows, but had some")
	}
	prod, err := Build(rows, pType)
	if prod != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}
}

func TestBuildSqlitePrimitive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sqlite test in short mode")
	}
	os.Remove("./gdb.db")

	db, err := sql.Open("sqlite3", "./gdb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	setupDb(db)

	//primitive
	stmt, err := db.Prepare("select name from product where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query("3")
	if err != nil {
		log.Fatal(err)
	}
	sType := reflect.TypeOf("")
	s, err := Build(rows, sType)
	if err != nil {
		t.Error(err)
	}
	if s == nil {
		t.Error("Got nil back, expected a string")
	}
	s2, ok := s.(string)
	if !ok {
		t.Error("Wrong type")
	}
	if s2 != "person3" {
		t.Errorf("Expected %s, got %s", "person3", s2)
	}

	s, err = Build(rows, sType)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		log.Fatal(err)
	}
}

func TestBuildSqlitePrimitivePtr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sqlite test in short mode")
	}
	os.Remove("./gdb.db")

	db, err := sql.Open("sqlite3", "./gdb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	setupDb(db)

	//primitive
	stmt, err := db.Prepare("select name from product where id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query("3")
	if err != nil {
		log.Fatal(err)
	}
	sType := reflect.TypeOf((*string)(nil))
	s, err := Build(rows, sType)
	if err != nil {
		t.Error(err)
	}
	if s == nil {
		t.Error("Got nil back, expected a string")
	}
	s2, ok := s.(*string)
	if !ok {
		t.Error("Wrong type")
	} else {
		if *s2 != "person3" {
			t.Errorf("Expected %s, got %s", "person3", s2)
		}
	}

	s, err = Build(rows, sType)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		log.Fatal(err)
	}
}

func TestBuildSqliteMap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sqlite test in short mode")
	}
	os.Remove("./gdb.db")

	db, err := sql.Open("sqlite3", "./gdb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	setupDb(db)

	rows, err := db.Query("select id, name, cost from product")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var m map[string]interface{}

	mType := reflect.TypeOf(m)

	for i := 0; i < 100; i++ {
		prod, err := Build(rows, mType)
		if err != nil {
			t.Fatal(err)
		}
		m2, ok := prod.(map[string]interface{})
		if !ok {
			t.Error("wrong type")
		} else {
			id, ok := m2["id"]
			if !ok {
				t.Errorf("id map value not found")
			} else {
				if id != int64(i) {
					t.Errorf("Wrong id, expected %d, got %d, existed: %v", i, id, ok)
				}
			}
			name, ok := m2["name"]
			if !ok {
				t.Errorf("name map value not found")
			} else {
				if string(name.([]byte)) != fmt.Sprintf("person%d", i) {
					t.Errorf("Wrong name, expected %s, got %s, existed: %v", fmt.Sprintf("person%d", i), name, ok)
				}
			}
			cost, ok := m2["cost"]
			if !ok {
				t.Errorf("cost map value not found")
			} else {
				if cost.(float64) != 1.1*float64(i) {
					t.Errorf("Wrong cost, expected %f, got %f", 1.1*float64(i), cost)
				}
			}
		}
		//fmt.Println(p2)
	}
	err = rows.Err()
	if err != nil {
		t.Error(err)
	}
	if rows.Next() {
		t.Error("Expected no more rows, but had some")
	}
	prod, err := Build(rows, mType)
	if prod != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}
}
