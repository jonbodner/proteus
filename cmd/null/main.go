package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/jonbodner/proteus"
	"github.com/jonbodner/proteus/logger"
	_ "github.com/lib/pq"
	_ "github.com/mutecomm/go-sqlcipher"
)

type Product2 struct {
	Id   int             `prof:"id"`
	Name string          `prof:"name"`
	Cost sql.NullFloat64 `prof:"cost"`
}

func (p Product2) String() string {
	c := "<nil>"
	if p.Cost.Valid {
		c = fmt.Sprintf("%v", p.Cost)
	}
	return fmt.Sprintf("%d: %s(%s)", p.Id, p.Name, c)
}

type Product2Dao struct {
	FindByID          func(e proteus.Querier, id int) (Product2, error)                           `proq:"select * from Product where id = :id:" prop:"id"`
	Update            func(e proteus.Executor, p Product2) (int64, error)                         `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost func(e proteus.Querier, name string, cost float64) ([]Product2, error)      `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert            func(e proteus.Executor, id int, name string, cost *float64) (int64, error) `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
}

var product2DaoSqlite = Product2Dao{}

func init() {
	proteus.SetLogLevel(logger.DEBUG)
	logger.Config(logger.LoggerFunc(func(vals ...interface{}) error {
		fmt.Printf("%s: (%s) - %s\n", vals[1], vals[3], vals[5])
		return nil
	}))
	err := proteus.Build(&product2DaoSqlite, proteus.Sqlite)
	if err != nil {
		panic(err)
	}
}

type setupDb func(c context.Context) *sql.DB

func main() {
	run(setupDbSqlite, product2DaoSqlite)
}

func run(setupDb setupDb, productDao Product2Dao) {
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	db := setupDb(c)
	defer db.Close()
	exec, err := db.Begin()
	if err != nil {
		panic(err)
	}

	pExec := proteus.Wrap(exec)

	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByID(pExec, 10)))
	cost := sql.NullFloat64{
		Float64: 56.23,
		Valid:   true,
	}
	p := Product2{10, "Thingie", cost}
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.Update(pExec, p)))
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByID(pExec, 10)))
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByNameAndCost(pExec, "fred", 54.10)))
	logger.Log(c, logger.DEBUG, fmt.Sprintln(productDao.FindByNameAndCost(pExec, "Thingie", 56.23)))

	exec.Commit()
}

func setupDbSqlite(c context.Context) *sql.DB {
	os.Remove("./proteus_test.db")

	db, err := sql.Open("sqlite3", "./proteus_test.db")

	if err != nil {
		logger.Log(c, logger.FATAL, fmt.Sprintln(err))
		os.Exit(1)
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
	populate(c, db, product2DaoSqlite)
	return db
}

func populate(c context.Context, db *sql.DB, productDao Product2Dao) {
	tx, err := db.Begin()
	if err != nil {
		logger.Log(c, logger.FATAL, fmt.Sprintln(err))
		os.Exit(1)
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
			os.Exit(1)
		}
		logger.Log(c, logger.DEBUG, fmt.Sprintln(rowCount))
	}
	tx.Commit()
}
