package proteus

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math"
	"testing"

	"github.com/jonbodner/proteus"
	_ "github.com/lib/pq"
)

func setupDbPostgres(ctx context.Context, db *sql.DB) {
	sqlStmt := `
	drop table if exists product;
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err := db.ExecContext(ctx, sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
	populate(ctx, db)
}

func populate(ctx context.Context, db *sql.DB) {
	productDao := BenchProductDao{}
	proteus.Build(&productDao, proteus.Postgres)
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()

	for i := 0; i < 10; i++ {
		var cost *float64
		if i%2 == 0 {
			c := 1.1 * float64(i)
			cost = &c
		}
		_, err := productDao.Insert(ctx, tx, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			panic(err)
		}
	}
}

type closer func()

func setup() (*sql.Tx, closer) {
	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	setupDbPostgres(ctx, db)
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	return tx, func() {
		tx.Commit()
		db.Close()
	}
}

func BenchmarkSelectProteus(b *testing.B) {
	var productDao BenchProductDao
	ctx := context.Background()

	err := proteus.ShouldBuild(ctx, &productDao, proteus.Postgres)
	if err != nil {
		panic(err)
	}

	tx, closer := setup()
	defer closer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, err := productDao.FindByID(ctx, tx, 4)
		if err != nil {
			panic(err)
		}
		validate(p, b)
	}
}

func validate(p BenchProduct, b *testing.B) {
	if p.ID != 4 {
		b.Errorf("should of had id 4, had %d instead", p.ID)
	}
	if p.Name != "person4" {
		b.Errorf("should of had person4, had %s instead", p.Name)
	}
	if p.Cost == nil {
		b.Errorf("cost should have been non-nil")
	} else {
		if math.Abs(*p.Cost-4.4) > 0.001 {
			b.Errorf("should have had 4.4, had %f instead", *p.Cost)
		}
	}
}

func BenchmarkSelectNative(b *testing.B) {
	ctx := context.Background()
	tx, closer := setup()
	defer closer()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var id int
		var name string
		cost := new(float64)
		rows, err := tx.QueryContext(ctx, "select id, name, cost from Product where id = $1", 4)
		if err != nil {
			panic(err)
		}
		if rows.Next() {
			err = rows.Scan(&id, &name, cost)
			if err != nil {
				panic(err)
			}
			p := BenchProduct{ID: id, Name: name, Cost: cost}
			validate(p, b)
		}
		rows.Close()
	}
}

type BenchProduct struct {
	ID   int      `prof:"id"`
	Name string   `prof:"name"`
	Cost *float64 `prof:"cost"`
}

type BenchProductDao struct {
	FindByID func(ctx context.Context, e proteus.ContextQuerier, id int) (BenchProduct, error)                       `proq:"select id, name, cost from Product where id = :id:" prop:"id"`
	Insert   func(ctx context.Context, e proteus.ContextExecutor, id int, name string, cost *float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}
