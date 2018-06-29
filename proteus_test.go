package proteus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"database/sql"
	"os"

	"time"

	"github.com/jonbodner/proteus/cmp"
	"github.com/jonbodner/proteus/logger"
)

func TestValidIdentifier(t *testing.T) {
	c := logger.WithLevel(context.Background(), logger.DEBUG)
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
		if _, err := validIdentifier(c, k); v != (err == nil) {
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

	c := logger.WithLevel(context.Background(), logger.DEBUG)
	for k, v := range values {
		q, qps, err := buildFixedQueryAndParamOrder(c, k, v.paramMap, v.funcType, MySQL)
		var qSimple string
		if err == nil {
			qSimple, _ = q.finalize(c, nil)
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
	os.Remove("./proteus_test.db")

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
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

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

func TestUnnamedStructs(t *testing.T) {
	os.Remove("./proteus_test.db")

	type ScannerProduct struct {
		Id   int    `prof:"id"`
		Name string `prof:"name"`
	}

	type ScannerProductDao struct {
		Insert   func(e Executor, p ScannerProduct) (int64, error) `proq:"insert into Product(name) values(:$1.Name:)"`
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
	defer db.Close()

	exec, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		t.Fatal(err)
	}

	gExec := Wrap(exec)

	p := ScannerProduct{
		Name: "bob",
	}

	rowId, err := productDao.Insert(gExec, p)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := productDao.FindById(gExec, rowId)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Id != 1 || roundTrip.Name != "bob" {
		t.Errorf("Expected {1 bob}, got %v", roundTrip)
	}
}

func TestEmbedded(t *testing.T) {
	os.Remove("./proteus_test.db")

	type InnerEmbeddedProductDao struct {
		Insert func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:p.Name:)" prop:"p"`
	}

	type OuterEmbeddedProductDao struct {
		InnerEmbeddedProductDao
		FindById func(e Querier, id int64) (Product, error) `proq:"select * from Product where id = :id:" prop:"id"`
	}

	productDao := OuterEmbeddedProductDao{}
	err := Build(&productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100))")
	if err != nil {
		t.Fatal(err)
	}

	gExec := Wrap(exec)

	p := Product{
		Name: "Bob",
	}

	rowId, err := productDao.Insert(gExec, p)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := productDao.FindById(gExec, rowId)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Id != 1 || roundTrip.Name != "Bob" {
		t.Errorf("Expected {1 Bob}, got %v", roundTrip)
	}
}

func TestShouldBuildEmbeddedWithNullField(t *testing.T) {

	type MyProduct struct {
		Id         int            `prof:"id"`
		Name       string         `prof:"name"`
		EmptyField sql.NullString `prof:"empty_field"`
	}

	type Inner struct {
		Name       string         `prof:"name"`
		EmptyField sql.NullString `prof:"empty_field"`
	}

	type MyNestedProduct struct {
		Inner
		Id int `prof:"id"`
	}

	type ProductDao struct {
		Insert    func(e Executor, p MyProduct) (int64, error)          `proq:"insert into product(name,empty_field) values(:p.Name:,:p.EmptyField:)" prop:"p"`
		Get       func(q Querier, name string) (MyProduct, error)       `proq:"select * from product where name=:name:" prop:"name"`
		GetNested func(q Querier, name string) (MyNestedProduct, error) `proq:"select * from product where name=:name:" prop:"name"`
		// GetNested has the same query as "Get" but with embedded structure. Currently fails with null values
	}

	productDao := ProductDao{}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
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

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100),empty_field VARCHAR(100))")

	if err != nil {
		t.Fatal(err)
	}

	count, err := productDao.Insert(Wrap(exec), MyProduct{Name: "foo", EmptyField: sql.NullString{String: "", Valid: false}})
	// Nullable field with non-null values work fine, e.g. line below
	//count, err := productDao.Insert(Wrap(exec), MyProduct{Name: "foo", EmptyField: sql.NullString{String:"field",Valid: true}})

	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("Should have modified 1 row")
	}
	prod, err := productDao.Get(Wrap(exec), "foo")
	if err != nil {
		t.Fatal(err)
	}

	if prod.Name != "foo" {
		t.Fatal(fmt.Sprintf("Expected prod with name of foo, got %+v", prod))
	}

	// This is currently failing
	nestedProd, err := productDao.GetNested(Wrap(exec), "foo")
	if err != nil {
		t.Fatal(err)
	}
	if nestedProd.Name != "foo" {
		t.Fatal(fmt.Sprintf("Expected nested product name of foo, got %+v", prod))
	}

}

func TestPositionalVariables(t *testing.T) {

	type PProduct struct {
		Name string
		E6   string
		Egg  string
	}

	// should be valid
	type ProductDao struct {
		Insert  func(e Executor, p PProduct) (int64, error) `proq:"insert into Product(name) values(:$1.Name:)"`
		Insert2 func(e Executor, p PProduct) (int64, error) `proq:"insert into Product(name) values(:$1.E6:)"`
		Insert3 func(e Executor, p PProduct) (int64, error) `proq:"insert into Product(name) values(:$1.Egg:)"`
	}

	productDao := ProductDao{}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
	if err != nil {
		t.Error(err)
	}
}

func TestShouldBuild(t *testing.T) {
	//test single problem, invalid query (missing closing :)
	type ProductDao struct {
		Insert func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:p.Name)" prop:"p"`
	}

	productDao := ProductDao{}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
	if err == nil {
		t.Fatal("This should have errors")
	}
	if err.Error() != "error in field #0 (Insert): missing a closing : somewhere: insert into Product(name) values(:p.Name)" {
		t.Error(err)
	}

	// test three problems, invalid query (missing closing :)
	//invalid function
	//no query for query mapper
	type ProductDao2 struct {
		Insert      func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:p.Name)" prop:"p"`
		Insert2     func(p Product) (int64, error)             `proq:"insert into Product(name) values(:p.Name:)" prop:"p"`
		I           int
		Insert3     func(e Executor, p Product) (int64, error) `proq:"q:nope" prop:"p"`
		InsertNoQ   func(e Executor, p Product) (int64, error)
		InsertNoP   func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:p.Name:)"`
		InsertOK    func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:p.Name:)" prop:"p"`
		InsertOKNoP func(e Executor, p Product) (int64, error) `proq:"insert into Product(name) values(:$1.Name:)"`
	}

	productDao2 := ProductDao2{}
	err2 := ShouldBuild(c, &productDao2, Sqlite)
	if err2 == nil {
		t.Error(err2)
	}
	if err2.Error() != `error in field #0 (Insert): missing a closing : somewhere: insert into Product(name) values(:p.Name)
error in field #1 (Insert2): first parameter must be of type Executor or Querier
error in field #3 (Insert3): no query found for name nope
error in field #5 (InsertNoP): query Parameter p cannot be found in the incoming parameters` {
		t.Error(err2)
	}
}

func TestShouldBuildEmbedded(t *testing.T) {
	os.Remove("./proteus_test.db")

	type Inner struct {
		Name string `prof:"name"`
	}
	type MyProduct struct {
		Inner
		Id int `prof:"id"`
	}
	type ProductDao struct {
		Insert func(e Executor, p MyProduct) (int64, error)    `proq:"insert into product(name) values(:p.Name:)" prop:"p"`
		Get    func(q Querier, name string) (MyProduct, error) `proq:"select * from product where name=:name:" prop:"name"`
	}

	productDao := ProductDao{}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100), null_field VARCHAR(100))")
	if err != nil {
		t.Fatal(err)
	}

	count, err := productDao.Insert(Wrap(exec), MyProduct{Inner: Inner{Name: "foo"}})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("Should have modified 1 row")
	}
	prod, err := productDao.Get(Wrap(exec), "foo")
	if err != nil {
		t.Fatal(err)
	}
	if prod.Name != "foo" {
		t.Fatal(fmt.Sprintf("Expected prod with name, got %+v", prod))
	}
}

func TestShouldBinaryColumn(t *testing.T) {
	os.Remove("./proteus_test.db")

	type MyProduct struct {
		Id   int    `prof:"id"`
		Name string `prof:"name"`
		Data []byte `prof:"data"`
	}

	type ProductDao struct {
		Insert func(e Executor, p MyProduct) (int64, error)    `proq:"insert into product(name, data) values(:p.Name:, :p.Data:)" prop:"p"`
		Get    func(q Querier, name string) (MyProduct, error) `proq:"select * from product where name=:name:" prop:"name"`
	}

	productDao := ProductDao{}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100), data blob)")
	if err != nil {
		t.Fatal(err)
	}

	count, err := productDao.Insert(Wrap(exec), MyProduct{Name: "Foo", Data: []byte("Hello")})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("Should have modified 1 row")
	}
	prod, err := productDao.Get(Wrap(exec), "Foo")
	if err != nil {
		t.Fatal(err)
	}
	if prod.Name != "Foo" {
		t.Fatal(fmt.Sprintf("Expected prod with name, got %+v", prod))
	}
	if string(prod.Data) != "Hello" {
		t.Fatal(fmt.Sprintf("Expected prod with data, got %+v", prod))
	}
}

func TestShouldTimeColumn(t *testing.T) {
	os.Remove("./proteus_test.db")

	type MyProduct struct {
		Id        int       `prof:"id"`
		Name      string    `prof:"name"`
		Timestamp time.Time `prof:"timestamp"`
	}

	type ProductDao struct {
		Insert func(e Executor, p MyProduct) (int64, error)    `proq:"insert into product(name, timestamp) values(:p.Name:, :p.Timestamp:)" prop:"p"`
		Get    func(q Querier, name string) (MyProduct, error) `proq:"select * from product where name=:name:" prop:"name"`
	}

	productDao := ProductDao{}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	err := ShouldBuild(c, &productDao, Sqlite)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exec, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	_, err = exec.Exec("CREATE TABLE product(id INTEGER PRIMARY KEY, name VARCHAR(100), timestamp datetime)")
	if err != nil {
		t.Fatal(err)
	}

	timestamp := time.Now()
	count, err := productDao.Insert(Wrap(exec), MyProduct{Name: "Foo", Timestamp: timestamp})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("Should have modified 1 row")
	}
	prod, err := productDao.Get(Wrap(exec), "Foo")
	if err != nil {
		t.Fatal(err)
	}
	if prod.Name != "Foo" {
		t.Fatal(fmt.Sprintf("Expected prod with name, got %+v", prod))
	}

	// crazy but true: intermediate formats can create different format outputs.
	// using a -07:00 for the time zone here, and then turning it back into a time, and then formatting it with
	// time.UnixDate produces different output than if you formatted with UnixDate, parsed it, and then formatted it again.
	// WAT
	s := timestamp.Format("2006-01-02 15:04:05.999999999-07:00")
	t2, _ := time.ParseInLocation("2006-01-02 15:04:05.999999999-07:00", s, time.UTC)
	fmt.Println(prod.Timestamp.Location(), prod.Timestamp.Format(time.UnixDate))
	fmt.Println(t2.Location(), t2.Format(time.UnixDate))
	if prod.Timestamp.Format(time.UnixDate) != t2.Format(time.UnixDate) {
		t.Fatal(fmt.Sprintf("Expected prod with timestamp, got %+v", prod))
	}
}
