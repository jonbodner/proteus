package proteus

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/jonbodner/proteus/cmp"
	"database/sql"
)

func TestValidIdentifier(t *testing.T) {
	values := map[string]bool{
		"a":             true,
		"main":          true,
		"int":           true, //can't tell a type from an ident
		"a b":           false,
		"a.b":           true,
		"a . b":         true, //surprising but legal
		"a..b":          false,
		".a.b":          false,
		"":              false,
		"a.b.c.d.e.f":   true,
		"a.b.":          false,
		"ab;":           false,
		"a.b //comment": true, //yeah, we can do comments
	}
	for k, v := range values {
		if _, err := validIdentifier(k); v != (err == nil) {
			t.Errorf("failed for %s == %v ", k, v)
		}
	}
}

func TestConvertToPositionalParameters(t *testing.T) {
	var f1 func(Executor, int)
	type pp struct {
		Name string
		Cost float64
		Id   int
	}

	var f2 func(Executor, pp)
	var f3 func(Executor, string, float64)

	type inner struct {
		paramMap map[string]int
		funcType reflect.Type
		query    string
		qps      []paramInfo
		err      error
	}
	values := map[string]inner{
		`select * from Product where id = :id:`: inner{
			map[string]int{"id": 1},
			reflect.TypeOf(f1),
			"select * from Product where id = ?",
			[]paramInfo{{"id", 1, false}},
			nil,
		},
		`update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:`: inner{
			map[string]int{"p": 1},
			reflect.TypeOf(f2),
			"update Product set name = ?, cost = ? where id = ?",
			[]paramInfo{{"p.Name", 1, false}, {"p.Cost", 1, false}, {"p.Id", 1, false}},
			nil,
		},
		`select * from Product where name=:name: and cost=:cost:`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"select * from Product where name=? and cost=?",
			[]paramInfo{{"name", 1, false}, {"cost", 2, false}},
			nil,
		},
		//forget ending :
		`select * from Product where name=:name: and cost=:cost`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"",
			nil,
			fmt.Errorf("missing a closing : somewhere: %s", `select * from Product where name=:name: and cost=:cost`),
		},
		//empty ::
		`select * from Product where name=:: and cost=:cost`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"",
			nil,
			errors.New("empty variable declaration at position 34"),
		},
		//invalid identifier
		`select * from Product where name=:a,b,c: and cost=:cost`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"",
			nil,
			errors.New("invalid character found in identifier: a,b,c"),
		},
		//escaped character (invalid sql, but not the problem at hand)
		`select * from Pr\:oduct where name=:name: and cost=:cost:`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"select * from Pr:oduct where name=? and cost=?",
			[]paramInfo{{"name", 1, false}, {"cost", 2, false}},
			nil,
		},
	}

	for k, v := range values {
		q, qps, err := buildFixedQueryAndParamOrder(k, v.paramMap, v.funcType, MySQL)
		var qSimple string
		if err == nil {
			qSimple, _ = q.finalize(nil)
		}
		if qSimple != v.query || !reflect.DeepEqual(qps, v.qps) || !cmp.Errors(err, v.err) {
			t.Errorf("failed for %s -> %#v: %v", k, v, err)
		}
	}
}

func TestBuildParamMap(t *testing.T) {
	values := map[string]map[string]int{
		"name,cost": map[string]int{
			"name": 1,
			"cost": 2,
		},
	}
	for k, v := range values {
		pm := buildNameOrderMap(k)
		if !reflect.DeepEqual(pm, v) {
			t.Errorf("failed for %s -> %v: %v", k, v, pm)
		}
	}
}

func TestValidateFunction(t *testing.T) {
	f := func(fType reflect.Type, msg string) {
		err := validateFunction(fType)
		if err == nil {
			t.Fatalf("Expected err")
		}
		eExp := errors.New(msg)
		if !cmp.Errors(err, eExp) {
			t.Errorf("Wrong error expected %s, got %s", eExp, err)
		}
	}

	fOk := func(fType reflect.Type, isExecIn bool) {
		err := validateFunction(fType)
		if err != nil {
			t.Errorf("Unexpected err %s", err)
		}
	}

	//invalid -- no parameters
	var f1 func()
	f(reflect.TypeOf(f1), "need to supply an Executor or Querier parameter")
	f(reflect.TypeOf(f1), "need to supply an Executor or Querier parameter")

	//invalid -- wrong first parameter type
	var f2 func(int)
	f(reflect.TypeOf(f2), "first parameter must be of type Executor or Querier")
	f(reflect.TypeOf(f2), "first parameter must be of type Executor or Querier")

	//invalid -- has a channel input param
	var f3 func(Executor, chan int)
	f(reflect.TypeOf(f3), "no input parameter can be a channel")
	f(reflect.TypeOf(f3), "no input parameter can be a channel")

	//valid -- only an Executor
	var g1 func(Executor)
	fOk(reflect.TypeOf(g1), true)

	var g1q func(Querier)
	fOk(reflect.TypeOf(g1q), false)

	//valid -- an Executor and a primitive
	var g2 func(Executor, int)
	fOk(reflect.TypeOf(g2), true)

	var g2q func(Querier, int)
	fOk(reflect.TypeOf(g2q), false)

	//valid -- an Executor, a primitive, a map and a struct
	var g3 func(Executor, int, map[string]interface{}, struct {
		A int
		B string
	})
	fOk(reflect.TypeOf(g3), true)

	//invalid -- an Executor, a primitive, a map and a struct, returning a struct and error
	var g4 func(Executor, int, map[string]interface{}, struct {
		A int
		B string
	}) (struct {
		C string
		D bool
	}, error)
	//invalid for Exec
	f(reflect.TypeOf(g4), "the 1st output parameter of an Executor must be int64")

	//valid for query
	var g4q func(Querier, int, map[string]interface{}, struct {
		A int
		B string
	}) (struct {
		C string
		D bool
	}, error)
	fOk(reflect.TypeOf(g4q), false)
}

func TestBuild(t *testing.T) {
	type args struct {
		dao interface{}
		pa  ParamAdapter
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		if err := Build(tt.args.dao, tt.args.pa); (err != nil) != tt.wantErr {
			t.Errorf("%q. Build() error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestNilScanner(t *testing.T) {
	type ScannerProduct struct {
		Id        int            `prof:"id"`
		Name      sql.NullString `prof:"name"`
		NullField sql.NullString `prof:"null_field"`
	}

	type ScannerProductDao struct {
		Insert   func(e Executor, p ScannerProduct) (int64, error) `proq:"insert into Product(name) values(:p.Name:)" prop:"p"`
		FindById func(e Querier, id int64) (ScannerProduct, error) `proq:"select * from Product where id = :id:" prop:"id"`
	}

	productDao := ScannerProductDao{}
	err := Build(&productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		t.Fatal(err)
	}

	exec, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100), null_field VARCHAR(100))")
	if err != nil {
		t.Fatal(err)
	}

	gExec := Wrap(exec)

	p := ScannerProduct{
		Name: sql.NullString{String: "hi", Valid: true},
	}

	rowId, err := productDao.Insert(gExec, p)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := productDao.FindById(gExec, rowId)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Id != 1 || roundTrip.Name.String != "hi" || roundTrip.Name.Valid != true || roundTrip.NullField.String != "" || roundTrip.NullField.Valid != false {
		t.Errorf("Expected {1 {hi true} { false}}, got %v", roundTrip)
	}
}