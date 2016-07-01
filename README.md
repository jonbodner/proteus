# Proteus
A simple tool for generating an application's data access layer.

## Purpose

## Quick Start
1. Define a struct that contains function fields and tags to indicate the query and the parameter names:

```
type ProductDao struct {
	FindById             func(e api.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(e api.Executor, p Product) (int64, error)                                    `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(e api.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIdMap          func(e api.Executor, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap            func(e api.Executor, p map[string]interface{}) (int64, error)                     `proe:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap func(e api.Executor, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
}
```

Input parameter types can be primitives, structs, or maps of string to interface{}.

2. If you want to map response fields to a struct, define a struct with struct tags to indicate the mapping:
```
type Product struct {
	Id   int     `prof:"id"`
	Name string  `prof:"name"`
	Cost float64 `prof:"cost"`
}
```

3. Pass an instance of the Dao struct to the proteus.Build function:
```
var productDao = ProductDao{}

func init() {
	err := proteus.Build(&productDao, adapter.Sqlite)
	if err != nil {
		panic(err)
	}
}
```

4. Open a connection to a data store that meets the Proteus interface:
```
	db := setupDb()
	defer db.Close()
	exec := adapter.Sql(db)
```

5. Make calls to the function fields in your Proteus-populated struct:
```
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
```

## Struct Tags

Proteus generates implementations of DAO functions by examining struct tags and parameters types on function fields in a struct.
The following are the recognized struct tags:

- proq - Query run by Executor.Query. Returns single entity or list of entities
- proe - Query run by Executor.Exec. Returns new id (if sql.Result has a non-zero value for LastInsertId) or number of rows changed
- prop - The parameter names. Should be in order for the function parameters (skipping over the first Executor parameter)

If you want to map the output of a DAO with a proq tag to a struct, then create a struct and put
 the following struct tag on each field that you want to map to a value in the output:
- prof - The fields on the dto that are mapped to select parameters in a query

## Valid function signatures

## API

## FAQ
1. Why doesn't Proteus generate a struct that meets an interface?

The reflection API in go has several limitations. One of them is that there is no way to use reflection to build a implementation
of a method; you can only build implementations of functions. The difference is subtle, but the net result is that you cannot supply an
interface to the reflection API and get back something that implements that interface.

Another go limitation is that there is also no way to attach metadata to an interface's method. only struct fields can
have associated metadata.

A third limitation is that a go interface is not satisfied by a struct that has fields of function type, even if the
names of the functions and the types of the function parameters match an interface.

Given these limitations, Proteus uses structs to hold the generated functions. If you want an interface that describes
the functionality provided by the struct, you can do something like this:
```
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

2. Why do I have to specify the parameter names with a struct tag?

This is another limitation of go's reflection API. The names of parameters are not available at runtime to be inspected,
and must be supplied by another way in order to be referenced in a query.

## Future Directions

There are more interesting features coming to Proteus. They are (in likely order of implementation):

- Support for slice input parameters
```
type FooDAO struct {
	FindSeveral func(e api.Executor, ids []int) ([]Product, error) `proq:"select * from Product where id in (:ids:)" prop:"ids"`
}


func UseFoo() {
	db, _ := sql.Open("sqlite3", "./proteus_test.db")
	defer db.Close()

	fooDAO := FooDAO{}
	proteus.Build(&fooDAO, adapter.Sqlite)
	ids := []int{1,2,3}
	exec := adapter.Sql(db)
	fmt.Println(fooDAO.FindSeveral(exec, ids)) //return any Products that have ids 1, 2, or 3
}
```

- Support for storing queries in property files
```
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