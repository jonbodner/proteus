package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jonbodner/dbtimer"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/logger"
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
	FindByIDMap                   func(e proteus.Querier, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(e proteus.Executor, p map[string]interface{}) (int64, error)                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(e proteus.Querier, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(e proteus.Executor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(e proteus.Querier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(e proteus.Querier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(e proteus.Querier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(e proteus.Querier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(e proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}

type setupDb func(c context.Context, p ProductDAO) *sql.DB

func main() {
	dbtimer.SetTimerLoggerFunc(func(ti dbtimer.TimerInfo) {
		fmt.Printf("%s %s %v %v %d\n", ti.Method, ti.Query, ti.Args, ti.Err, ti.End.Sub(ti.Start).Nanoseconds()/1000)
	})
	logger.Config(logger.LoggerFunc(func(vals ...interface{}) error {
		fmt.Printf("%s: (%s) - %+v\n", vals[1], vals[3], vals[5])
		return nil
	}))

	var productDaoPostgres ProductDAO
	err := proteus.Build(&productDaoPostgres, proteus.Postgres)
	if err != nil {
		panic(err)
	}

	run(setupDbPostgres, productDaoPostgres)
}

func run(setupDb setupDb, productDAO ProductDAO) {
	ctx := logger.WithLevel(context.Background(), logger.DEBUG)
	db := setupDb(ctx, productDAO)
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()

	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDAO.FindByID(tx, 10)))
	cost := new(float64)
	*cost = 56.23
	p := Product{10, "Thingie", cost}
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDAO.Update(tx, p)))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDAO.FindByID(tx, 10)))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDAO.FindByNameAndCost(tx, "fred", 54.10)))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDAO.FindByNameAndCost(tx, "Thingie", 56.23)))

	//using a map of [string]interface{} works too!
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByIDMap(tx, 10))))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByNameAndCostMap(tx, "Thingie", 56.23))))

	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByID(tx, 11))))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.UpdateMap(tx, m))))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByID(tx, 11))))

	//searching using a slice
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByIDSlice(tx, []int{1, 3, 5}))))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByIDSliceAndName(tx, []int{1, 3, 5}, "person1"))))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByIDSliceNameAndCost(tx, []int{1, 3, 5}, "person3", nil))))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByIDSliceCostAndNameSlice(tx, []int{1, 3, 5}, []string{"person3", "person5"}, nil))))

	//using positional parameters instead of names
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln((productDAO.FindByNameAndCostUnlabeled(tx, "Thingie", 56.23))))
}

func setupDbPostgres(c context.Context, productDAO ProductDAO) *sql.DB {
	//db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	db, err := sql.Open("timer", "postgres postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")

	if err != nil {
		logger.Log(c, logger.FATAL, fmt.Sprintln(err))
	}
	sqlStmt := `
	drop table if exists product;
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		logger.Log(c, logger.FATAL, fmt.Sprintf("%q: %s\n", err, sqlStmt))
		return nil
	}
	populate(c, db, productDAO)
	return db
}

func populate(ctx context.Context, db *sql.DB, productDao ProductDAO) {
	tx, err := db.Begin()
	if err != nil {
		logger.Log(ctx, logger.FATAL, fmt.Sprintln(err))
	}
	defer tx.Commit()

	for i := 0; i < 100; i++ {
		var cost *float64
		if i%2 == 0 {
			c := 1.1 * float64(i)
			cost = &c
		}
		rowCount, err := productDao.Insert(tx, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			logger.Log(ctx, logger.FATAL, fmt.Sprintln(err))
		}
		logger.Log(ctx, logger.DEBUG, fmt.Sprintln(rowCount))
	}
}
