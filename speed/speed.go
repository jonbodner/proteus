package main

import (
	"database/sql"
	"fmt"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
	"github.com/pkg/profile"
)

func SelectProteus() {
	var productDao BenchProductDao

	err := proteus.Build(&productDao, adapter.Sqlite)
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()
	start := time.Now()
	pExec := adapter.Sql(tx)
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
			if *p.Cost != 1.1 * float64(i) {
				fmt.Errorf("should have had %f, had %f instead", 1.1 * float64(i), *p.Cost)
			}
		}
	} else {
		if p.Cost != nil {
			fmt.Errorf("should have been nil, was %f", *p.Cost)
		}
	}
}

func SelectNative() {
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()
	defer db.Close()
	start := time.Now()
	for i := 0; i < 1000; i++ {
		var id int
		var name string
		var cost *float64
		for j := 0;j< 10;j++ {
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
				validate(j,p)
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
	FindById func(e api.Executor, id int) (BenchProduct, error) `proq:"select id, name, cost from Product where id = :id:" prop:"id"`
}

func main() {
	defer profile.Start().Stop()
	for i := 0;i<5;i++ {
		fmt.Println("round ",i+1)
		SelectProteus()
		SelectNative()
		fmt.Println("round ", i+2)
		SelectNative()
		SelectProteus()
	}
}
