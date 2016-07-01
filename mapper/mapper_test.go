package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jonbodner/proteus/cmp"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"reflect"
	"testing"
)

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

func setupDb(t *testing.T) *sql.DB {
	if testing.Short() {
		t.Skip("skipping sqlite test in short mode")
	}
	os.Remove("./proteus_test.db")

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
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
	return db
}

func TestBuildSqliteStruct(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//struct
	type Product struct {
		Id   int     `prof:"id,pk"`
		Name *string `prof:"name"`
		Cost float64 `prof:"cost"`
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
	db := setupDb(t)
	defer db.Close()

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
	db := setupDb(t)
	defer db.Close()

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
	db := setupDb(t)
	defer db.Close()

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
