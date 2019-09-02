package proteus

import (
	"context"
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
	FindById             func(ctx context.Context, e ContextExecutor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(ctx context.Context, e ContextExecutor, p Product) (int64, error)                                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(ctx context.Context, e ContextExecutor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIdMap          func(ctx context.Context, e ContextExecutor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap            func(ctx context.Context, e ContextExecutor, p map[string]interface{}) (int64, error)                     `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap func(ctx context.Context, e ContextExecutor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
}

func Example_readUpdate() {
	productDao := ProductDao{}
	err := Build(&productDao, Postgres)
	if err != nil {
		panic(err)
	}

	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	defer tx.Commit()

	fmt.Println(productDao.FindById(ctx, tx, 10))
	p := Product{10, "Thingie", 56.23}
	fmt.Println(productDao.Update(ctx, tx, p))
	fmt.Println(productDao.FindById(ctx, tx, 10))
	fmt.Println(productDao.FindByNameAndCost(ctx, tx, "fred", 54.10))
	fmt.Println(productDao.FindByNameAndCost(ctx, tx, "Thingie", 56.23))

	//using a map of [string]interface{} works too!
	fmt.Println(productDao.FindByIdMap(ctx, tx, 10))
	fmt.Println(productDao.FindByNameAndCostMap(ctx, tx, "Thingie", 56.23))

	fmt.Println(productDao.FindById(ctx, tx, 11))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	fmt.Println(productDao.UpdateMap(ctx, tx, m))
	fmt.Println(productDao.FindById(ctx, tx, 11))
}
