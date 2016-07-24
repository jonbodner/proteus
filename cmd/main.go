package main

import (
	"database/sql"
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
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
	FindByID                      func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(e api.Executor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(e api.Executor, p map[string]interface{}) (int64, error)                     `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(e api.Executor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(e api.Executor, id int, name string, cost *float64) (int64, error)           `proe:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(e api.Executor, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(e api.Executor, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(e api.Executor, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(e api.Executor, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
}

var productDaoSqlite = ProductDao{}
var productDaoPostgres = ProductDao{}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{})
	err := proteus.Build(&productDaoPostgres, adapter.Postgres)
	if err != nil {
		panic(err)
	}

	err = proteus.Build(&productDaoSqlite, adapter.Sqlite)
	if err != nil {
		panic(err)
	}
}

type setupDb func() *sql.DB

func main() {
	run(setupDbSqlite, productDaoSqlite)
	run(setupDbPostgres, productDaoPostgres)
}

func run(setupDb setupDb, productDao ProductDao) {
	db := setupDb()
	defer db.Close()
	exec, err := db.Begin()
	if err != nil {
		panic(err)
	}

	pExec := adapter.Sql(exec)

	log.Debug(productDao.FindByID(pExec, 10))
	cost := new(float64)
	*cost = 56.23
	p := Product{10, "Thingie", cost}
	log.Debug(productDao.Update(pExec, p))
	log.Debug(productDao.FindByID(pExec, 10))
	log.Debug(productDao.FindByNameAndCost(pExec, "fred", 54.10))
	log.Debug(productDao.FindByNameAndCost(pExec, "Thingie", 56.23))

	//using a map of [string]interface{} works too!
	log.Debug(productDao.FindByIDMap(pExec, 10))
	log.Debug(productDao.FindByNameAndCostMap(pExec, "Thingie", 56.23))

	log.Debug(productDao.FindByID(pExec, 11))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	log.Debug(productDao.UpdateMap(pExec, m))
	log.Debug(productDao.FindByID(pExec, 11))

	log.Debug(productDao.FindByIDSlice(pExec, []int{1, 3, 5}))
	log.Debug(productDao.FindByIDSliceAndName(pExec, []int{1, 3, 5}, "person1"))
	log.Debug(productDao.FindByIDSliceNameAndCost(pExec, []int{1, 3, 5}, "person3", nil))
	log.Debug(productDao.FindByIDSliceCostAndNameSlice(pExec, []int{1, 3, 5}, []string{"person3", "person5"}, nil))

	exec.Commit()
}

func setupDbPostgres() *sql.DB {
	db, err := sql.Open("postgres", "postgres://jon:jon@localhost/jon")

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
	populate(db, productDaoPostgres)
	return db
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
	populate(db, productDaoSqlite)
	return db
}

func populate(db *sql.DB, productDao ProductDao) {
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	pExec := adapter.Sql(tx)

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
