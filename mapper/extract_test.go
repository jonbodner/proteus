package mapper

import (
	"testing"
	"errors"
	"github.com/jonbodner/proteus/cmp"
	"reflect"
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

