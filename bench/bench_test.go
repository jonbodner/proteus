package proteus

import (
	"testing"
	"database/sql"
	"log"
	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	_ "github.com/mattn/go-sqlite3"
)

func BenchmarkSelectProteus(b *testing.B) {
	var productDao BenchProductDao

	err := Build(&productDao, adapter.Sqlite)
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	b.ResetTimer()
	pExec := adapter.Sql(db)
	for i := 0;i<b.N;i++ {
		p, err := productDao.FindById(pExec,4)
		if err != nil {
			panic(err)
		}
		if p.Id != 4 {
			b.Errorf("should of had id 4, had %d instead", p.Id)
		}
		if p.Name != "person4" {
			b.Errorf("should of had person4, had %s instead", p.Name)
		}
		if p.Cost == nil {
			b.Errorf("cost should have been non-nil")
		} else {
			if *p.Cost != 4.4 {
				b.Errorf("should have had 4.4, had %f instead", p.Cost)
			}
		}

	}
}

func BenchmarkSelectNative(b *testing.B) {
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	b.ResetTimer()
	for i := 0;i<b.N;i++ {
		var id int
		var name string
		cost := new(float64)
		rows, err := db.Query("select id, name, cost from Product where id = ?", 4)
		if err != nil {
			panic(err)
		}
		if rows.Next() {
			err = rows.Scan(&id, &name, cost)
			if err != nil {
				panic(err)
			}
			_ = BenchProduct{Id: id, Name: name, Cost: cost}
		}
	}
}

type BenchProduct struct {
	Id   int     `prof:"id"`
	Name string  `prof:"name"`
	Cost *float64 `prof:"cost"`
}

type BenchProductDao struct {
	FindById             func(e api.Executor, id int) (BenchProduct, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Insert               func(e api.Executor, id int, name string, cost *float64) (int64, error)            `proe:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

/*
func setupDb() *sql.DB {
	os.Remove("./proteus_test.db")

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	pExec := adapter.Sql(tx)

	for i := 0; i < 10; i++ {
		var cost *float64
		if i % 2 == 0 {
			c := 1.1*float64(i)
			cost = &c
		}
		_, err = productDao.Insert(pExec, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()
	return db
}

func TestMain(m *testing.M) {
	flag.Parse()

	os.Exit(m.Run())
}
*/