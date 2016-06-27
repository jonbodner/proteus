package main

import (
	"fmt"
	"github.com/jonbodner/gdb"
	"strings"
	"github.com/jonbodner/gdb/api"
	"github.com/jonbodner/gdb/adapter"
)

type Product struct {
	Id   int     `gdbf:"id,pk"`
	Name string  `gdbf:"name"`
	Cost float64 `gdbf:"cost"`
}

type ProductDao struct {
	FindById          func(e api.Executor, id int) Product                      `gdbq:"select * from Product where id = :id" gdbp:"id"`
	Update            func(e api.Executor, p Product) int64                     `gdbe:"update Product set name = :p.Name, cost = :p.Cost where id = :p.Id" gdbp:"p"`
	FindByNameAndCost func(e api.Executor, name string, cost float64) []Product `gdbq:"select * from Product where name=:name and cost=:cost" gdbp:"name,cost"`
}

type ProductDaoW struct {
	ProductDao
}

func (pd ProductDaoW) FindById(e api.Executor, id int) Product {
	return pd.ProductDao.FindById(e, id)
}

func (pd ProductDaoW) Update(e api.Executor, p Product) int64 {
	return pd.ProductDao.Update(e, p)
}

func (pd ProductDaoW) FindByNameAndCost(e api.Executor, name string, cost float64) []Product {
	return pd.ProductDao.FindByNameAndCost(e, name, cost)
}

type ProductDaoInt interface {
	FindById(e api.Executor, id int) Product
	Update(e api.Executor, p Product) int64
	FindByNameAndCost(e api.Executor, name string, cost float64) []Product
}

var productDao ProductDaoInt

func init() {
	v := ProductDao{}
	err := gdb.Build(&v, adapter.Sqlite)
	if err != nil {
		fmt.Println(err)
	}
	productDao = ProductDaoW{v}
}

type FakeResult bool

func (fr FakeResult) LastInsertId() (int64, error) {
	if fr {
		return 1, nil
	}
	return 0, nil
}

func (fr FakeResult) RowsAffected() (int64, error) {
	return 1, nil
}

type FakeRows struct{}

func (fr FakeRows) Next() bool {
	return false
}

func (fr FakeRows) Err() error {
	return nil
}

func (fr FakeRows) Columns() ([]string, error) {
	return []string{"id", "name", "count"}, nil
}

func (fr FakeRows) Scan(dest ...interface{}) error {
	return nil
}

func (fr FakeRows) Close() error {
	return nil
}

type FakeExecutor struct{}

func (fe FakeExecutor) Exec(query string, args ...interface{}) (api.Result, error) {
	fmt.Println("Exec for query", query, "with params", args)
	res := strings.HasPrefix(query, "insert")
	return FakeResult(res), nil
}

func (fe FakeExecutor) Query(query string, args ...interface{}) (api.Rows, error) {
	fmt.Println("Query for query", query, "with params", args)
	fr := &FakeRows{}
	return fr, nil
}

func main() {
	fe := FakeExecutor{}
	fmt.Println(productDao.FindById(fe, 10))
	p := Product{10, "Thingie", 56.23}
	fmt.Println(productDao.Update(fe, p))
	fmt.Println(productDao.FindByNameAndCost(fe, "fred", 54.10))
}
