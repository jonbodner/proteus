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
	rows, err := b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", map[string]interface{}{"name": "Fred", "age": 20})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	fmt.Println(rows)
	rows, err = b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", map[string]interface{}{"name": "Bob", "age": 50})
	if err != nil {
		t.Fatalf("create 2 failed: %v", err)
	}
	fmt.Println(rows)
	rows, err = b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", map[string]interface{}{"name": "Julia", "age": 32})
	if err != nil {
		t.Fatalf("create 3 failed: %v", err)
	}
	fmt.Println(rows)
	rows, err = b.Execute(ctx, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", map[string]interface{}{"name": "Pat", "age": 37})
	if err != nil {
		t.Fatalf("create 4 failed: %v", err)
	}
	fmt.Println(rows)

	var p Person
	err = b.Query(ctx, db, "SELECT * FROM PERSON WHERE id = :id:", map[string]interface{}{"id": 1}, &p)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	fmt.Println(p)

	var people []Person
	err = b.Query(ctx, db, "SELECT * FROM PERSON", nil, &people)
	if err != nil {
		t.Fatalf("get all failed: %v", err)
	}
	fmt.Println(people)

	err = b.Query(ctx, db, "SELECT * from PERSON WHERE name=:name: and age in (:ages:) and id = :id:", map[string]interface{}{"id": 1, "ages": []int{20, 32}, "name": "Fred"}, &people)
	if err != nil {
		t.Fatalf("get by age failed: %v", err)
	}
	fmt.Println(people)

	err = b.Query(ctx, db, "SELECT * from PERSON WHERE name=:$1: and age in (:$2:) and id = :$3:", map[string]interface{}{"$1": "Fred", "$2": []int{20, 32}, "$3": 1}, &people)
	if err != nil {
		t.Fatalf("get by age 2 failed: %v", err)
	}
	fmt.Println(people)

	// or really non-typed
	var p2 map[string]interface{}
	err = b.Query(ctx, db, "SELECT * FROM PERSON WHERE id = :id:", map[string]interface{}{"id": 1}, &p2)
	if err != nil {
		t.Fatalf("get map failed: %v", err)
	}
	fmt.Println(p2)

	var people2 []map[string]interface{}
	err = b.Query(ctx, db, "SELECT * FROM PERSON where age > :$1: and age < :$2:", map[string]interface{}{"$1": 20, "$2": 50}, &people2)
	if err != nil {
		t.Fatalf("get map slice failed: %v", err)
	}
	fmt.Println(people2)

	var p3 map[string]interface{}
	err = b.Query(ctx, db, "SELECT * FROM PERSON WHERE id = :p.Id:", map[string]interface{}{"p": p}, &p3)
	if err != nil {
		t.Fatalf("get map 2 failed: %v", err)
	}
	fmt.Println(p3)

	var p4 map[string]interface{}
	err = b.Query(ctx, db, "SELECT * FROM PERSON WHERE id = :p.id:", map[string]interface{}{"p": p3}, &p4)
	if err != nil {
		t.Fatalf("get map 2 failed: %v", err)
	}
	fmt.Println(p4)
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
