package proteus

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	"github.com/jonbodner/proteus/cmp"
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
	var f1 func(api.Executor, int)
	type pp struct {
		Name string
		Cost float64
		Id   int
	}

	var f2 func(api.Executor, pp)
	var f3 func(api.Executor, string, float64)

	type inner struct {
		paramMap map[string]int
		funcType reflect.Type
		query    string
		qps      queryParams
		err      error
	}
	values := map[string]inner{
		`select * from Product where id = :id:`: inner{
			map[string]int{"id": 1},
			reflect.TypeOf(f1),
			"select * from Product where id = ?",
			queryParams{{"id", 1}},
			nil,
		},
		`update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:`: inner{
			map[string]int{"p": 1},
			reflect.TypeOf(f2),
			"update Product set name = ?, cost = ? where id = ?",
			queryParams{{"p.Name", 1}, {"p.Cost", 1}, {"p.Id", 1}},
			nil,
		},
		`select * from Product where name=:name: and cost=:cost:`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"select * from Product where name=? and cost=?",
			queryParams{{"name", 1}, {"cost", 2}},
			nil,
		},
		//forget ending :
		`select * from Product where name=:name: and cost=:cost`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"",
			nil,
			fmt.Errorf("Missing a closing : somewhere: %s", `select * from Product where name=:name: and cost=:cost`),
		},
		//empty ::
		`select * from Product where name=:: and cost=:cost`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"",
			nil,
			errors.New("Empty variable declaration at position 34"),
		},
		//invalid identifier
		`select * from Product where name=:a,b,c: and cost=:cost`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"",
			nil,
			errors.New("Invalid character found in identifier: a,b,c"),
		},
		//escaped character (invalid sql, but not the problem at hand)
		`select * from Pr\:oduct where name=:name: and cost=:cost:`: inner{
			map[string]int{"name": 1, "cost": 2},
			reflect.TypeOf(f3),
			"select * from Pr:oduct where name=? and cost=?",
			queryParams{{"name", 1}, {"cost", 2}},
			nil,
		},
	}

	for k, v := range values {
		q, qps, _, err := convertToPositionalParameters(k, v.paramMap, v.funcType, adapter.MySQL)
		if q.simple != v.query || !reflect.DeepEqual(qps, v.qps) || !cmp.Errors(err, v.err) {
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
		pm := buildParamMap(k)
		if !reflect.DeepEqual(pm, v) {
			t.Errorf("failed for %s -> %v: %v", k, v, pm)
		}
	}
}

func TestValidateFunction(t *testing.T) {
	f := func(fType reflect.Type, isExec bool, msg string) {
		err := validateFunction(fType, isExec)
		if err == nil {
			t.Fatalf("Expected err")
		}
		eExp := errors.New(msg)
		if !cmp.Errors(err, eExp) {
			t.Errorf("Wrong error expected %s, got %s", eExp, err)
		}
	}

	fOk := func(fType reflect.Type, isExec bool) {
		err := validateFunction(fType, isExec)
		if err != nil {
			t.Errorf("Unexpected err %s", err)
		}
	}

	//invalid -- no parameters
	var f1 func()
	f(reflect.TypeOf(f1), true, "Need to supply an Executor parameter")
	f(reflect.TypeOf(f1), false, "Need to supply an Executor parameter")

	//invalid -- wrong first parameter type
	var f2 func(int)
	f(reflect.TypeOf(f2), true, "First parameter must be of type api.Executor")
	f(reflect.TypeOf(f2), false, "First parameter must be of type api.Executor")

	//invalid -- has a channel input param
	var f3 func(api.Executor, chan int)
	f(reflect.TypeOf(f3), true, "no input parameter can be a channel")
	f(reflect.TypeOf(f3), false, "no input parameter can be a channel")

	//valid -- only an Executor
	var g1 func(api.Executor)
	fOk(reflect.TypeOf(g1), true)
	fOk(reflect.TypeOf(g1), false)

	//valid -- an Executor and a primitive
	var g2 func(api.Executor, int)
	fOk(reflect.TypeOf(g2), true)
	fOk(reflect.TypeOf(g2), false)

	//valid -- an Executor, a primitive, a map and a struct
	var g3 func(api.Executor, int, map[string]interface{}, struct {
		A int
		B string
	})
	fOk(reflect.TypeOf(g3), true)
	fOk(reflect.TypeOf(g3), false)

	//valid -- an Executor, a primitive, a map and a struct, returning a struct and error
	// for query only
	var g4 func(api.Executor, int, map[string]interface{}, struct {
		A int
		B string
	}) (struct {
		C string
		D bool
	}, error)
	fOk(reflect.TypeOf(g4), false)
	//invalid for Exec
	f(reflect.TypeOf(g4), true, "The 1st output parameter of an Exec must be int64")
}
