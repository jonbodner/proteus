package proteus

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

func BenchmarkProteus(b *testing.B) {
	b.StopTimer()
	err := Build(&personDao, Postgres)
	if err != nil {
		panic(err)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := setupDbPostgres()
		wrapper := Wrap(db)
		b.StartTimer()
		doPersonStuffForProteusTest(b, wrapper)
		b.StopTimer()
		db.Close()
	}
}

type Person struct {
	Id   int    `prof:"id"`
	Name string `prof:"name"`
	Age  int    `prof:"age"`
}

func (p Person) String() string {
	return fmt.Sprintf("Id: %d\tName:%s\tAge:%d", p.Id, p.Name, p.Age)
}

type PersonDao struct {
	Create   func(e Executor, name string, age int) (int64, error)              `proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
	Get      func(q Querier, id int) (*Person, error)                           `proq:"SELECT * FROM PERSON WHERE id = :id:" prop:"id"`
	Update   func(e Executor, id int, name string, age int) (int64, error)      `proq:"UPDATE PERSON SET name = :name:, age=:age: where id=:id:" prop:"id,name,age"`
	Delete   func(e Executor, id int) (int64, error)                            `proq:"DELETE FROM PERSON WHERE id = :id:" prop:"id"`
	GetAll   func(q Querier) ([]Person, error)                                  `proq:"SELECT * FROM PERSON"`
	GetByAge func(q Querier, id int, ages []int, name string) ([]Person, error) `proq:"SELECT * from PERSON WHERE name=:name: and age in (:ages:) and id = :id:" prop:"id,ages,name"`
}

var personDao PersonDao

func setupDbPostgres() *sql.DB {
	db, err := sql.Open("postgres", "postgres://jon:jon@localhost/jon?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	drop table if exists person;
	create table person (id serial primary key, name text, age int);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
	}
	return db
}

func doPersonStuffForProteusTest(b *testing.B, wrapper Wrapper) (int64, *Person, []Person, error) {
	count, err := personDao.Create(wrapper, "Fred", 20)
	if err != nil {
		b.Fatalf("create failed: %v", err)
	}

	count, err = personDao.Create(wrapper, "Bob", 50)
	if err != nil {
		b.Fatalf("create 2 failed: %v", err)
	}

	count, err = personDao.Create(wrapper, "Julia", 32)
	if err != nil {
		b.Fatalf("create 3 failed: %v", err)
	}

	count, err = personDao.Create(wrapper, "Pat", 37)
	if err != nil {
		b.Fatalf("create 4 failed: %v", err)
	}

	person, err := personDao.Get(wrapper, 1)
	if err != nil {
		b.Fatalf("get failed: %v", err)
	}

	people, err := personDao.GetAll(wrapper)
	if err != nil {
		b.Fatalf("get all failed: %v", err)
	}

	people, err = personDao.GetByAge(wrapper, 1, []int{20, 32}, "Fred")
	if err != nil {
		b.Fatalf("get by age failed: %v", err)
	}

	count, err = personDao.Update(wrapper, 1, "Freddie", 30)
	if err != nil {
		b.Fatalf("update failed: %v", err)
	}

	person, err = personDao.Get(wrapper, 1)
	if err != nil {
		b.Fatalf("get 2 failed: %v", err)
	}

	count, err = personDao.Delete(wrapper, 1)
	if err != nil {
		b.Fatalf("delete failed: %v", err)
	}

	count, err = personDao.Delete(wrapper, 1)
	if err != nil {
		b.Fatalf("delete 2 failed: %v", err)
	}

	person, err = personDao.Get(wrapper, 1)
	if err != nil {
		b.Fatalf("get 3 failed: %v", err)
	}

	return count, person, people, err
}

func BenchmarkStandard(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		db := setupDbPostgres()
		b.StartTimer()
		doPersonStuffForStandardTest(b, db)
		b.StopTimer()
		db.Close()
	}
}

type standardPersonDao struct{}

//	`proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
func (spd standardPersonDao) Create(db *sql.DB, name string, age int) (int64, error) {
	result, err := db.Exec("INSERT INTO PERSON(name, age) VALUES($1, $2)", name, age)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return count, err
}

//`proq:"SELECT * FROM PERSON WHERE id = :id:" prop:"id"`
func (spd standardPersonDao) Get(db *sql.DB, id int) (*Person, error) {
	rows, err := db.Query("SELECT age, name, id FROM PERSON WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	var p *Person
	if rows.Next() {
		err = rows.Err()
		if err == nil {
			p = &Person{}
			err = rows.Scan(&p.Age, &p.Name, &p.Id)
		}
	}
	rows.Close()
	return p, err
}

//`proq:"UPDATE PERSON SET name = :name:, age=:age: where id=:id:" prop:"id,name,age"`
func (spd standardPersonDao) Update(db *sql.DB, id int, name string, age int) (int64, error) {
	result, err := db.Exec("UPDATE PERSON SET name = $1, age=$2 where id=$3", name, age, id)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return count, err
}

//`proq:"DELETE FROM PERSON WHERE id = :id:" prop:"id"`
func (spd standardPersonDao) Delete(db *sql.DB, id int) (int64, error) {
	result, err := db.Exec("DELETE FROM PERSON WHERE id = $1", id)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return count, err
}

//`proq:"SELECT * FROM PERSON"`
func (spd standardPersonDao) GetAll(db *sql.DB) ([]Person, error) {
	rows, err := db.Query("SELECT age, name, id FROM PERSON")
	if err != nil {
		return nil, err
	}
	var out []Person
	for rows.Next() {
		err = rows.Err()
		if err != nil {
			break
		}
		p := Person{}
		err = rows.Scan(&p.Age, &p.Name, &p.Id)
		if err != nil {
			break
		}
		out = append(out, p)
	}
	rows.Close()
	return out, err
}

//`proq:"SELECT * from PERSON WHERE name=:name: and age in (:ages:) and id = :id:" prop:"id,ages,name"`
func (spd standardPersonDao) GetByAge(db *sql.DB, id int, ages []int, name string) ([]Person, error) {
	startQuery := "SELECT age, name, id from PERSON WHERE name=$1 and age in (:ages:) and id = :id:"
	params := make([]string, len(ages))
	for i := 0; i < len(ages); i++ {
		params[i] = fmt.Sprintf("$%d", i+2)
	}

	inClause := strings.Join(params, ",")

	finalQuery := strings.Replace(startQuery, ":ages:", inClause, -1)
	finalQuery = strings.Replace(finalQuery, ":id:", fmt.Sprintf("$%d", len(ages)+2), -1)

	args := make([]interface{}, 0, len(ages)+2)
	args = append(args, name)
	for _, v := range ages {
		args = append(args, v)
	}
	args = append(args, id)

	rows, err := db.Query(finalQuery, args...)
	if err != nil {
		return nil, err
	}
	var out []Person
	for rows.Next() {
		err = rows.Err()
		if err != nil {
			break
		}
		p := Person{}
		err = rows.Scan(&p.Age, &p.Name, &p.Id)
		if err != nil {
			break
		}
		out = append(out, p)
	}
	rows.Close()
	return out, err
}

var sPersonDao = standardPersonDao{}

func doPersonStuffForStandardTest(b *testing.B, db *sql.DB) (int64, *Person, []Person, error) {
	count, err := sPersonDao.Create(db, "Fred", 20)
	if err != nil {
		b.Fatalf("create failed: %v", err)
	}

	count, err = sPersonDao.Create(db, "Bob", 50)
	if err != nil {
		b.Fatalf("create 2 failed: %v", err)
	}

	count, err = sPersonDao.Create(db, "Julia", 32)
	if err != nil {
		b.Fatalf("create 3 failed: %v", err)
	}

	count, err = sPersonDao.Create(db, "Pat", 37)
	if err != nil {
		b.Fatalf("create 4 failed: %v", err)
	}

	person, err := sPersonDao.Get(db, 1)
	if err != nil {
		b.Fatalf("get failed: %v", err)
	}

	people, err := sPersonDao.GetAll(db)
	if err != nil {
		b.Fatalf("get all failed: %v", err)
	}

	people, err = sPersonDao.GetByAge(db, 1, []int{20, 32}, "Fred")
	if err != nil {
		b.Fatalf("get by age failed: %v", err)
	}

	count, err = sPersonDao.Update(db, 1, "Freddie", 30)
	if err != nil {
		b.Fatalf("update failed: %v", err)
	}

	person, err = sPersonDao.Get(db, 1)
	if err != nil {
		b.Fatalf("get 2 failed: %v", err)
	}

	count, err = sPersonDao.Delete(db, 1)
	if err != nil {
		b.Fatalf("delete failed: %v", err)
	}

	count, err = sPersonDao.Delete(db, 1)
	if err != nil {
		b.Fatalf("delete 2 failed: %v", err)
	}

	person, err = sPersonDao.Get(db, 1)
	if err != nil {
		b.Fatalf("get 3 failed: %v", err)
	}

	return count, person, people, err
}
