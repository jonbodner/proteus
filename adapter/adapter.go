package adapter

import "fmt"

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
