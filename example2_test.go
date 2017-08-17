package proteus

import (
	"database/sql"
	"fmt"
	"log"
)

type CreateProductDao struct {
	Insert func(e Executor, id int, name string, cost float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

func Example_create() {
	db, err := sql.Open("sqlite3", "./proteus_test.db")
	if err != nil {
		log.Fatal(err)
	}

	var productDao = CreateProductDao{}
	err = Build(&productDao, Sqlite)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	pExec := Wrap(tx)

	for i := 0; i < 100; i++ {
		_, err = productDao.Insert(pExec, i, fmt.Sprintf("person%d", i), 1.1*float64(i))
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()
}
