package proteus

import (
	"database/sql"
	"fmt"
	"log"
)

type Product struct {
	Id   int     `prof:"id"`
	Name string  `prof:"name"`
	Cost float64 `prof:"cost"`
}

type ProductDao struct {
	FindById             func(e Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(e Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(e Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIdMap          func(e Executor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap            func(e Executor, p map[string]interface{}) (int64, error)                     `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap func(e Executor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
}

func Example_readUpdate() {
	productDao := ProductDao{}
	err := Build(&productDao, Sqlite)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}

	exec, err := db.Begin()
	if err != nil {
		panic(err)
	}

	gExec := Wrap(exec)

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
