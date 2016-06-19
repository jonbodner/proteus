package gdb

import (
	"testing"
	"reflect"
	"fmt"
	"errors"
)

func TestValidIdentifier(t *testing.T) {
	values := map[string]bool{
		"a":           true,
		"main":        true,
		"int":         true, //can't tell a type from an ident
		"a b":         false,
		"a.b":         true,
		"a . b":       true, //surprising but legal
		"a..b":        false,
		".a.b":        false,
		"":            false,
		"a.b.c.d.e.f": true,
		"a.b.":        false,
		"ab;":         false,
		"a.b //comment": true, //yeah, we can do comments
	}
	for k, v := range values {
		if _, err := validIdentifier(k); v != (err == nil) {
			t.Errorf("failed for %s == %v ", k, v)
		}
	}
}

func TestConvertToPositionalParameters(t *testing.T) {
	type inner struct {
		query      string
		paramNames []string
		err        error
	}
	values := map[string]inner{
		`select * from Product where id = :id:`: inner{
			"select * from Product where id = ?",
			[]string{"id"},
			nil,
		},
		`update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:`: inner{
			"update Product set name = ?, cost = ? where id = ?",
			[]string{"p.Name", "p.Cost", "p.Id"},
			nil,
		},
		`select * from Product where name=:name: and cost=:cost:`: inner{
			"select * from Product where name=? and cost=?",
			[]string{"name", "cost"},
			nil,
		},
		//forget ending :
		`select * from Product where name=:name: and cost=:cost`: inner{
			"",
			nil,
			fmt.Errorf("Missing a closing : somewhere: %s", `select * from Product where name=:name: and cost=:cost`),
		},
		//empty ::
		`select * from Product where name=:: and cost=:cost`: inner{
			"",
			nil,
			errors.New("Empty variable declaration at position 34"),
		},
		//invalid identifier
		`select * from Product where name=:a,b,c: and cost=:cost`: inner{
			"",
			nil,
			errors.New("Invalid character found in identifier: a,b,c"),
		},
		//escaped character (invalid sql, but not the problem at hand)
		`select * from Pr\:oduct where name=:name: and cost=:cost:`: inner{
			"select * from Pr:oduct where name=? and cost=?",
			[]string{"name", "cost"},
			nil,
		},
	}

	for k, v := range values {
		q, pn, err := convertToPositionalParameters(k)
		if q != v.query || !reflect.DeepEqual(pn, v.paramNames) ||!equalErrors(err, v.err) {
			t.Errorf("failed for %s -> %v: {%v %v %v}", k, v, q, pn, err)
		}
	}
}

func TestBuildParamMap(t *testing.T) {
	values := map[string]map[string]int {
		"name,cost": map[string]int {
			"name":1,
			"cost":2,
		},
	}
	for k, v := range values {
		pm := buildParamMap(k)
		if !reflect.DeepEqual(pm, v) {
			t.Errorf("failed for %s -> %v: %v",k,v,pm)
		}
	}
}

/*
type paramInfo struct {
	name        string
	posInParams int
}

// key == position in query
// value == name to evaluate & position in function in parameters
type queryParams []paramInfo

func buildQueryParams(posNameMap []string, paramMap map[string]int, funcType reflect.Type) (queryParams, error)
 */
func TestBuildQueryParams(t *testing.T) {
	paramMap := map[string]int{
		"id": 2,
		"p":1,
	}
	type inner struct {
		Name string
		Count int
	}
	var f func(Executor,inner, int) (int, error)
	funcType := reflect.TypeOf(f)

	type vals struct {
		input []string
		output queryParams
		err error
	}

	values := []vals {
		{
			[]string{"id","p.Name","p.Count"},
			queryParams{{"id", 2},{"p.Name",1}, {"p.Count",1} },
			nil,
		},
		{
			[]string{"id.X","p.Name","p.Count"},
			nil,
			errors.New("Query Parameter id.X has a path, but the incoming parameter is not a map or a struct"),
		},
		{
			[]string{"id","p.Name","p.Count","bob"},
			nil,
			errors.New("Query Parameter bob cannot be found in the incoming parameters"),
		},
	}

	for _, v := range values {
		qp, err := buildQueryParams(v.input, paramMap, funcType)
		if !reflect.DeepEqual(qp, v.output) || !equalErrors(err, v.err) {
			t.Errorf("Expected %v -> %v %v, got %v %v",v.input, v.output, v.err, qp, err)
		}
	}
}

func equalErrors(e1, e2 error) bool {
	if e1 == nil || e2 == nil {
		if e1 != nil || e2 != nil {
			return false
		}
		return true
	}
	return e1.Error() == e2.Error()
}