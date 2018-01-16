package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/jonbodner/dbtimer"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/logger"
	_ "github.com/lib/pq"
	_ "github.com/mutecomm/go-sqlcipher"
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

type ProductDao struct {
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

var productDaoSqlite = ProductDao{}
var productDaoPostgres = ProductDao{}

func init() {
	dbtimer.SetTimerLoggerFunc(func(ti dbtimer.TimerInfo) {
		fmt.Printf("%s %s %v %v %d\n", ti.Method, ti.Query, ti.Args, ti.Err, ti.End.Sub(ti.Start).Nanoseconds()/1000)
	})
	logger.Config(logger.FormatterFunc(func(vals ...interface{}) {
		fmt.Printf("%s: (%s) - %s\n", vals[1], vals[3], vals[5])
	}))
	err := proteus.Build(&productDaoPostgres, proteus.Postgres)
	if err != nil {
		panic(err)
	}

	err = proteus.Build(&productDaoSqlite, proteus.Sqlite)
	if err != nil {
		panic(err)
	}
}

type setupDb func(c context.Context) *sql.DB

func main() {
	run(setupDbSqlite, productDaoSqlite)
	run(setupDbPostgres, productDaoPostgres)
}

func run(setupDb setupDb, productDao ProductDao) {
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	db := setupDb(c)
	defer db.Close()
	exec, err := db.Begin()
	if err != nil {
		panic(err)
	}

	pExec := proteus.Wrap(exec)

	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByID(pExec, 10)))
	cost := new(float64)
	*cost = 56.23
	p := Product{10, "Thingie", cost}
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.Update(pExec, p)))
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByID(pExec, 10)))
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByNameAndCost(pExec, "fred", 54.10)))
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByNameAndCost(pExec, "Thingie", 56.23)))

	//using a map of [string]interface{} works too!
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByIDMap(pExec, 10))))
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByNameAndCostMap(pExec, "Thingie", 56.23))))

	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByID(pExec, 11))))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.UpdateMap(pExec, m))))
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByID(pExec, 11))))

	//searching using a slice
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByIDSlice(pExec, []int{1, 3, 5}))))
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByIDSliceAndName(pExec, []int{1, 3, 5}, "person1"))))
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByIDSliceNameAndCost(pExec, []int{1, 3, 5}, "person3", nil))))
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByIDSliceCostAndNameSlice(pExec, []int{1, 3, 5}, []string{"person3", "person5"}, nil))))

	//using positional parameters instead of names
	logger.Log(c, logger.DEBUG, fmt.Sprintln((productDao.FindByNameAndCostUnlabeled(pExec, "Thingie", 56.23))))

	exec.Commit()
}

func setupDbPostgres(c context.Context) *sql.DB {
	//db, err := sql.Open("postgres", "postgres://jon:jon@localhost/jon?sslmode=disable")
	db, err := sql.Open("timer", "postgres postgres://jon:jon@localhost/jon?sslmode=disable")

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
	populate(c, db, productDaoPostgres)
	return db
}

func setupDbSqlite(c context.Context) *sql.DB {
	os.Remove("./proteus_test.db")

	//db, err := sql.Open("sqlite3", "./proteus_test.db")
	db, err := sql.Open("timer", "sqlite3 ./proteus_test.db")

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
	populate(c, db, productDaoSqlite)
	return db
}

func populate(c context.Context, db *sql.DB, productDao ProductDao) {
	tx, err := db.Begin()
	if err != nil {
		logger.Log(c, logger.FATAL, fmt.Sprintln(err))
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
			logger.Log(c, logger.FATAL, fmt.Sprintln(err))
		}
		logger.Log(c, logger.DEBUG, fmt.Sprintln(rowCount))
	}
	tx.Commit()
}
