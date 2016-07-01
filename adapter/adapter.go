package adapter

import (
	"fmt"
	"github.com/jonbodner/proteus/api"
	"database/sql"
)

func MySQL(pos int) string {
	return "?"
}

func Sqlite(pos int) string {
	return "?"
}

func Postgres(pos int) string {
	return fmt.Sprintf("$%d", pos)
}

func Oracle(pos int) string {
	return fmt.Sprintf(":%d", pos)
}

// Sql returns a wrapper that adapts several standard go sql types to work with Proteus.
func Sql(sqle sqlExecutor) api.Executor {
	return wrapper{sqle}
}

type wrapper struct {
	sqlExecutor
}

func (w wrapper) Exec(query string, args ...interface{}) (api.Result, error) {
	return w.sqlExecutor.Exec(query, args...)
}

func (w wrapper) Query(query string, args ...interface{}) (api.Rows, error) {
	return w.sqlExecutor.Query(query, args...)
}

// sqlExecutor matches the interface provided by several types in the standard go sql package.
type sqlExecutor interface {
	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(query string, args ...interface{}) (*sql.Rows, error)
}
