package proteus

import (
	"database/sql"
	"fmt"
)

type executorWrapper struct {
	Executor
	query string
	args  []interface{}
}

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (stubResult) RowsAffected() (int64, error) {
	return 0, nil
}

func (ew *executorWrapper) Exec(query string, args ...interface{}) (sql.Result, error) {
	ew.query = ew.query + query + ";"
	ew.args = append(ew.args, args...)
	return stubResult{}, nil
}

func Batch(ex Executor, f func(Executor) error) (int64, error) {
	ew := &executorWrapper{Executor: ex, query: ""}
	err := f(ew)
	if err != nil {
		return 0, err
	}
	fmt.Println(ew.query)
	fmt.Println(ew.args)
	result, err := ex.Exec(ew.query, ew.args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
