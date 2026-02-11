package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/logger"
	_ "github.com/lib/pq"
)

type Product2 struct {
	ID   int             `prof:"id"`
	Name string          `prof:"name"`
	Cost sql.NullFloat64 `prof:"cost"`
}

func (p Product2) String() string {
	c := "<nil>"
	if p.Cost.Valid {
		c = fmt.Sprintf("%f", p.Cost.Float64)
	}
	return fmt.Sprintf("%d: %s(%s)", p.ID, p.Name, c)
}

type Product2Dao struct {
	FindByID          func(ctx context.Context, e proteus.ContextQuerier, id int) (Product2, error)                           `proq:"select * from Product where id = :id:" prop:"id"`
	Update            func(ctx context.Context, e proteus.ContextExecutor, p Product2) (int64, error)                         `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.ID:" prop:"p"`
	FindByNameAndCost func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]Product2, error)      `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert            func(ctx context.Context, e proteus.ContextExecutor, id int, name string, cost *float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

type closer func(error) error

func main() {
	proteus.SetLogLevel(logger.DEBUG)
	logger.Config(logger.LoggerFunc(func(vals ...interface{}) error {
		fmt.Printf("%s: (%s) - %s\n", vals[1], vals[3], vals[5])
		return nil
	}))
	var product2DaoPostgres Product2Dao
	err := proteus.Build(&product2DaoPostgres, proteus.Postgres)
	if err != nil {
		panic(err)
	}
	run(product2DaoPostgres)
}

func run(productDao Product2Dao) {
	ctx := logger.WithLevel(context.Background(), logger.DEBUG)
	tx, closer := setupDBPostgres(ctx)
	var err error
	defer func() {
		err = closer(err)
	}()

	err = defineSchema(ctx, tx)
	if err != nil {
		return
	}

	populate(ctx, tx, productDao)

	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDao.FindByID(ctx, tx, 10)))
	cost := sql.NullFloat64{
		Float64: 56.23,
		Valid:   true,
	}
	p := Product2{10, "Thingie", cost}
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDao.Update(ctx, tx, p)))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDao.FindByID(ctx, tx, 10)))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDao.FindByNameAndCost(ctx, tx, "fred", 54.10)))
	logger.Log(ctx, logger.DEBUG, fmt.Sprintln(productDao.FindByNameAndCost(ctx, tx, "Thingie", 56.23)))
}

func defineSchema(ctx context.Context, tx proteus.ContextWrapper) error {
	sqlStmt := `
	drop table if exists product;
	create table product (id integer not null primary key, name text, cost real);
	`
	_, err := tx.ExecContext(ctx, sqlStmt)
	if err != nil {
		logger.Log(ctx, logger.FATAL, fmt.Sprintf("%q: %s\n", err, sqlStmt))
		return err
	}
	return nil
}

func setupDBPostgres(ctx context.Context) (proteus.ContextWrapper, closer) {
	db, err := sql.Open("postgres", "postgres://pro_user:pro_pwd@localhost/proteus?sslmode=disable")

	if err != nil {
		logger.Log(ctx, logger.FATAL, fmt.Sprintln(err))
		os.Exit(1)
	}

	tx, err := db.Begin()
	if err != nil {
		logger.Log(ctx, logger.FATAL, fmt.Sprintln(err))
		os.Exit(1)
	}

	return tx, func(err error) error {
		_ = tx.Rollback()
		_ = db.Close()
		return err
	}
}

func populate(ctx context.Context, w proteus.ContextWrapper, productDao Product2Dao) {
	for i := 0; i < 100; i++ {
		var cost *float64
		if i%2 == 0 {
			c := 1.1 * float64(i)
			cost = &c
		}
		rowCount, err := productDao.Insert(ctx, w, i, fmt.Sprintf("person%d", i), cost)
		if err != nil {
			logger.Log(ctx, logger.FATAL, fmt.Sprintln(err))
			os.Exit(1)
		}
		logger.Log(ctx, logger.DEBUG, fmt.Sprintln(rowCount))
	}
}
