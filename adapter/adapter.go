package adapter

import (
	"fmt"
	"github.com/jonbodner/gdb/api"
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

func Sql(tx *sql.Tx) api.Executor {
	return wrapper{tx}
}

type wrapper struct {
	*sql.Tx
}

func (w wrapper) Exec(query string, args ...interface{}) (api.Result, error) {
	return w.Tx.Exec(query, args...)
}

func (w wrapper) Query(query string, args ...interface{}) (api.Rows, error) {
	return w.Tx.Query(query, args...)
}
