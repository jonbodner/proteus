package proteus

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"testing"

	"github.com/jonbodner/proteus/cmp"
	"github.com/jonbodner/proteus/logger"
	"github.com/jonbodner/proteus/mapper"
)

func TestMapRows(t *testing.T) {
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	//todo
	b, _ := mapper.MakeBuilder(c, reflect.TypeOf(10))
	v, err := mapRows(c, nil, b)
	if v != nil {
		t.Error("Expected nil when passing in nil rows")
	}
	eExp := errors.New("rows must be non-nil")
	if !cmp.Errors(err, eExp) {
		t.Errorf("Expected error %s, got %s", eExp, err)
	}
}

func setupDb(t *testing.T) *sql.DB {
	if testing.Short() {
		t.Skip("skipping sqlite test in short mode")
	}

	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
		drop table if exists product; 
		create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		panic(err)
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Commit()
	stmt, err := tx.Prepare("insert into product(id, name, cost) values($1, $2, $3)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 5; i++ {
		var name *string
		if i%2 == 0 {
			n := fmt.Sprintf("person%d", i)
			name = &n
		}
		_, err = stmt.Exec(i, name, 1.1*float64(i))
		if err != nil {
			log.Fatal(err)
		}
	}
	return db
}

func TestBuildSqliteStruct(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//struct
	type Product struct {
		ID   int     `prof:"id,pk"`
		Name *string `prof:"name"`
		Cost float64 `prof:"cost"`
	}

	rows, err := db.Query("select id, name, cost from product")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	pType := reflect.TypeOf((*Product)(nil)).Elem()
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	b, _ := mapper.MakeBuilder(c, pType)
	for i := 0; i < 5; i++ {
		prod, err := mapRows(c, rows, b)
		if err != nil {
			t.Fatal(err)
		}
		p2, ok := prod.(Product)
		if !ok {
			t.Error("wrong type")
		} else {
			if p2.ID != i {
				t.Errorf("Wrong id, expected %d, got %d", i, p2.ID)
			}
			if i%2 == 0 {
				if *p2.Name != fmt.Sprintf("person%d", i) {
					t.Errorf("Wrong name, expected %s, got %s", fmt.Sprintf("person%d", i), *p2.Name)
				}
			} else {
				if p2.Name != nil {
					t.Errorf("Wrong name, expected nil, got %v", p2.Name)
				}
			}
			if math.Abs(p2.Cost-1.1*float64(i)) > 0.01 {
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
	prod, err := mapRows(c, rows, b)
	if prod != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}
}

func TestBuildSqlitePrimitive(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query("4")
	if err != nil {
		log.Fatal(err)
	}
	sType := reflect.TypeOf("")
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	b, _ := mapper.MakeBuilder(c, sType)
	s, err := mapRows(c, rows, b)
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
	if s2 != "person4" {
		t.Errorf("Expected %s, got %s", "person4", s2)
	}

	s, err = mapRows(c, rows, b)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		log.Fatal(err)
	}
}

func TestBuildSqlitePrimitiveNilFail(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query("3")
	if err != nil {
		log.Fatal(err)
	}
	sType := reflect.TypeOf("")
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	b, _ := mapper.MakeBuilder(c, sType)
	s, err := mapRows(c, rows, b)
	if err == nil {
		t.Error("Expected error didn't get one")
	}
	if err.Error() != "attempting to return nil for non-pointer type string" {
		t.Errorf("Expected error message '%s', got '%s'", "attempting to return nil for non-pointer type string", err.Error())
	}

	s, err = mapRows(c, rows, b)
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
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query("4")
	if err != nil {
		log.Fatal(err)
	}
	sType := reflect.TypeOf((*string)(nil))
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	b, _ := mapper.MakeBuilder(c, sType)
	s, err := mapRows(c, rows, b)
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
		if *s2 != "person4" {
			t.Errorf("Expected %s, got %s", "person4", *s2)
		}
	}

	s, err = mapRows(c, rows, b)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		log.Fatal(err)
	}
}

func TestBuildSqlitePrimitivePtrNil(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	rows, err := stmt.Query("3")
	if err != nil {
		log.Fatal(err)
	}
	sType := reflect.TypeOf((*string)(nil))
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	b, _ := mapper.MakeBuilder(c, sType)
	s, err := mapRows(c, rows, b)
	if err != nil {
		t.Error(err)
	}
	s2, ok := s.(*string)
	if !ok {
		t.Error("Wrong type")
	} else {
		if s2 != nil {
			t.Errorf("Expected nil, got %v %v", reflect.TypeOf(s2), *s2)
		}
	}

	s, err = mapRows(c, rows, b)
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
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	b, _ := mapper.MakeBuilder(c, mType)
	for i := 0; i < 5; i++ {
		prod, err := mapRows(c, rows, b)
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
			if i%2 == 0 {
				//should have a name
				if !ok {
					t.Errorf("name map value not found")
				}
				if name.(string) != fmt.Sprintf("person%d", i) {
					t.Errorf("Wrong name, expected %s, got %s, existed: %v", fmt.Sprintf("person%d", i), name, ok)
				}
			} else {
				if ok {
					t.Errorf("name map value should not be found")
				}
			}
			cost, ok := m2["cost"]
			if !ok {
				t.Errorf("cost map value not found")
			} else {
				if math.Abs(cost.(float64)-1.1*float64(i)) > 0.01 {
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
	prod, err := mapRows(c, rows, b)
	if prod != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}
}
