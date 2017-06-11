package proteus

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

type NoErrType string

func (NoErrType) Error() string {
	return "NOERR"
}

type DummyDB struct {
	pos     int
	Queries []string
	Args    [][]interface{}
}

func (dd *DummyDB) Query(query string, args ...interface{}) (Rows, error) {
	return nil, dd.checkExpectedData(query, args...)
}

func (dd *DummyDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, dd.checkExpectedData(query, args...)
}

func (dd *DummyDB) checkExpectedData(query string, args ...interface{}) error {
	if dd.pos >= len(dd.Queries) || dd.pos >= len(dd.Args) {
		return fmt.Errorf("Expected at least %d queries and args, only have %d queries and %d args", dd.pos, len(dd.Queries), len(dd.Args))
	}
	var msg string
	if dd.Queries[dd.pos] != query {
		msg = fmt.Sprintf("Expected query %s, got query %s", dd.Queries[dd.pos], query)
	}
	for k, v := range dd.Args[dd.pos] {
		if k >= len(args) {
			msg = msg + fmt.Sprintf(" Expected at least %d args to be passed in, got %d", k, len(args))
			break
		}
		if v != args[k] {
			msg = msg + fmt.Sprintf(" Expected arg %v at pos %d, got %v", v, k, args[k])
		}
	}
	dd.pos = dd.pos + 1
	if len(msg) == 0 {
		return NoErrType("")
	}
	return errors.New(msg)
}

func TestMapMapper(t *testing.T) {
	m := MapMapper{
		"q1": "select * from foo where id = :id:",
		"q2": "update foo set x=:x: where id = :id:",
	}

	runMapper(m, t)
}

func TestPropMapper(t *testing.T) {
	m, err := PropFileToQueryMapper("test.properties")
	if err != nil {
		t.Fatal(err)
	}
	runMapper(m, t)
}

func runMapper(m QueryMapper, t *testing.T) {

	type f struct {
		Id string
		X  string
	}
	type s struct {
		GetF   func(e Querier, id string) (f, error)                `proq:"q:q1" prop:"id"`
		Update func(e Executor, id string, x string) (int64, error) `proq:"q:q2" prop:"id,x"`
	}
	sImpl := s{}
	err := Build(&sImpl, Sqlite, m)
	if err != nil {
		t.Error("error while building", err)
	}

	dummyDB := &DummyDB{
		Queries: []string{
			"select * from foo where id = ?",
			"update foo set x=? where id = ?",
		},
		Args: [][]interface{}{
			[]interface{}{"1"},
			[]interface{}{"Hello", "2"},
		},
	}

	_, err = sImpl.GetF(dummyDB, "1")
	if _, ok := err.(NoErrType); !ok {
		t.Errorf("Expected no error, got %v", err)
	}
	_, err = sImpl.Update(dummyDB, "2", "Hello")
	if _, ok := err.(NoErrType); !ok {
		t.Errorf("Expected no error, got %v", err)
	}
}
