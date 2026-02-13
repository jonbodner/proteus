package proteus

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"testing"

	"github.com/jonbodner/proteus/cmp"
	"github.com/jonbodner/proteus/mapper"
	"github.com/jonbodner/stackerr"
)

func TestMapRows(t *testing.T) {
	ctx := context.Background()
	//todo
	b, _ := mapper.MakeBuilder(ctx, reflect.TypeOf(10))
	v, err := mapRows(ctx, nil, b)
	if v != nil {
		t.Error("Expected nil when passing in nil rows")
	}
	eExp := stackerr.New("rows must be non-nil")
	if !cmp.Errors(err, eExp) {
		t.Errorf("Expected error %s, got %s", eExp, err)
	}
}

func setupDb(t *testing.T) *sql.DB {
	ctx := context.Background()
	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}

	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	sqlStmt := `
		drop table if exists product; 
		create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		slog.Log(ctx, slog.LevelError, "err", "error", slog.AnyValue(err), "query", slog.StringValue(sqlStmt))
		t.FailNow()
	}

	tx, err := db.Begin()
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer tx.Commit()
	stmt, err := tx.Prepare("insert into product(id, name, cost) values($1, $2, $3)")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
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
			slog.Error("err", "error", slog.AnyValue(err))
			t.FailNow()
		}
	}
	return db
}

func TestBuildStruct(t *testing.T) {
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
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer rows.Close()
	pType := reflect.TypeOf((*Product)(nil)).Elem()
	ctx := context.Background()
	b, _ := mapper.MakeBuilder(ctx, pType)
	for i := 0; i < 5; i++ {
		prod, err := mapRows(ctx, rows, b)
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
	prod, err := mapRows(ctx, rows, b)
	if prod != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}
}

func TestBuildPrimitive(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer stmt.Close()

	rows, err := stmt.Query("4")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	sType := reflect.TypeOf("")
	ctx := context.Background()
	b, _ := mapper.MakeBuilder(ctx, sType)
	s, err := mapRows(ctx, rows, b)
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

	s, err = mapRows(ctx, rows, b)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
}

func TestBuildPrimitiveNilFail(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer stmt.Close()

	rows, err := stmt.Query("3")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	sType := reflect.TypeOf("")
	ctx := context.Background()
	b, _ := mapper.MakeBuilder(ctx, sType)
	_, err = mapRows(ctx, rows, b)
	if err == nil {
		t.Error("Expected error didn't get one")
	}
	if err.Error() != "attempting to return nil for non-pointer type string" {
		t.Errorf("Expected error message '%s', got '%s'", "attempting to return nil for non-pointer type string", err.Error())
	}

	s, err := mapRows(ctx, rows, b)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
}

func TestBuildPrimitivePtr(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer stmt.Close()

	rows, err := stmt.Query("4")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	sType := reflect.TypeOf((*string)(nil))
	ctx := context.Background()
	b, _ := mapper.MakeBuilder(ctx, sType)
	s, err := mapRows(ctx, rows, b)
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

	s, err = mapRows(ctx, rows, b)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
}

func TestBuildPrimitivePtrNil(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	//primitive
	stmt, err := db.Prepare("select name from product where id = $1")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer stmt.Close()

	rows, err := stmt.Query("3")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	sType := reflect.TypeFor[*string]()
	ctx := context.Background()
	b, _ := mapper.MakeBuilder(ctx, sType)
	s, err := mapRows(ctx, rows, b)
	if err != nil {
		t.Error(err)
	}
	s2, ok := s.(*string)
	if !ok {
		t.Errorf("Wrong type: %T", s)
	} else {
		if s2 != nil {
			t.Errorf("Expected nil, got %T %v", s2, *s2)
		}
	}

	s, err = mapRows(ctx, rows, b)
	if s != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}

	_, err = db.Exec("delete from product")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
}

func TestBuildMap(t *testing.T) {
	db := setupDb(t)
	defer db.Close()

	rows, err := db.Query("select id, name, cost from product")
	if err != nil {
		slog.Error("err", "error", slog.AnyValue(err))
		t.FailNow()
	}
	defer rows.Close()

	var m map[string]interface{}

	mType := reflect.TypeOf(m)
	ctx := context.Background()
	b, _ := mapper.MakeBuilder(ctx, mType)
	for i := 0; i < 5; i++ {
		prod, err := mapRows(ctx, rows, b)
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
	prod, err := mapRows(ctx, rows, b)
	if prod != nil || err != nil {
		t.Error("Expected to be at end, but wasn't")
	}
}
