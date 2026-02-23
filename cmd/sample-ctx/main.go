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
	FindByID                      func(ctx context.Context, e proteus.ContextQuerier, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(ctx context.Context, e proteus.ContextExecutor, p Product) (int64, error)                                   `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(ctx context.Context, e proteus.ContextQuerier, id int) (map[string]any, error)                              `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(ctx context.Context, e proteus.ContextExecutor, p map[string]any) (int64, error)                            `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]map[string]any, error)         `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(ctx context.Context, e proteus.ContextExecutor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(ctx context.Context, e proteus.ContextQuerier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(ctx context.Context, e proteus.ContextQuerier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(ctx context.Context, e proteus.ContextQuerier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(ctx context.Context, e proteus.ContextQuerier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}

type setupDb func(ctx context.Context, p ProductDAO) *sql.DB

func main() {
	dbtimer.SetTimerLoggerFunc(func(ti dbtimer.TimerInfo) {
		fmt.Printf("%s %s %v %v %d\n", ti.Method, ti.Query, ti.Args, ti.Err, ti.End.Sub(ti.Start).Nanoseconds()/1000)
	})

	ctx := context.Background()
	var productDAO ProductDAO
	err := proteus.ShouldBuild(ctx, &productDAO, proteus.Postgres)
	if err != nil {
		panic(err)
	}

	run(setupDbPostgres, productDAO)
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

	product, err := productDAO.FindByID(ctx, tx, 10)
	slog.DebugContext(ctx, "FindByID", "product", product, "err", err)

	cost := new(float64)
	*cost = 56.23
	p := Product{10, "Thingie", cost}
	rowsAffected, err := productDAO.Update(ctx, tx, p)
	slog.DebugContext(ctx, "Update", "rowsAffected", rowsAffected, "err", err)

	product, err = productDAO.FindByID(ctx, tx, 10)
	slog.DebugContext(ctx, "FindByID after update", "product", product, "err", err)

	products, err := productDAO.FindByNameAndCost(ctx, tx, "fred", 54.10)
	slog.DebugContext(ctx, "FindByNameAndCost fred", "products", products, "err", err)

	products, err = productDAO.FindByNameAndCost(ctx, tx, "Thingie", 56.23)
	slog.DebugContext(ctx, "FindByNameAndCost Thingie", "products", products, "err", err)

	//using a map of [string]any works too!
	m2, err := productDAO.FindByIDMap(ctx, tx, 10)
	slog.DebugContext(ctx, "FindByIDMap", "result", m2, "err", err)

	maps, err := productDAO.FindByNameAndCostMap(ctx, tx, "Thingie", 56.23)
	slog.DebugContext(ctx, "FindByNameAndCostMap", "results", maps, "err", err)

	product, err = productDAO.FindByID(ctx, tx, 11)
	slog.DebugContext(ctx, "FindByID 11", "product", product, "err", err)

	m := map[string]any{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	rowsAffected, err = productDAO.UpdateMap(ctx, tx, m)
	slog.DebugContext(ctx, "UpdateMap", "rowsAffected", rowsAffected, "err", err)

	product, err = productDAO.FindByID(ctx, tx, 11)
	slog.DebugContext(ctx, "FindByID after UpdateMap", "product", product, "err", err)

	//searching using a slice
	products, err = productDAO.FindByIDSlice(ctx, tx, []int{1, 3, 5})
	slog.DebugContext(ctx, "FindByIDSlice", "products", products, "err", err)

	products, err = productDAO.FindByIDSliceAndName(ctx, tx, []int{1, 3, 5}, "person1")
	slog.DebugContext(ctx, "FindByIDSliceAndName", "products", products, "err", err)

	products, err = productDAO.FindByIDSliceNameAndCost(ctx, tx, []int{1, 3, 5}, "person3", nil)
	slog.DebugContext(ctx, "FindByIDSliceNameAndCost", "products", products, "err", err)

	products, err = productDAO.FindByIDSliceCostAndNameSlice(ctx, tx, []int{1, 3, 5}, []string{"person3", "person5"}, nil)
	slog.DebugContext(ctx, "FindByIDSliceCostAndNameSlice", "products", products, "err", err)

	//using positional parameters instead of names
	products, err = productDAO.FindByNameAndCostUnlabeled(ctx, tx, "Thingie", 56.23)
	slog.DebugContext(ctx, "FindByNameAndCostUnlabeled", "products", products, "err", err)

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
		rowCount, err := productDao.Insert(ctx, tx, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			slog.ErrorContext(ctx, "insert failed", "err", err)
		}
		slog.DebugContext(ctx, "inserted row", "rowCount", rowCount)
	}

	if err := tx.Commit(); err != nil {
		slog.ErrorContext(ctx, "commit failed", "err", err)
	}
}
