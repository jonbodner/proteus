// +build postgres

package proteus_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jonbodner/proteus"
	_ "github.com/lib/pq"
)

func TestBatching(t *testing.T) {
	create := `create table if not exists test (
  id serial primary key,
  name varchar(100),
  age int
)`
	type Person struct {
		ID   int    `prof:"id"`
		Name string `prof:"name"`
		Age  int    `prof:"age"`
	}

	type MyCommands struct {
		AddPeople  func(e proteus.Executor, names []string, ages []int) (int64, error) `proq:"insert into test (name, age) values {:names:, :ages:}" prop:"names,ages"`
		AddPeople2 func(e proteus.Executor, people []Person) (int64, error)            `proq:"insert into test (name, age) values {:people.name:, :people.age:}" prop:"people"`
		AddPerson  func(e proteus.Executor, name string, age int) (int64, error)       `proq:"insert into test (name, age) values (:name:, :age:)" prop:"name,age"`
		GetPerson  func(q proteus.Querier, name string) (Person, error)                `proq:"select id, name, age from test where name = :name:" prop:"name"`
	}

	mc := MyCommands{}
	err := proteus.Build(&mc, proteus.Postgres)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("postgres", "postgres://postgres@localhost/postgres?sslmode=disable")

	if err != nil {
		panic(err)
	}
	_, err = db.Exec(create)
	if err != nil {
		panic(err)
	}

	e := proteus.Wrap(db)
	//build data
	var names []string
	var ages []int
	for i := 0; i < 100; i++ {
		names = append(names, fmt.Sprintf("name%d", i))
		ages = append(ages, i)
	}
	rows, err := mc.AddPeople(e, names, ages)

	if err != nil {
		panic(err)
	}
	if rows != 100 {
		t.Error("Should have added 100 rows, added ", rows)
	}
}
