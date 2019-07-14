package proteus

import (
	"context"
	"fmt"
	"testing"
)

func TestBuilder_BuildFunction(t *testing.T) {
	b := NewBuilder(Postgres, nil)
	db := setupDbPostgres()
	defer db.Close()
	ctx := context.Background()

	var f func(c context.Context, e ContextExecutor, name string, age int) (int64, error)
	err := b.BuildFunction(ctx, &f, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []string{"name", "age"})
	if err != nil {
		t.Fatalf("build function failed: %v", err)
	}

	var g func(c context.Context, q ContextQuerier, id int) (*Person, error)
	err = b.BuildFunction(ctx, &g, "SELECT * FROM PERSON WHERE id = :id:", []string{"id"})
	if err != nil {
		t.Fatalf("build function 2 failed: %v", err)
	}

	rows, err := f(ctx, db, "Fred", 20)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	fmt.Println(rows)

	p, err := g(ctx, db, 1)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	fmt.Println(p)
}

func TestBuilder_ExecuteQuery(t *testing.T) {
	b := NewBuilder(Postgres, nil)
	db := setupDbPostgres()
	defer db.Close()
	ctx := context.Background()
	rows, err := b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []interface{}{"Fred", 20}, []string{"name", "age"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	fmt.Println(rows)
	rows, err = b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []interface{}{"Bob", 50}, []string{"name", "age"})
	if err != nil {
		t.Fatalf("create 2 failed: %v", err)
	}
	fmt.Println(rows)
	rows, err = b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []interface{}{"Julia", 32}, []string{"name", "age"})
	if err != nil {
		t.Fatalf("create 3 failed: %v", err)
	}
	fmt.Println(rows)
	rows, err = b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []interface{}{"Pat", 37}, []string{"name", "age"})
	if err != nil {
		t.Fatalf("create 4 failed: %v", err)
	}
	fmt.Println(rows)

	var p Person
	err = b.Query(ctx, db, "SELECT * FROM PERSON WHERE id = :id:", []interface{}{1}, []string{"id"}, &p)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	fmt.Println(p)

	var people []Person
	err = b.Query(ctx, db, "SELECT * FROM PERSON", nil, nil, &people)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}
	fmt.Println(people)

	err = b.Query(ctx, db, "SELECT * from PERSON WHERE name=:name: and age in (:ages:) and id = :id:", []interface{}{1, []int{20, 32}, "Fred"}, []string{"id", "ages", "name"}, &people)
	if err != nil {
		t.Fatalf("get by age failed: %v", err)
	}
	fmt.Println(people)

	err = b.Query(ctx, db, "SELECT * from PERSON WHERE name=:$1: and age in (:$2:) and id = :$3:", []interface{}{"Fred", []int{20, 32}, 1}, nil, &people)
	if err != nil {
		t.Fatalf("get by age 2 failed: %v", err)
	}
	fmt.Println(people)
}

//func doPersonStuffForProteusTest(b *testing.B, wrapper Wrapper) (int64, *Person, []Person, error) {
//
//	count, err = personDao.Update(wrapper, 1, "Freddie", 30)
//	if err != nil {
//		b.Fatalf("update failed: %v", err)
//	}
//
//	person, err = personDao.Get(wrapper, 1)
//	if err != nil {
//		b.Fatalf("get 2 failed: %v", err)
//	}
//
//	count, err = personDao.Delete(wrapper, 1)
//	if err != nil {
//		b.Fatalf("delete failed: %v", err)
//	}
//
//	count, err = personDao.Delete(wrapper, 1)
//	if err != nil {
//		b.Fatalf("delete 2 failed: %v", err)
//	}
//
//	person, err = personDao.Get(wrapper, 1)
//	if err != nil {
//		b.Fatalf("get 3 failed: %v", err)
//	}
//
//	return count, person, people, err
//}
//
