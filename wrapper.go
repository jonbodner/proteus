package proteus

import (
	"database/sql"
)

// Wrapper returns a wrapper that adapts several standard go sql types to work with Proteus.
func Wrap(sqle Sql) Wrapper {
	return sqlWrapper{sqle}
}

type sqlWrapper struct {
	Sql
}

func (w sqlWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	return w.Sql.Exec(query, args...)
}

func (w sqlWrapper) Query(query string, args ...interface{}) (Rows, error) {
	return w.Sql.Query(query, args...)
}

// Sql matches the interface provided by several types in the standard go sql package.
type Sql interface {
	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
