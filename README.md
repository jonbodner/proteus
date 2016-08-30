# Proteus
[![Go Report Card](https://goreportcard.com/badge/github.com/jonbodner/proteus)](https://goreportcard.com/report/github.com/jonbodner/proteus)
[![Join the chat at https://gitter.im/jonbodner/proteus](https://badges.gitter.im/jonbodner/proteus.svg)](https://gitter.im/jonbodner/proteus?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
A simple tool for generating an application's data access layer.

## Purpose

## Quick Start
1. Define a struct that contains function fields and tags to indicate the query and the parameter names:

```go
type ProductDao struct {
	FindById                      func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIdMap                   func(e api.Executor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(e api.Executor, p map[string]interface{}) (int64, error)                     `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(e api.Executor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDSlice                 func(e api.Executor, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(e api.Executor, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(e api.Executor, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(e api.Executor, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
}
```

The first input parameter is always of type api.Executor.
```go
// Executor runs the queries that are processed by proteus.
type Executor interface {
	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(query string, args ...interface{}) (Result, error)

	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(query string, args ...interface{}) (Rows, error)
}
```

The remaining input parameters can be primitives, structs, maps of string to interface{}, or slices. 

For queries, return types can be:
- empty
- a single value being returned (a primitive, struct, or map of string to interface{})
- a single value that's a slice of primitive, struct, or a map of string to interface{}
- a primitive, struct, or map of string to interface{} and an error
- a slice of primitive, struct, or a map of string to interface{} and an error

For insert/updates, return types can be:
- empty
- an int64 that indicates the number of rows affected
- an int64 that indicates the number of rows affected and an error

2\. If you want to map response fields to a struct, define a struct with struct tags to indicate the mapping:

```go
type Product struct {
	Id   int     `prof:"id"`
	Name string  `prof:"name"`
	Cost float64 `prof:"cost"`
}
```

3\. Pass an instance of the Dao struct to the proteus.Build function:

```go
var productDao = ProductDao{}

func init() {
	err := proteus.Build(&productDao, adapter.Sqlite)
	if err != nil {
		panic(err)
	}
}
```

4\. Open a connection to a data store that meets the Proteus interface:

```go
	db := setupDb()
	defer db.Close()
	exec := adapter.Sql(db)
```

5\. Make calls to the function fields in your Proteus-populated struct:

```go
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

	fmt.Println(productDao.FindByIDSlice(pExec, []int{1, 3, 5}))
	fmt.Println(productDao.FindByIDSliceAndName(pExec, []int{1, 3, 5}, "person1"))
	fmt.Println(productDao.FindByIDSliceNameAndCost(pExec, []int{1, 3, 5}, "person3", nil))
	fmt.Println(productDao.FindByIDSliceCostAndNameSlice(pExec, []int{1, 3, 5}, []string{"person3", "person5"}, nil))
```

## Struct Tags

Proteus generates implementations of DAO functions by examining struct tags and parameters types on function fields in a struct.
The following are the recognized struct tags:

- proq - Query run by Executor.Query. Returns single entity or list of entities
- proe - Query run by Executor.Exec. Returns the number of rows changed
- prop - The parameter names. Should be in the order of the function parameters (skipping over the first Executor parameter)

The prop struct tag is optional. If it is not supplied, the query must contain positional parameters ($1, $2, etc.) instead
of named parameters. For example:

```go
type ProductDaoS struct {
	FindById             func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :$1:"`
	Update               func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :$1.Name:, cost = :$1.Cost: where id = :$1.Id:"`
	FindByNameAndCost    func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}
```

If you want to map the output of a DAO with a proq tag to a struct, then create a struct and put
 the following struct tag on each field that you want to map to a value in the output:
- prof - The fields on the dto that are mapped to select parameters in a query

## Valid function signatures

## API

## FAQ
1\. Why doesn't Proteus generate a struct that meets an interface?

Go has some surprising limitations. One of them is that there is no way to use reflection to build a implementation
of a method; you can only build implementations of functions. The difference is subtle, but the net result is that you cannot supply an
interface to the reflection API and get back something that implements that interface.

Another go limitation is that there is also no way to attach metadata to an interface's method. only struct fields can
have associated metadata.

A third limitation is that a go interface is not satisfied by a struct that has fields of function type, even if the
names of the functions and the types of the function parameters match an interface.

Given these limitations, Proteus uses structs to hold the generated functions. If you want an interface that describes
the functionality provided by the struct, you can do something like this:
```go
type ProductDaoS struct {
	FindById             func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
}

type ProductDao interface {
    FindById(api.Executor, int) (Product, error)
    Update(api.Executor, Product) (int64, error)
    FindByNameAndCost(api.Executor, string, float64) ([]Product, error)
}

type productWrapper struct {
    ProductDaoS
}

func (pw productWrapper) FindById(exec api.Executor, id int) (Product, error) {
    return pw.ProductDaoS.FindById(exec, id)
}

func (pw productWrapper) Update(exec api.Executor, p Product) (int64, error) {
    return pw.ProductDaoS.Update(exec, p)
}

func (pw productWrapper) FindByNameAndCost(exec api.Executor, n string, c float64) ([]Product, error) {
    return pw.ProductDaoS.FindByNameAndCost(exec, n, c)
}

func NewProductDao() ProductDao {
    p := ProductDaoS{}
    proteus.Build(&p,adapter.Sqlite)
    return productWrapper{p}
}
```

A future version of Proteus may include a tool that can be used with go generate to automatically create the wrapper and interface.

2\. Why do I have to specify the parameter names with a struct tag?

This is another limitation of go's reflection API. The names of parameters are not available at runtime to be inspected,
and must be supplied by another way in order to be referenced in a query.

## Future Directions

There are more interesting features coming to Proteus. They are (in likely order of implementation):

- Support for storing queries in property files
```go
type FooDAO struct {
	Find func(e api.Executor, ids int) (Product, error) `proqe:"product.get" prop:"ids"`
}


func UseFoo() {
	db, _ := sql.Open("sqlite3", "./proteus_test.db")
	defer db.Close()

	var queryStore api.QueryStore //this will be a new interface
	fooDAO := FooDAO{}
	proteus.BuildWithExternalQueries(&fooDAO, adapter.Sqlite, queryStore)
	exec := adapter.Sql(db)
	fmt.Println(fooDAO.Find(exec, 1))
}
```

- go generate tool to create a wrapper struct and interface for a Proteus DAO

- API implementations for nonSQL data stores and HTTP requests

- Additional struct tags to generate basic CRUD queries automatically. This is about as far as I'd go in implementing
ORM-like features. The goal for Proteus is to be an adapter between a data source and business logic, not to define your
entire programming model.
