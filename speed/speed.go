package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jonbodner/proteus"
	_ "github.com/lib/pq"
	"github.com/pkg/profile"
	log "github.com/sirupsen/logrus"
)

func SelectProteus(ctx context.Context, db *sql.DB) time.Duration {
	var productDao BenchProductDao

	err := proteus.Build(&productDao, proteus.Postgres)
	if err != nil {
		panic(err)
	}
	start := time.Now()
	for i := 0; i < 1000; i++ {
		for j := 0; j < 10; j++ {
			p, err := productDao.FindById(ctx, db, j)
			if err != nil {
				panic(err)
			}
			validate(j, p)
		}
	}
	end := time.Now()
	return end.Sub(start)
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

func SelectNative(ctx context.Context, db *sql.DB) time.Duration {
	start := time.Now()
	for i := 0; i < 1000; i++ {
		var id int
		var name string
		var cost *float64
		for j := 0; j < 10; j++ {
			rows, err := db.QueryContext(ctx, "select id, name, cost from Product where id = $1", j)
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
	return end.Sub(start)
}

type BenchProduct struct {
	Id   int      `prof:"id"`
	Name string   `prof:"name"`
	Cost *float64 `prof:"cost"`
}

type BenchProductDao struct {
	FindById func(ctx context.Context, e proteus.ContextQuerier, id int) (BenchProduct, error)                       `proq:"select id, name, cost from Product where id = :id:" prop:"id"`
	Insert   func(ctx context.Context, e proteus.ContextExecutor, id int, name string, cost *float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

func main() {
	ctx := context.Background()
	db := setupDbPostgres(ctx)
	defer profile.Start().Stop()
	defer db.Close()

	for i := 0; i < 5; i++ {
		fmt.Printf("Round %d - Proteus first\n", i+1)
		fmt.Println("Proteus: ", SelectProteus(ctx, db))
		fmt.Println("Native: ", SelectNative(ctx, db))
		fmt.Printf("Round %d - Native first\n", i+1)
		fmt.Println("Native: ", SelectNative(ctx, db))
		fmt.Println("Proteus: ", SelectProteus(ctx, db))
	}
}

func setupDbPostgres(ctx context.Context) *sql.DB {

	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")

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
	populate(ctx, db)
	return db
}

func populate(ctx context.Context, db *sql.DB) {
	productDao := BenchProductDao{}
	proteus.Build(&productDao, proteus.Postgres)
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Commit()

	for i := 0; i < 100; i++ {
		var cost *float64
		if i%2 == 0 {
			c := 1.1 * float64(i)
			cost = &c
		}
		rowCount, err := productDao.Insert(ctx, tx, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			log.Fatal(err)
		}
		log.Debug(rowCount)
	}
}
