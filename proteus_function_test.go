package proteus

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jonbodner/proteus/logger"
)

func TestBuilder_BuildFunctionErrors(t *testing.T) {
	b := NewBuilder(Postgres)
	db := setupDbPostgres()
	defer db.Close()
	ctx := context.Background()

	var f func()
	var f2 func(context.Context, ContextExecutor)

	data := []struct {
		name   string
		f      interface{}
		query  string
		params []string
		errMsg string
	}{
		{
			name:   "not a pointer",
			f:      func() {},
			query:  "",
			params: nil,
			errMsg: "not a pointer",
		},
		{
			name:   "not a func",
			f:      &struct{}{},
			query:  "",
			params: nil,
			errMsg: "not a pointer to func",
		},
		{
			name:   "not a valid func",
			f:      &f,
			query:  "",
			params: nil,
			errMsg: "need to supply an Executor or Querier parameter",
		},
		{
			name:   "not a valid query",
			f:      &f2,
			query:  "q:nope",
			params: nil,
			errMsg: "no query found for name nope",
		},
		{
			name:   "not a valid params",
			f:      &f2,
			query:  "SELECT * FROM PERSON WHERE id = :id:",
			params: nil,
			errMsg: "query Parameter id cannot be found in the incoming parameters",
		},
	}
	for _, v := range data {
		t.Run(v.name, func(t *testing.T) {
			err := b.BuildFunction(ctx, v.f, v.query, v.params)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != v.errMsg {
				t.Errorf("Expected error `%s`, got `%s`", v.errMsg, errMsg)
			}
		})
	}
}

func TestBuilder_BuildFunction(t *testing.T) {
	b := NewBuilder(Postgres)
	db := setupDbPostgres()
	defer db.Close()
	ctx := context.Background()

	var f func(ctx context.Context, e ContextExecutor, name string, age int) (int64, error)
	err := b.BuildFunction(ctx, &f, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []string{"name", "age"})
	if err != nil {
		t.Fatalf("build function failed: %v", err)
	}
	rows, err := f(ctx, db, "Fred", 20)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rows != 1 {
		t.Error("Expected 1 row modified, got ", rows)
	}

	err = b.BuildFunction(ctx, &f, "INSERT INTO PERSON(name, age) VALUES(:$1:, :$2:)", nil)
	if err != nil {
		t.Fatalf("build function failed: %v", err)
	}
	rows, err = f(ctx, db, "Fred", 20)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rows != 1 {
		t.Error("Expected 1 row modified, got ", rows)
	}

	var g func(ctx context.Context, q ContextQuerier, id int) (*Person, error)
	err = b.BuildFunction(ctx, &g, "SELECT * FROM PERSON WHERE id = :id:", []string{"id"})
	if err != nil {
		t.Fatalf("build function 2 failed: %v", err)
	}

	p, err := g(ctx, db, 1)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if p == nil || *p != (Person{1, "Fred", 20}) {
		t.Error("Unexpected output: ", p)
	}

	var lines []string
	logger.Config(logger.LoggerFunc(func(vals ...interface{}) error {
		lines = append(lines, fmt.Sprintln(vals))
		return nil
	}))
	SetLogLevel(logger.DEBUG)
	err = b.BuildFunction(ctx, &g, "SELECT * FROM PERSON WHERE id = :id:", []string{"id"})
	if err != nil {
		t.Fatalf("build function 2a failed: %v", err)
	}
	if len(lines) != 3 {
		t.Errorf("Expected %d lines, got %d: %#v", 3, len(lines), lines)
	}

}

func TestBuilder_Execute(t *testing.T) {
	b := NewBuilder(Postgres)
	db := setupDbPostgres()
	defer db.Close()
	ctx := context.Background()

	var lines []string
	logger.Config(logger.LoggerFunc(func(vals ...interface{}) error {
		lines = append(lines, fmt.Sprintln(vals))
		return nil
	}))

	data := []struct {
		name      string
		ctx       context.Context
		logLevel  logger.Level
		query     string
		params    map[string]interface{}
		rows      int64
		errMsg    string
		lineCount int
	}{
		{
			name:   "insert",
			query:  "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)",
			params: map[string]interface{}{"name": "Fred", "age": 20},
			rows:   1,
		},
		{
			name:   "insert struct",
			query:  "INSERT INTO PERSON(name, age) VALUES(:p.Name:, :p.Age:)",
			params: map[string]interface{}{"p": Person{Name: "Bob", Age: 50}},
			rows:   1,
		},
		{
			name:   "insert map",
			query:  "INSERT INTO PERSON(name, age) VALUES(:p.name:, :p.age:)",
			params: map[string]interface{}{"p": map[string]interface{}{"name": "Julia", "age": 32}},
			rows:   1,
		},
		{
			name:   "insert unnamed",
			query:  "INSERT INTO PERSON(name, age) VALUES(:$1:, :$2:)",
			params: map[string]interface{}{"$1": "Pat", "$2": 37},
			rows:   1,
		},
		{
			name:   "bad query",
			query:  "INSERT INTO PERSON(name, age VALUES(:name:, :age:)",
			params: map[string]interface{}{"name": "Fred", "age": 20},
			rows:   0,
			errMsg: "pq: syntax error at or near \"VALUES\"",
		},
		{
			name:   "bad args",
			query:  "INSERT INTO PERSON(name, age) VALUES(:name stuff:, :age:)",
			params: map[string]interface{}{"name": "Fred", "age": 20},
			rows:   0,
			errMsg: ". missing between parts of an identifier: name stuff",
		},
		{
			name:   "no such query",
			query:  "q:nope",
			params: map[string]interface{}{"id": 1},
			rows:   0,
			errMsg: "no query found for name nope",
		},
		{
			name:      "logging context",
			ctx:       logger.WithLevel(ctx, logger.DEBUG),
			query:     "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)",
			params:    map[string]interface{}{"name": "Fred", "age": 20},
			rows:      1,
			lineCount: 7,
		},
		{
			name:      "logging default",
			logLevel:  logger.DEBUG,
			query:     "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)",
			params:    map[string]interface{}{"name": "Fred", "age": 20},
			rows:      1,
			lineCount: 7,
		},
	}
	for _, v := range data {
		t.Run(v.name, func(t *testing.T) {
			lines = nil
			SetLogLevel(v.logLevel)
			c := ctx
			if v.ctx != nil {
				c = v.ctx
			}
			rows, err := b.Exec(c, db, v.query, v.params)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != v.errMsg {
				t.Errorf("Expected error `%s`, got `%s`", v.errMsg, errMsg)
			}
			if rows != v.rows {
				t.Error("Unexpected # rows: ", rows)
			}
			if v.lineCount != len(lines) {
				t.Errorf("Expected %d lines of debug, got %d: %#v", v.lineCount, len(lines), lines)
			}
			lines = nil
			result, err := b.ExecResult(c, db, v.query, v.params)
			errMsg = ""
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != v.errMsg {
				t.Errorf("Expected error `%s`, got `%s`", v.errMsg, errMsg)
			}
			if err == nil {
				rowsAffected, err := result.RowsAffected()
				if err != nil {
					t.Fatal("unexpected error", err)
				}
				if rowsAffected != v.rows {
					t.Error("Unexpected # rows: ", rows)
				}
				if v.lineCount != len(lines) {
					t.Errorf("Expected %d lines of debug, got %d: %#v", v.lineCount, len(lines), lines)
				}
			}
		})
	}
}

func TestBuilder_Query(t *testing.T) {
	b := NewBuilder(Postgres)
	db := setupDbPostgres()
	defer db.Close()
	ctx := context.Background()

	// set up some data
	_, err := b.Exec(ctx, db,
		"INSERT INTO PERSON(name, age) VALUES(:name1:, :age1:),(:name2:, :age2:),(:name3:, :age3:),(:name4:, :age4:)",
		map[string]interface{}{
			"name1": "Fred", "age1": 20,
			"name2": "Bob", "age2": 50,
			"name3": "Julia", "age3": 32,
			"name4": "Pat", "age4": 37,
		})

	if err != nil {
		panic(err)
	}

	type BadPerson struct {
		Id   int    `prof:"name"`
		Name string `prof:"id"`
		Age  string `prof:"age"`
	}

	var lines []string
	logger.Config(logger.LoggerFunc(func(vals ...interface{}) error {
		lines = append(lines, fmt.Sprintln(vals))
		return nil
	}))

	data := []struct {
		name      string
		ctx       context.Context
		logLevel  logger.Level
		query     string
		params    map[string]interface{}
		out       interface{}
		expected  interface{}
		errMsg    string
		lineCount int
	}{
		{
			name:     "person by id",
			query:    "SELECT * FROM PERSON WHERE id = :id:",
			params:   map[string]interface{}{"id": 1},
			out:      &Person{},
			expected: &Person{Id: 1, Name: "Fred", Age: 20},
		},
		{
			name:   "all people",
			query:  "SELECT * FROM PERSON",
			params: nil,
			out:    &[]Person{},
			expected: &[]Person{
				{Id: 1, Name: "Fred", Age: 20},
				{Id: 2, Name: "Bob", Age: 50},
				{Id: 3, Name: "Julia", Age: 32},
				{Id: 4, Name: "Pat", Age: 37},
			},
		},
		{
			name:   "all people in ages",
			query:  "SELECT * from PERSON WHERE name=:name: and age in (:ages:) and id = :id:",
			params: map[string]interface{}{"id": 1, "ages": []int{20, 32}, "name": "Fred"},
			out:    &[]Person{},
			expected: &[]Person{
				{Id: 1, Name: "Fred", Age: 20},
			},
		},
		{
			name:   "all people in ages unnamed",
			query:  "SELECT * from PERSON WHERE name=:$1: and age in (:$2:) and id = :$3:",
			params: map[string]interface{}{"$1": "Fred", "$2": []int{20, 32}, "$3": 1},
			out:    &[]Person{},
			expected: &[]Person{
				{Id: 1, Name: "Fred", Age: 20},
			},
		},
		// or really non-typed
		{
			name:   "id untyped out",
			query:  "SELECT * FROM PERSON WHERE id = :id:",
			params: map[string]interface{}{"id": 1},
			out:    &map[string]interface{}{},
			expected: &map[string]interface{}{
				"id": int64(1), "name": "Fred", "age": int64(20),
			},
		},
		{
			name:   "slice age untyped out",
			query:  "SELECT * FROM PERSON where age > :$1: and age < :$2:",
			params: map[string]interface{}{"$1": 20, "$2": 50},
			out:    &[]map[string]interface{}{},
			expected: &[]map[string]interface{}{
				{"id": int64(3), "name": "Julia", "age": int64(32)},
				{"id": int64(4), "name": "Pat", "age": int64(37)},
			},
		},
		{
			name:   "id from struct",
			query:  "SELECT * FROM PERSON WHERE id = :p.Id:",
			params: map[string]interface{}{"p": Person{Id: 1}},
			out:    &map[string]interface{}{},
			expected: &map[string]interface{}{
				"id": int64(1), "name": "Fred", "age": int64(20),
			},
		},
		{
			name:   "id from map",
			query:  "SELECT * FROM PERSON WHERE id = :p.id:",
			params: map[string]interface{}{"p": map[string]interface{}{"id": 1}},
			out:    &map[string]interface{}{},
			expected: &map[string]interface{}{
				"id": int64(1), "name": "Fred", "age": int64(20),
			},
		},
		{
			name:     "nothing returned",
			query:    "SELECT * FROM PERSON WHERE id = :id:",
			params:   map[string]interface{}{"id": 100},
			out:      &Person{},
			expected: &Person{},
		},
		{
			name:      "nothing returned logging context",
			ctx:       logger.WithLevel(ctx, logger.DEBUG),
			query:     "SELECT * FROM PERSON WHERE id = :id:",
			params:    map[string]interface{}{"id": 100},
			out:       &Person{},
			expected:  &Person{},
			lineCount: 4,
		},
		{
			name:      "nothing returned logging default",
			logLevel:  logger.DEBUG,
			query:     "SELECT * FROM PERSON WHERE id = :id:",
			params:    map[string]interface{}{"id": 100},
			out:       &Person{},
			expected:  &Person{},
			lineCount: 4,
		},
		// errors
		{
			name:     "not a pointer",
			query:    "SELECT * FROM PERSON WHERE id = :id:",
			params:   map[string]interface{}{"id": 1},
			out:      Person{},
			expected: Person{},
			errMsg:   "not a pointer",
		},
		{
			name:     "no such query",
			query:    "q:nope",
			params:   map[string]interface{}{"id": 1},
			out:      &Person{},
			expected: &Person{},
			errMsg:   "no query found for name nope",
		},
		{
			name:     "bad query",
			query:    "SELECT * FROM PER SON WHERE id = :id:",
			params:   map[string]interface{}{"id": 1},
			out:      &Person{},
			expected: &Person{},
			errMsg:   `pq: relation "per" does not exist`,
		},
		{
			name:     "bad target",
			query:    "SELECT * FROM PERSON WHERE id = :id:",
			params:   map[string]interface{}{"id": 1},
			out:      &map[int]interface{}{},
			expected: &map[int]interface{}{},
			errMsg:   `only maps with string keys are supported`,
		},
		{
			name:     "bad mapping",
			query:    "SELECT * FROM PERSON WHERE id = :id:",
			params:   map[string]interface{}{"id": 1},
			out:      &BadPerson{},
			expected: &BadPerson{},
			errMsg:   "unable to assign value Fred of type string to struct field Id of type int",
		},
		{
			name:     "bad in clause",
			query:    "SELECT * FROM PERSON WHERE id in (:p.id:)",
			params:   map[string]interface{}{"p": map[string]interface{}{"foo": "bar"}},
			out:      &Person{},
			expected: &Person{},
			errMsg:   "cannot extract value; no such map key id",
		},
	}
	for _, v := range data {
		t.Run(v.name, func(t *testing.T) {
			lines = nil
			SetLogLevel(v.logLevel)
			c := ctx
			if v.ctx != nil {
				c = v.ctx
			}
			err := b.Query(c, db, v.query, v.params, v.out)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			if errMsg != v.errMsg {
				t.Errorf("Expected error `%s`, got `%s`", v.errMsg, errMsg)
			}
			if diff := cmp.Diff(v.out, v.expected); diff != "" {
				t.Error(diff)
			}
			if v.lineCount != len(lines) {
				t.Errorf("Expected %d lines of debug, got %d: %#v", v.lineCount, len(lines), lines)
			}
		})
	}
}
