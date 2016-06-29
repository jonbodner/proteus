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
	r, err := w.Tx.Exec(query, args...)
	return api.Result(r),err
}

func (w wrapper) Query(query string, args ...interface{}) (api.Rows, error) {
	r, err := w.Tx.Query(query, args...)
	return api.Rows(r), err
}
