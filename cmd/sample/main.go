package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/jonbodner/dbtimer"
	"github.com/jonbodner/proteus"
	_ "github.com/lib/pq"
)

type Product struct {
	Id   int      `prof:"id"`
	Name string   `prof:"name"`
	Cost *float64 `prof:"cost"`
}

func (p Product) String() string {
	c := "<nil>"
	if p.Cost != nil {
		c = fmt.Sprintf("%f", *p.Cost)
	}
	return fmt.Sprintf("%d: %s(%s)", p.Id, p.Name, c)
}

type ProductDAO struct {
	FindByID                      func(e proteus.Querier, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(e proteus.Executor, p Product) (int64, error)                                   `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(e proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(e proteus.Querier, id int) (map[string]any, error)                              `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(e proteus.Executor, p map[string]any) (int64, error)                            `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(e proteus.Querier, name string, cost float64) ([]map[string]any, error)         `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(e proteus.Executor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(e proteus.Querier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(e proteus.Querier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(e proteus.Querier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(e proteus.Querier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(e proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}

type setupDb func(ctx context.Context, p ProductDAO) *sql.DB

func main() {
	dbtimer.SetTimerLoggerFunc(func(ti dbtimer.TimerInfo) {
		fmt.Printf("%s %s %v %v %d\n", ti.Method, ti.Query, ti.Args, ti.Err, ti.End.Sub(ti.Start).Nanoseconds()/1000)
	})

	var productDaoPostgres ProductDAO
	err := proteus.Build(&productDaoPostgres, proteus.Postgres)
	if err != nil {
		panic(err)
	}

	run(setupDbPostgres, productDaoPostgres)
}

func run(setupDb setupDb, productDAO ProductDAO) {
	ctx := context.Background()
	db := setupDb(ctx, productDAO)
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer func() { _ = tx.Rollback() }()

	product, err := productDAO.FindByID(tx, 10)
	slog.DebugContext(ctx, "FindByID", "result", product, "error", err)

	cost := new(float64)
	*cost = 56.23
	p := Product{10, "Thingie", cost}
	rowsAffected, err := productDAO.Update(tx, p)
	slog.DebugContext(ctx, "Update", "result", rowsAffected, "error", err)

	product, err = productDAO.FindByID(tx, 10)
	slog.DebugContext(ctx, "FindByID after update", "result", product, "error", err)

	results, err := productDAO.FindByNameAndCost(tx, "fred", 54.10)
	slog.DebugContext(ctx, "FindByNameAndCost fred", "result", results, "error", err)

	results, err = productDAO.FindByNameAndCost(tx, "Thingie", 56.23)
	slog.DebugContext(ctx, "FindByNameAndCost Thingie", "result", results, "error", err)

	//using a map of [string]any works too!
	mapResult, err := productDAO.FindByIDMap(tx, 10)
	slog.DebugContext(ctx, "FindByIDMap", "result", mapResult, "error", err)

	mapResults, err := productDAO.FindByNameAndCostMap(tx, "Thingie", 56.23)
	slog.DebugContext(ctx, "FindByNameAndCostMap", "result", mapResults, "error", err)

	product, err = productDAO.FindByID(tx, 11)
	slog.DebugContext(ctx, "FindByID 11", "result", product, "error", err)

	m := map[string]any{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	rowsAffected, err = productDAO.UpdateMap(tx, m)
	slog.DebugContext(ctx, "UpdateMap", "result", rowsAffected, "error", err)

	product, err = productDAO.FindByID(tx, 11)
	slog.DebugContext(ctx, "FindByID after UpdateMap", "result", product, "error", err)

	//searching using a slice
	results, err = productDAO.FindByIDSlice(tx, []int{1, 3, 5})
	slog.DebugContext(ctx, "FindByIDSlice", "result", results, "error", err)

	results, err = productDAO.FindByIDSliceAndName(tx, []int{1, 3, 5}, "person1")
	slog.DebugContext(ctx, "FindByIDSliceAndName", "result", results, "error", err)

	results, err = productDAO.FindByIDSliceNameAndCost(tx, []int{1, 3, 5}, "person3", nil)
	slog.DebugContext(ctx, "FindByIDSliceNameAndCost", "result", results, "error", err)

	results, err = productDAO.FindByIDSliceCostAndNameSlice(tx, []int{1, 3, 5}, []string{"person3", "person5"}, nil)
	slog.DebugContext(ctx, "FindByIDSliceCostAndNameSlice", "result", results, "error", err)

	//using positional parameters instead of names
	results, err = productDAO.FindByNameAndCostUnlabeled(tx, "Thingie", 56.23)
	slog.DebugContext(ctx, "FindByNameAndCostUnlabeled", "result", results, "error", err)

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "commit failed", "err", err)
	}
}

func setupDbPostgres(ctx context.Context, productDAO ProductDAO) *sql.DB {
	//db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	db, err := sql.Open("timer", "postgres postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")

	if err != nil {
		slog.ErrorContext(ctx, "db open failed", "err", err)
	}
	sqlStmt := `
	drop table if exists product;
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		slog.ErrorContext(ctx, "exec failed", "err", err, "statement", sqlStmt)
		return nil
	}
	populate(ctx, db, productDAO)
	return db
}

func populate(ctx context.Context, db *sql.DB, productDao ProductDAO) {
	tx, err := db.Begin()
	if err != nil {
		slog.ErrorContext(ctx, "begin tx failed", "err", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	for i := 0; i < 100; i++ {
		var cost *float64
		if i%2 == 0 {
			c := 1.1 * float64(i)
			cost = &c
		}
		rowCount, err := productDao.Insert(tx, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			slog.ErrorContext(ctx, "insert failed", "err", err)
		}
		slog.DebugContext(ctx, "inserted row", "rowCount", rowCount)
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "commit failed", "err", err)
	}
}
