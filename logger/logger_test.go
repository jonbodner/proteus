package logger_test

import (
	"context"
	"testing"

	"github.com/jonbodner/proteus/logger"
)

func TestLogging(t *testing.T) {
	logger.Log(logger.WithLevel(context.Background(), logger.DEBUG), logger.DEBUG, "this is a message", []logger.Pair{
		{"Foo", "Bar"},
		{"int", 1},
		{"bool", true},
		{"float", 3.14},
		{"struct", struct {
			A int
			B string
		}{1, "he\"ll:{},o"}},
	}...)
}
