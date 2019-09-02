package proteus

import (
	"context"
	"database/sql"
)

// Executor runs queries that modify the data store.
type Executor interface {
	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Querier runs queries that return Rows from the data store
type Querier interface {
	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type Wrapper interface {
	Executor
	Querier
}

// ParamAdapter maps to valid positional parameters in a DBMS.
// For example, MySQL uses ? for every parameter, while Postgres uses $NUM and Oracle uses :NUM
type ParamAdapter func(pos int) string

// QueryMapper maps from a query name to an actual query
// It is used to support the proq struct tag, when it contains q:name
type QueryMapper interface {
	// Maps the supplied name to a query string
	// returns an empty string if there is no query associated with the supplied name
	Map(name string) string
}

// ContextQuerier defines the interface of a type that runs a SQL query with a context
type ContextQuerier interface {
	// QueryContext executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// ContextExecutor defines the interface of a type that runs a SQL exec with a context
type ContextExecutor interface {
	// ExecContext executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type ContextWrapper interface {
	ContextQuerier
	ContextExecutor
}
