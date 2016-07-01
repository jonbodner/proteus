package main

import (
	"database/sql"
	"fmt"
	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/adapter"
	"github.com/jonbodner/proteus/api"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/Sirupsen/logrus"
	"os"
)

type Product struct {
	Id   int     `prof:"id,pk"`
	Name string  `prof:"name"`
	Cost float64 `prof:"cost"`
}

type ProductDao struct {
	FindById             func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIdMap          func(e api.Executor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap            func(e api.Executor, p map[string]interface{}) (int64, error)                     `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap func(e api.Executor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
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

	gExec := adapter.Sql(exec)

	fmt.Println(productDao.FindById(gExec, 10))
	p := Product{10, "Thingie", 56.23}
	fmt.Println(productDao.Update(gExec, p))
	fmt.Println(productDao.FindById(gExec, 10))
	fmt.Println(productDao.FindByNameAndCost(gExec, "fred", 54.10))
	fmt.Println(productDao.FindByNameAndCost(gExec, "Thingie", 56.23))

	//using a map of [string]interface{} works too!
	fmt.Println(productDao.FindByIdMap(gExec, 10))
	fmt.Println(productDao.FindByNameAndCostMap(gExec, "Thingie", 56.23))

	fmt.Println(productDao.FindById(gExec, 11))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	fmt.Println(productDao.UpdateMap(gExec, m))
	fmt.Println(productDao.FindById(gExec, 11))

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
	stmt, err := tx.Prepare("insert into product(id, name, cost) values(?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for i := 0; i < 100; i++ {
		_, err = stmt.Exec(i, fmt.Sprintf("person%d", i), 1.1*float64(i))
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()
	return db
}
