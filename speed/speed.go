package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jonbodner/proteus"
	_ "github.com/mutecomm/go-sqlcipher"
	"github.com/pkg/profile"
)

func SelectProteus(db *sql.DB) {
	var productDao BenchProductDao

	err := proteus.Build(&productDao, proteus.Sqlite)
	if err != nil {
		panic(err)
	}
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()
	start := time.Now()
	pExec := proteus.Wrap(tx)
	for i := 0; i < 1000; i++ {
		for j := 0; j < 10; j++ {
			p, err := productDao.FindById(pExec, j)
			if err != nil {
				panic(err)
			}
			validate(j, p)
		}
	}
	end := time.Now()
	fmt.Println("Proteus", end.Sub(start))
}

func validate(i int, p BenchProduct) {
	if p.Id != i {
		fmt.Errorf("should of had id %d, had %d instead", i, p.Id)
	}
	if p.Name != fmt.Sprintf("person%d", i) {
		fmt.Errorf("should of had person4, had %s instead", p.Name)
	}
	if i%2 == 0 {
		if p.Cost == nil {
			fmt.Errorf("cost should have been non-nil")
		} else {
			if *p.Cost != 1.1*float64(i) {
				fmt.Errorf("should have had %f, had %f instead", 1.1*float64(i), *p.Cost)
			}
		}
	} else {
		if p.Cost != nil {
			fmt.Errorf("should have been nil, was %f", *p.Cost)
		}
	}
}

func SelectNative(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()
	start := time.Now()
	for i := 0; i < 1000; i++ {
		var id int
		var name string
		var cost *float64
		for j := 0; j < 10; j++ {
			rows, err := tx.Query("select id, name, cost from Product where id = ?", j)
			if err != nil {
				panic(err)
			}
			if rows.Next() {
				err = rows.Scan(&id, &name, &cost)
				if err != nil {
					panic(err)
				}
				p := BenchProduct{Id: id, Name: name, Cost: cost}
				validate(j, p)
			}
			rows.Close()
		}
	}
	end := time.Now()
	fmt.Println("Native:", end.Sub(start))
}

type BenchProduct struct {
	Id   int      `prof:"id"`
	Name string   `prof:"name"`
	Cost *float64 `prof:"cost"`
}

type BenchProductDao struct {
	FindById func(e proteus.Querier, id int) (BenchProduct, error)                       `proq:"select id, name, cost from Product where id = :id:" prop:"id"`
	Insert   func(e proteus.Executor, id int, name string, cost *float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

func main() {
	db := setupDbSqlite()
	defer profile.Start().Stop()
	defer db.Close()

	for i := 0; i < 5; i++ {
		fmt.Println("round ", i+1)
		SelectProteus(db)
		SelectNative(db)
		fmt.Println("round ", i+2)
		SelectNative(db)
		SelectProteus(db)
	}
}

func setupDbSqlite() *sql.DB {
	os.Remove("./proteus_test.db")

	db, err := sql.Open("sqlite3", "./proteus_test.db")

	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	drop table if exists product;
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
	}
	populate(db)
	return db
}

func populate(db *sql.DB) {
	productDao := BenchProductDao{}
	proteus.Build(&productDao, proteus.Sqlite)
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	pExec := proteus.Wrap(tx)

	for i := 0; i < 100; i++ {
		var cost *float64
		if i%2 == 0 {
			c := 1.1 * float64(i)
			cost = &c
		}
		rowCount, err := productDao.Insert(pExec, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			log.Fatal(err)
		}
		log.Debug(rowCount)
	}
	tx.Commit()
}
