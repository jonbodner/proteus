# Proteus

[![Go Report Card](https://goreportcard.com/badge/github.com/jonbodner/proteus)](https://goreportcard.com/report/github.com/jonbodner/proteus)
[![Sourcegraph](https://sourcegraph.com/github.com/jonbodner/proteus/-/badge.svg)](https://sourcegraph.com/github.com/jonbodner/proteus?badge)
[![Join the chat at https://gitter.im/jonbodner/proteus](https://badges.gitter.im/jonbodner/proteus.svg)](https://gitter.im/jonbodner/proteus?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

A simple tool for generating an application's data access layer.

## Purpose

## Quick Start
1. Define a struct that contains function fields and tags to indicate the query and the parameter names:

```go
type ProductDao struct {
	FindByID                      func(e proteus.Querier, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(e proteus.Executor, p Product) (int64, error)                                   `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(e proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(e proteus.Querier, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(e proteus.Executor, p map[string]interface{}) (int64, error)                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(e proteus.Querier, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(e proteus.Executor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(e proteus.Querier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(e proteus.Querier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(e proteus.Querier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(e proteus.Querier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(e proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}
```

The first input parameter is either of type `proteus.Executor` or `proteus.Querier`:
```go
// Executor runs queries that modify the data store.
type Executor interface {
	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(query string, args ...interface{}) (sql.Result, error)
}
```
```go
// Querier runs queries that return Rows from the data store
type Querier interface {
	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
```

As of Proteus 0.10.0, you can also pass a context into your SQL queries:

```go
type ProductDaoCtx struct {
	FindByID                      func(ctx context.Context, e proteus.ContextQuerier, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(ctx context.Context, e proteus.ContextExecutor, p Product) (int64, error)                                   `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(ctx context.Context, e proteus.ContextQuerier, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(ctx context.Context, e proteus.ContextExecutor, p map[string]interface{}) (int64, error)                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(ctx context.Context, e proteus.ContextExecutor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(ctx context.Context, e proteus.ContextQuerier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(ctx context.Context, e proteus.ContextQuerier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(ctx context.Context, e proteus.ContextQuerier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(ctx context.Context, e proteus.ContextQuerier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(ctx context.Context, e proteus.ContextQuerier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}
```

The first input parameter is of type `context.Context`, and the second input parameter is of type `proteus.ContextExecutor` or `proteus.ContextQuerier`:

```go
// ContextQuerier defines the interface of a type that runs a SQL query with a context
type ContextQuerier interface {
	// QueryContext executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}
```

```go
// ContextExecutor defines the interface of a type that runs a SQL exec with a context
type ContextExecutor interface {
	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
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
	err := proteus.Build(&productDao, proteus.Sqlite)
	if err != nil {
		panic(err)
	}
}
```

The proteus.Build factory function only returns errors if the wrong data type is passed in for the first parameter. If you want errors returned when there is a failure to generate a function field, use `proteus.ShouldBuild` instead.

4\. Open a connection to a SQL database:

```go
	db := setupDb()
	defer db.Close()
```

5\. Make calls to the function fields in your Proteus-populated struct:

```go
	fmt.Println(productDao.FindById(db, 10))
	p := Product{10, "Thingie", 56.23}
	fmt.Println(productDao.Update(db, p))
	fmt.Println(productDao.FindById(db, 10))
	fmt.Println(productDao.FindByNameAndCost(db, "fred", 54.10))
	fmt.Println(productDao.FindByNameAndCost(db, "Thingie", 56.23))

	//using a map of [string]interface{} works too!
	fmt.Println(productDao.FindByIdMap(db, 10))
	fmt.Println(productDao.FindByNameAndCostMap(db, "Thingie", 56.23))

	fmt.Println(productDao.FindById(db, 11))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	fmt.Println(productDao.UpdateMap(db, m))
	fmt.Println(productDao.FindById(db, 11))

	fmt.Println(productDao.FindByIDSlice(db, []int{1, 3, 5}))
	fmt.Println(productDao.FindByIDSliceAndName(db, []int{1, 3, 5}, "person1"))
	fmt.Println(productDao.FindByIDSliceNameAndCost(db, []int{1, 3, 5}, "person3", nil))
	fmt.Println(productDao.FindByIDSliceCostAndNameSlice(db, []int{1, 3, 5}, []string{"person3", "person5"}, nil))
```

## Struct Tags

Proteus generates implementations of DAO functions by examining struct tags and parameters types on function fields in a struct.
The following are the recognized struct tags:

- `proq` - The query. Returns single entity or list of entities
- `prop` - The parameter names. Should be in the order of the function parameters (skipping over the first Executor or Querier parameter)

The `prop` struct tag is optional. If it is not supplied, the query must contain positional parameters ($1, $2, etc.) instead
of named parameters. For example:

```go
type ProductDaoS struct {
	FindById             func(e proteus.Querier, id int) (Product, error)                                     `proq:"select * from Product where id = :$1:"`
	Update               func(e proteus.Executor, p Product) (int64, error)                                   `proq:"update Product set name = :$1.Name:, cost = :$1.Cost: where id = :$1.Id:"`
	FindByNameAndCost    func(e proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}
```

If you want to map the output of a DAO with a `proq` tag to a struct, then create a struct and put
 the following struct tag on each field that you want to map to a value in the output:
- `prof` - The fields on the dto that are mapped to select parameters in a query

## Storing queries outside of struct tags
Struct tags are cumbersome for all but the shortest queries. In order to allow a more natural way to store longer queries,
one or more instances of the `proteus.QueryMapper` interface can be passed into the `proteus.Build` function. In order to 
reference a query stored in an `proteus.QueryMapper`, you should put `q:name` as the value of the `proq` struct tag, where
`name` is the name for the query.

For example:

```go
	m := proteus.MapMapper{
		"q1": "select * from foo where id = :id:",
		"q2": "update foo set x=:x: where id = :id:",
	}

	type s struct {
		GetF func(e proteus.Querier, id string) (f, error) `proq:"q:q1" prop:"id"`
		Update func(e proteus.Executor, id string, x string) (int64, error) `proq:"q:q2" prop:"id,x"`
	}
	
	sImpl := s{}
	err := proteus.Build(&sImpl, proteus.Sqlite, m)
```

Out of the box, you can use either a `map[string]string` or a unix properties file to store your queries. In order
to use a `map[string]string`, cast your map to `proteus.MapMapper` (or just declare your variable to be of type `proteus.MapMapper`). To use a properties file, call the method 
`proteus.PropFileToQueryMapper` with the name of the property file that contains your queries.

## Context support

As of Proteus 0.10.0, Proteus optionally builds functions that invoke the the `ExecContext` and `QueryContext` methods on `sql.DB` and `sql.Tx`. In order to do so, declare your
function fields with a first parameter of `context.Context` and a second parameter of either `proteus.ContextQuerier` or `ContextExecutor`. You then call `proteus.Build` or 
`proteus.ShouldBuild` as you normally would. When invoking the functions, you pass in a `context.Context` instance as the first parameter, and an instance of `sql.DB` or `sql.Tx`
as the second parameter. 

Note that if you are using the context variants, you cannot use the `proteus.Wrap` function; this is OK, as Wrap is now a no-op and is considered deprecated.

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
	FindById             func(e proteus.Executor, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(e proteus.Executor, p Product) (int64, error)                                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(e proteus.Executor, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
}

type ProductDao interface {
    FindById(proteus.Executor, int) (Product, error)
    Update(proteus.Executor, Product) (int64, error)
    FindByNameAndCost(proteus.Executor, string, float64) ([]Product, error)
}

type productWrapper struct {
    ProductDaoS
}

func (pw productWrapper) FindById(exec proteus.Executor, id int) (Product, error) {
    return pw.ProductDaoS.FindById(exec, id)
}

func (pw productWrapper) Update(exec proteus.Executor, p Product) (int64, error) {
    return pw.ProductDaoS.Update(exec, p)
}

func (pw productWrapper) FindByNameAndCost(exec proteus.Executor, n string, c float64) ([]Product, error) {
    return pw.ProductDaoS.FindByNameAndCost(exec, n, c)
}

func NewProductDao() ProductDao {
    p := ProductDaoS{}
    proteus.Build(&p,proteus.Sqlite)
    return productWrapper{p}
}
```

A future version of Proteus may include a tool that can be used with go generate to automatically create the wrapper and interface.

2\. Why do I have to specify the parameter names with a struct tag?

This is another limitation of go's reflection API. The names of parameters are not available at runtime to be inspected,
and must be supplied by another way in order to be referenced in a query. If you do not want to use a prop struct tag, you
can use positional parameters ($1, $2, etc.) instead.

3\. Why do I need to use the `proteus.Wrap` function to wrap a `sql.DB` or `sql.Tx` from the standard library?

As of Proteus 0.10.0, the `proteus.Wrap` function is no longer needed and should be considered deprecated. Proteus is no longer
written using its own `Rows` interface. Existing code that uses `Wrap` continues to work, but the function is a no-op; it returns
back the instance that was passed in.

## Logging

In order to avoid tying the client code to one particular implementation, Proteus includes its own logger that can be bridged to any other Go logger.

By default, Proteus will log nothing. The simplest way to change the amount of information logged by Proteus is by calling the function `proteus.SetLogLevel`.
This function takes in a value of type `logger.Level`.

If you are using the `proteus.ShouldBuild` function to generate your DAOs, you can supply a logging level by passing in the context returned by the function `logger.WithLevel` .
This value is overridden by the logging level set by `proteus.SetLogLevel`.

You can also include additional values in the logs by passing in the context returned by the function `logger.WithValues`. This function adds one or more `logger.Pair` values to the context.

If you are using the `context.Context` support in Proteus 0.10.0 and later, the logger level can be supplied in the context passed into the function. This will override any logger level
specified when `proteus.ShouldBuild` was invoked. You can also supply additional logging fields for the function call by using a context returned by the `logger.WithValues` function.

All Proteus logging output include 3 default fields (in order):

- time (value is a `time.Time` in UTC)
- level (the level of the log as a `logger.Level`)
- message (the message supplied with the log

The default logger provided with the Proteus logging package outputs to `os.Stdout` in a JSON format. The default logger implementation is configurable; you can
specify a different `io.Writer` by using the call:

```go
    logger.Config(logger.DefaultLogger{
        Writer: myWriterImpl,
    })
```

The format of the output of the DefaultLogger can be configured by supplying a `logger.Formatter`:

```go
    logger.Config(logger.DefaultLogger{
        Writer: myWriterImpl,
        Formatter: myFormatter,
    })
```

A helper type `logger.FormatterFunc` will turn any function with the signature `func(vals ...interface{}) string` into a `logger.Formatter`.

The first 6 values passed to the `Format` method are:

|position|value|
|--------|------|
|0 | "time"|
|1 | a `time.Time` in UTC |
|2 | "level"|
|3 | a `logger.Level`|
|4 | "message"|
|5 | the message for the log|


If you want to supply your own logger implementation, pass an implementation
of `logger.Logger` into `logger.Config`. This interface matches the definition used by go-kit. The default parameters are in the
order specified above. There is a `logger.LoggerFunc` helper type to convert
any function with the signature of `func(vals ...interface{}) error` into a `logger.Logger`.

Feel free to use this logger within your own code. If this logger proves to be useful, it might be broken into its own top-level package.

## Future Directions

There are more interesting features coming to Proteus. They are (in likely order of implementation):

- more expansive performance measurement support and per-request logging control

- go generate tool to create a wrapper struct and interface for a Proteus DAO

- API implementations for nonSQL data stores and HTTP requests

- Additional struct tags to generate basic CRUD queries automatically. This is about as far as I'd go in implementing
ORM-like features. The goal for Proteus is to be an adapter between a data source and business logic, not to define your
entire programming model.
