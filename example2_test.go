package proteus

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

type CreateProductDao struct {
	Insert func(ctx context.Context, e ContextExecutor, id int, name string, cost float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

func Example_create() {
	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}

	var productDao = CreateProductDao{}
	err = Build(&productDao, Postgres)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Commit()

	ctx := context.Background()

	for i := 0; i < 100; i++ {
		_, err = productDao.Insert(ctx, tx, i, fmt.Sprintf("person%d", i), 1.1*float64(i))
		if err != nil {
			log.Fatal(err)
		}
	}
}
