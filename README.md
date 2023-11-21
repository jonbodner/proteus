# Proteus

[![Go Report Card](https://goreportcard.com/badge/github.com/jonbodner/proteus)](https://goreportcard.com/report/github.com/jonbodner/proteus)
[![Sourcegraph](https://sourcegraph.com/github.com/jonbodner/proteus/-/badge.svg)](https://sourcegraph.com/github.com/jonbodner/proteus?badge)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/jonbodner/proteus)](https://pkg.go.dev/github.com/jonbodner/proteus)

A simple tool for generating an application's data access layer.

## Purpose

Proteus makes your SQL queries type-safe and prevents SQL injection attacks. It processes structs with struct tags on function fields to generate 
Go functions at runtime. These functions map input parameters to SQL query parameters and optionally map the output parameters to the output of your
SQL queries.

In addition to being type-safe, Proteus also prevents SQL injection by generating prepared statements from your SQL queries. Even dynamic `in` clauses 
are converted into injection-proof prepared statements.

Proteus is _not_ an ORM; it does not generate SQL. It just automates away the boring parts of interacting with databases in Go.

## Quick Start
1. Define a struct that contains function fields and tags to indicate the query and the parameter names:

```go
type ProductDaoCtx struct {
	FindByID                      func(ctx context.Context, q proteus.ContextQuerier, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(ctx context.Context, e proteus.ContextExecutor, p Product) (int64, error)                                   `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(ctx context.Context, q proteus.ContextQuerier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(ctx context.Context, q proteus.ContextQuerier, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(ctx context.Context, e proteus.ContextExecutor, p map[string]interface{}) (int64, error)                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(ctx context.Context, q proteus.ContextQuerier, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(ctx context.Context, e proteus.ContextExecutor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(ctx context.Context, q proteus.ContextQuerier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(ctx context.Context, q proteus.ContextQuerier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(ctx context.Context, q proteus.ContextQuerier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(ctx context.Context, q proteus.ContextQuerier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(ctx context.Context, q proteus.ContextQuerier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
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
- a `sql.Result`
- an int64 that indicates the number of rows affected and an error
- a `sql.Result` and an error

The `proq` struct tag stores the query. You place variable substitutions between `:` s. Proteus allows you
to refer to fields in maps and structs, as well as elements in arrays or slices using `.` as a path separator. If you have a
struct like this:

```go
type Person struct {
    Name string
    Address Address
    Pets []Pet
} 

type Pet struct {
    Name string
    Species string
}

type Address struct {
    Street string
    City string
    State string
}
```
You can write a query like this:

```
insert into person(name, city, pet1_name, pet2_name) values (:p.Name:, :p.Address.City:, :p.Pets.0.Name:, :p.Pets.1.Name:)
```

Note that the index for an array or slice must be an int literal and the key for a map must be a string.


2\. If you want to map response fields to a struct, define a struct with struct tags to indicate the mapping:

```go
type Product struct {
	Id   int     `prof:"id"`
	Name string  `prof:"name"`
	Cost float64 `prof:"cost"`
}
```

3\. Pass an instance of the Dao struct to the proteus.ShouldBuild function:

```go
var productDao = ProductDao{}

func init() {
	err := proteus.ShouldBuild(context.Background(), &productDao, proteus.Sqlite)
	if err != nil {
		panic(err)
	}
}
```

4\. Open a connection to a SQL database:

```go
	db := setupDb()
	defer db.Close()
```

5\. Make calls to the function fields in your Proteus-populated struct:

```go
    ctx := context.Background()
	fmt.Println(productDao.FindById(ctx, db, 10))
	p := Product{10, "Thingie", 56.23}
	fmt.Println(productDao.Update(ctx, db, p))
	fmt.Println(productDao.FindById(ctx, db, 10))
	fmt.Println(productDao.FindByNameAndCost(ctx, db, "fred", 54.10))
	fmt.Println(productDao.FindByNameAndCost(ctx, db, "Thingie", 56.23))

	//using a map of [string]interface{} works too!
	fmt.Println(productDao.FindByIdMap(ctx, db, 10))
	fmt.Println(productDao.FindByNameAndCostMap(ctx, db, "Thingie", 56.23))

	fmt.Println(productDao.FindById(ctx, db, 11))
	m := map[string]interface{}{
		"Id":   11,
		"Name": "bobbo",
		"Cost": 12.94,
	}
	fmt.Println(productDao.UpdateMap(ctx, db, m))
	fmt.Println(productDao.FindById(ctx, db, 11))

	fmt.Println(productDao.FindByIDSlice(ctx, db, []int{1, 3, 5}))
	fmt.Println(productDao.FindByIDSliceAndName(ctx, db, []int{1, 3, 5}, "person1"))
	fmt.Println(productDao.FindByIDSliceNameAndCost(ctx, db, []int{1, 3, 5}, "person3", nil))
	fmt.Println(productDao.FindByIDSliceCostAndNameSlice(ctx, db, []int{1, 3, 5}, []string{"person3", "person5"}, nil))
```

### Proteus without the context

If you are using an older database driver that does not work with the `context.Context`, Proteus provides support for them as well:

```go
type ProductDao struct {
	FindByID                      func(q proteus.Querier, id int) (Product, error)                                     `proq:"select * from Product where id = :id:" prop:"id"`
	Update                        func(e proteus.Executor, p Product) (int64, error)                                   `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost             func(q proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	FindByIDMap                   func(q proteus.Querier, id int) (map[string]interface{}, error)                      `proq:"select * from Product where id = :id:" prop:"id"`
	UpdateMap                     func(e proteus.Executor, p map[string]interface{}) (int64, error)                    `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCostMap          func(q proteus.Querier, name string, cost float64) ([]map[string]interface{}, error) `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
	Insert                        func(e proteus.Executor, id int, name string, cost *float64) (int64, error)          `proq:"insert into product(id, name, cost) values(:id:, :name:, :cost:)" prop:"id,name,cost"`
	FindByIDSlice                 func(q proteus.Querier, ids []int) ([]Product, error)                                `proq:"select * from Product where id in (:ids:)" prop:"ids"`
	FindByIDSliceAndName          func(q proteus.Querier, ids []int, name string) ([]Product, error)                   `proq:"select * from Product where name = :name: and id in (:ids:)" prop:"ids,name"`
	FindByIDSliceNameAndCost      func(q proteus.Querier, ids []int, name string, cost *float64) ([]Product, error)    `proq:"select * from Product where name = :name: and id in (:ids:) and (cost is null or cost = :cost:)" prop:"ids,name,cost"`
	FindByIDSliceCostAndNameSlice func(q proteus.Querier, ids []int, names []string, cost *float64) ([]Product, error) `proq:"select * from Product where id in (:ids:) and (cost is null or cost = :cost:) and name in (:names:)" prop:"ids,names,cost"`
	FindByNameAndCostUnlabeled    func(q proteus.Querier, name string, cost float64) ([]Product, error)                `proq:"select * from Product where name=:$1: and cost=:$2:"`
}
```

In this case, the first input parameter is either of type `proteus.Executor` or `proteus.Querier`:

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

Use `proteus.Build` to generate your DAO functions:

```go
var productDao = ProductDao{}

func init() {
	err := proteus.Build(&productDao, proteus.Sqlite)
	if err != nil {
		panic(err)
	}
}
```
### Database error behavior configuration

Having every DAO function returning an error can make things very cumbersone and repetitive quickly. Sometimes you just want to return a single value without having to check for errors because you are sure that any errors would be fatal, unrecoverable and require human intervention.

```go
type UserRepository struct {
	Fetch () func (ctx context.Context, q ContextQuerier)[]*models.Users `proq:"select * from users"`
}
```
The above struct works fine, but if the database throws an error, you won't be able to know. The following configurations will control the behavior of such scenarios.

By passing the following values in the `context.Context` to the `ShouldBuild` function.

```go
c := context.WithValue(context.Background(), ContextKeyErrorBehavior, PanicAlways)
ShouldBuild(c, &dao, Postgres)
```

**`ErrorBehavior` Values**:


- `DoNothing` - proteus does not do anything when the underlying data source throws an error.
- `PanicWhenAbsent` - proteus will `panic`, if the DAO function being called does not have the `error` return type.
- `PanicAlways` - proteus will always `panic` if there is an error from the data source, whether or not the DAO function being called indicates an `error` return type.



## Struct Tags

Proteus generates implementations of DAO functions by examining struct tags and parameters types on function fields in a struct.
The following are the recognized struct tags:

- `proq` - The query. Returns single entity or list of entities
- `prop` - The parameter names. Should be in the order of the function parameters (skipping over the first Executor or Querier parameter)

The `prop` struct tag is optional. If it is not supplied, the query must contain positional parameters ($1, $2, etc.) instead
of named parameters. For example:

```go
type ProductDaoS struct {
	FindById             func(ctx context.Context, q proteus.ContextQuerier, id int) (Product, error)                      `proq:"select * from Product where id = :$1:"`
	Update               func(ctx context.Context, e proteus.ContextExecutor, p Product) (int64, error)                    `proq:"update Product set name = :$1.Name:, cost = :$1.Cost: where id = :$1.Id:"`
	FindByNameAndCost    func(ctx context.Context, q proteus.ContextQuerier, name string, cost float64) ([]Product, error) `proq:"select * from Product where name=:$1: and cost=:$2:"`
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
		GetF   func(ctx context.Context, q proteus.ContextQuerier, id string) (f, error)                `proq:"q:q1" prop:"id"`
		Update func(ctx context.Context, e proteus.ContextExecutor, id string, x string) (int64, error) `proq:"q:q2" prop:"id,x"`
	}
	
	sImpl := s{}
	err := proteus.ShouldBuild(context.Background(), &sImpl, proteus.Sqlite, m)
```

Out of the box, you can use either a `map[string]string` or a unix properties file to store your queries. In order
to use a `map[string]string`, cast your map to `proteus.MapMapper` (or just declare your variable to be of type `proteus.MapMapper`). To use a properties file, call the method 
`proteus.PropFileToQueryMapper` with the name of the property file that contains your queries.

## Generating function variables

Some people don't want to use structs and struct tags to implement their SQL mapping layer. Starting with version 0.11.0, Proteus can also generate functions that aren't fields in a struct.

First, create an instance of a `proteus.Builder`. The factory function takes a `proteus.Adapter` and zero or more `proteus.QueryMapper` instances:

```go
    b := NewBuilder(Postgres)
```

Next, declare a function variable with the signature you want. The parameters for the function variables follow the same rules as the function fields: 

```go
    var f func(c context.Context, e ContextExecutor, name string, age int) (int64, error)
    var g func(c context.Context, q ContextQuerier, id int) (*Person, error)
```

Then call the `BuildFunction` method on your `proteus.Builder` instance, passing in a pointer to your function variable, the SQL query (or a query mapper reference),
and the parameter names as a string slice:

```go
    err := b.BuildFunction(ctx, &f, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", []string{"name", "age"})
    if err != nil {
        t.Fatalf("build function failed: %v", err)
    }
    
    err = b.BuildFunction(ctx, &g, "SELECT * FROM PERSON WHERE id = :id:", []string{"id"})
    if err != nil {
        t.Fatalf("build function 2 failed: %v", err)
    }
```

Finally, call your functions, to run your SQL queries:

```go
    db := setupDbPostgres()
    defer db.Close()
    ctx := context.Background()
    
    rows, err := f(ctx, db, "Fred", 20)
    if err != nil {
        t.Fatalf("create failed: %v", err)
    }
    fmt.Println(rows) // prints 1
    
    p, err := g(ctx, db, 1)
    if err != nil {
        t.Fatalf("get failed: %v", err)
    }
    fmt.Println(p) // prints {1, Fred, 20}
```

## Ad-hoc database queries

While Proteus is focused on type safety, sometimes you just want to run a query without associating it with a function. 
Starting with version 0.11.0, Proteus allows you to run ad-hoc database queries.

First, create an instance of a `proteus.Builder`. The factory function takes a `proteus.Adapter` and zero or more `proteus.QueryMapper` instances:

```go
    b := NewBuilder(Postgres)
```

Next, run your query by passing it to the `Exec`, `ExecResult`, or `Query` methods on `proteus.Builder`. 

`Exec` expects a `context.Context`, a `proteus.ContextExecutor`, the query, and a 
map of `string` to `interface{}`, where the keys are the parameter names and the values are the parameter values. It returns an int64 with the number of rows modified and
and error. 	

`ExecResult` expects a `context.Context`, a `proteus.ContextExecutor`, the query, and a 
map of `string` to `interface{}`, where the keys are the parameter names and the values are the parameter values. It returns a `sql.Result` with the number of rows modified and
and error. 	

`Query` expected a `context.Context`, a `proteus.ContextQuerier`, the query, 
a map of `string` to `interface{}`, where the keys are the parameter names and the values are the parameter values, and a pointer to the value that
should be populated by the query. The method returns an error.

```go
    db := setupDbPostgres()
    defer db.Close()
    ctx := context.Background()

    rows, err := b.Exec(c, db, "INSERT INTO PERSON(name, age) VALUES(:name:, :age:)", map[string]interface{}{"name": "Fred", "age": 20})
    if err != nil {
        t.Fatalf("create failed: %v", err)
    } 
    fmt.Println(rows) // prints 1

    var p *Person
    err = b.Query(c, db, "SELECT * FROM PERSON WHERE id = :id:", map[string]interface{}{"id": 1}, &p)
    if err != nil {
        t.Fatalf("get failed: %v", err)
    }
    fmt.Println(*p) // prints {1, Fred, 20}
```

Ad-hoc queries support all of the functionality of Proteus except for type safety. You can reference queries in `proteus.QueryMapper` instances, build out dynamic
`in` clauses, extract values from `struct` instances, and map to a struct with `prof` tags on its fields.  

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
	FindById             func(ctx context.Context, q proteus.ContextQuerier, id int) (Product, error)                       `proq:"select * from Product where id = :id:" prop:"id"`
	Update               func(ctx context.Context, e proteus.ContextExecutor, p Product) (int64, error)                     `proq:"update Product set name = :p.Name:, cost = :p.Cost: where id = :p.Id:" prop:"p"`
	FindByNameAndCost    func(ctx context.Context, q proteus.ContextQuerier, name string, cost float64) ([]Product, error)  `proq:"select * from Product where name=:name: and cost=:cost:" prop:"name,cost"`
}

type ProductDao interface {
    FindById(context.Context, proteus.ContextQuerier, int) (Product, error)
    Update(context.Context, proteus.ContextExecutor, Product) (int64, error)
    FindByNameAndCost(context.Context, proteus.ContextQuerier, string, float64) ([]Product, error)
}

type productWrapper struct {
    ProductDaoS
}

func (pw productWrapper) FindById(ctx context.Context, q proteus.ContextQuerier, id int) (Product, error) {
    return pw.ProductDaoS.FindById(ctx, q, id)
}

func (pw productWrapper) Update(ctx context.Context, e proteus.ContextExecutor, p Product) (int64, error) {
    return pw.ProductDaoS.Update(ctx, e, p)
}

func (pw productWrapper) FindByNameAndCost(ctx context.Context, q proteus.ContextQuerier, n string, c float64) ([]Product, error) {
    return pw.ProductDaoS.FindByNameAndCost(ctx, q, n, c)
}

func NewProductDao(ctx context.Context) ProductDao {
    p := ProductDaoS{}
    err := proteus.ShouldBuild(ctx, &p,proteus.Sqlite)
    if err != nil {
        panic(err)
    }
    return productWrapper{p}
}
```

A future version of Proteus may include a tool that can be used with go generate to automatically create the wrapper and interface.

2\. Why do I have to specify the parameter names with a struct tag?

This is another limitation of go's reflection API. The names of parameters are not available at runtime to be inspected,
and must be supplied by another way in order to be referenced in a query. If you do not want to use a prop struct tag, you
can use positional parameters ($1, $2, etc.) instead.

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

- Build `in` clauses using a field from a slice of struct or map

- Generate batch `values` clauses using a slice of struct or map

- more expansive performance measurement support and per-request logging control

- go generate tool to create a wrapper struct and interface for a Proteus DAO

- API implementations for nonSQL data stores and HTTP requests

- Additional struct tags to generate basic CRUD queries automatically. This is about as far as I'd go in implementing
ORM-like features. The goal for Proteus is to be an adapter between a data source and business logic, not to define your
entire programming model.
