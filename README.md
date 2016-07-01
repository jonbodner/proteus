# Proteus
A simple tool for generating an application's data access layer.

## Purpose

## Quick Start

## Struct Tags

## API

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

- API implementations for nonSQL data stores and HTTP requests

- Additional struct tags to generate basic CRUD queries automatically