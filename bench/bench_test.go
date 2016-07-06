package proteus

import (
	"database/sql"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"testing"
)

func BenchmarkSelectProteus(b *testing.B) {
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
	b.ResetTimer()
	pExec := adapter.Sql(tx)
	for i := 0; i < b.N; i++ {
		p, err := productDao.FindById(pExec, 4)
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
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()
	defer db.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id int
		var name string
		cost := new(float64)
		rows, err := tx.Query("select id, name, cost from Product where id = ?", 4)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		if rows.Next() {
			err = rows.Scan(&id, &name, cost)
			if err != nil {
				panic(err)
			}
			p := BenchProduct{Id: id, Name: name, Cost: cost}
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
}

type BenchProduct struct {
	Id   int      `prof:"id"`
	Name string   `prof:"name"`
	Cost *float64 `prof:"cost"`
}

type BenchProductDao struct {
	FindById func(e api.Executor, id int) (BenchProduct, error) `proq:"select id, name, cost from Product where id = :id:" prop:"id"`
}
