package main

import (
	"database/sql"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

type Product struct {
	Id   int     `prof:"id"`
	Name string  `prof:"name"`
	Cost *float64 `prof:"cost"`
}

func (p Product) String() string {
	c := "<nil>"
	if p.Cost != nil {
		c = fmt.Sprintf("%f",*p.Cost)
	}
	return fmt.Sprintf("%d: %s(%s)",p.Id, p.Name, c)
}

type ProductDao struct {
	FindById             func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIdMap          func(e api.Executor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap            func(e api.Executor, p map[string]interface{}) (int64, error)                     `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap func(e api.Executor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert               func(e api.Executor, id int, name string, cost *float64) (int64, error)            `proe:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

var productDao = ProductDao{}

func init() {
	err := proteus.Build(&productDao, adapter.Sqlite)
	if err != nil {
		panic(err)
	}
}

func main() {
	db := setupDb()
	defer db.Close()
	exec, err := db.Begin()
	if err != nil {
		panic(err)
	}

	pExec := adapter.Sql(exec)

	fmt.Println(productDao.FindById(pExec, 10))
	cost := new(float64)
	*cost = 56.23
	p := Product{10, "Thingie", cost}
	fmt.Println(productDao.Update(pExec, p))
	fmt.Println(productDao.FindById(pExec, 10))
	fmt.Println(productDao.FindByNameAndCost(pExec, "fred", 54.10))
	fmt.Println(productDao.FindByNameAndCost(pExec, "Thingie", 56.23))

	//using a map of [string]interface{} works too!
	fmt.Println(productDao.FindByIdMap(pExec, 10))
	fmt.Println(productDao.FindByNameAndCostMap(pExec, "Thingie", 56.23))

	fmt.Println(productDao.FindById(pExec, 11))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	fmt.Println(productDao.UpdateMap(pExec, m))
	fmt.Println(productDao.FindById(pExec, 11))

	exec.Commit()
}

func setupDb() *sql.DB {
	os.Remove("./proteus_test.db")

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}
	sqlStmt := `
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	pExec := adapter.Sql(tx)

	for i := 0; i < 100; i++ {
		var cost *float64
		if i % 2 == 0 {
			c := 1.1*float64(i)
			cost = &c
		}
		_, err = productDao.Insert(pExec, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()
	return db
}
