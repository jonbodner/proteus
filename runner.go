package proteus

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"database/sql"

	"github.com/jonbodner/proteus/logger"
	"github.com/jonbodner/proteus/mapper"
)

func buildQueryArgs(c context.Context, funcArgs []reflect.Value, paramOrder []paramInfo) ([]interface{}, error) {

	//walk through the rest of the input parameters and build a slice for args
	var out []interface{}
	for _, v := range paramOrder {
		val, err := mapper.Extract(c, funcArgs[v.posInParams].Interface(), strings.Split(v.name, "."))
		if err != nil {
			return nil, err
		}
		if v.isSlice {
			curSlice := reflect.ValueOf(val)
			for i := 0; i < curSlice.Len(); i++ {
				out = append(out, curSlice.Index(i).Interface())
			}
		} else {
			out = append(out, val)
		}
	}

	return out, nil
}

var (
	errType = reflect.TypeOf((*error)(nil)).Elem()
	errZero = reflect.Zero(errType)
	zero    = reflect.ValueOf(int64(0))
)

func makeContextExecutorImplementation(c context.Context, funcType reflect.Type, query queryHolder, paramOrder []paramInfo) func(args []reflect.Value) []reflect.Value {
	buildRetVals := makeExecutorReturnVals(funcType)
	return func(args []reflect.Value) []reflect.Value {

		executor := args[1].Interface().(ContextExecutor)
		ctx := args[0].Interface().(context.Context)

		// if the passed-in context doesn't contain any logging settings, use the one that were supplied
		// at build time
		if _, ok := logger.LevelFromContext(ctx); !ok {
			if defaultLevel, ok := logger.LevelFromContext(c); ok {
				ctx = logger.WithLevel(ctx, defaultLevel)
			}
		}
		// copy over any logging values that were supplied at build time
		if defaultVals, ok := logger.ValuesFromContext(c); ok {
			ctx = logger.WithValues(ctx, defaultVals...)
		}

		var result sql.Result
		finalQuery, err := query.finalize(ctx, args)

		if err != nil {
			return buildRetVals(result, err)
		}

		queryArgs, err := buildQueryArgs(ctx, args, paramOrder)

		if err != nil {
			return buildRetVals(result, err)
		}

		logger.Log(ctx, logger.DEBUG, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
		result, err = executor.ExecContext(ctx, finalQuery, queryArgs...)

		return buildRetVals(result, err)
	}
}

func makeExecutorImplementation(c context.Context, funcType reflect.Type, query queryHolder, paramOrder []paramInfo) func(args []reflect.Value) []reflect.Value {
	buildRetVals := makeExecutorReturnVals(funcType)
	return func(args []reflect.Value) []reflect.Value {

		executor := args[0].Interface().(Executor)

		var result sql.Result
		finalQuery, err := query.finalize(c, args)

		if err != nil {
			return buildRetVals(result, err)
		}

		queryArgs, err := buildQueryArgs(c, args, paramOrder)

		if err != nil {
			return buildRetVals(result, err)
		}

		logger.Log(c, logger.DEBUG, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
		result, err = executor.Exec(finalQuery, queryArgs...)

		return buildRetVals(result, err)
	}
}

func makeExecutorReturnVals(funcType reflect.Type) func(sql.Result, error) []reflect.Value {
	numOut := funcType.NumOut()

	//handle the 0,1,2 out parameter cases
	if numOut == 0 {
		return func(sql.Result, error) []reflect.Value {
			return []reflect.Value{}
		}
	}

	sType := funcType.Out(0)
	if numOut == 1 {
		return func(result sql.Result, err error) []reflect.Value {
			if err != nil {
				return []reflect.Value{zero}
			}
			val, err := result.RowsAffected()
			if err != nil {
				return []reflect.Value{zero}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType)}
		}
	}
	if numOut == 2 {
		return func(result sql.Result, err error) []reflect.Value {
			eType := funcType.Out(1)
			if err != nil {
				return []reflect.Value{zero, reflect.ValueOf(err).Convert(eType)}
			}
			val, err := result.RowsAffected()
			if err != nil {
				return []reflect.Value{zero, reflect.ValueOf(err).Convert(eType)}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType), errZero}
		}
	}

	// impossible case since validation should happen first, but be safe
	return func(result sql.Result, err error) []reflect.Value {
		return []reflect.Value{zero, reflect.ValueOf(errors.New("should never get here"))}
	}
}

func makeContextQuerierImplementation(c context.Context, funcType reflect.Type, query queryHolder, paramOrder []paramInfo) (func(args []reflect.Value) []reflect.Value, error) {
	numOut := funcType.NumOut()
	var builder mapper.Builder
	var err error
	if numOut > 0 {
		builder, err = mapper.MakeBuilder(c, funcType.Out(0))
		if err != nil {
			return nil, err
		}
	}
	buildRetVals := makeQuerierReturnVals(c, funcType, builder)
	return func(args []reflect.Value) []reflect.Value {
		querier := args[1].Interface().(ContextQuerier)
		ctx := args[0].Interface().(context.Context)

		// if the passed-in context doesn't contain any logging settings, use the one that were supplied
		// at build time
		if _, ok := logger.LevelFromContext(ctx); !ok {
			if defaultLevel, ok := logger.LevelFromContext(c); ok {
				ctx = logger.WithLevel(ctx, defaultLevel)
			}
		}
		// copy over any logging values that were supplied at build time
		if defaultVals, ok := logger.ValuesFromContext(c); ok {
			ctx = logger.WithValues(ctx, defaultVals...)
		}

		var rows *sql.Rows
		finalQuery, err := query.finalize(ctx, args)
		if err != nil {
			return buildRetVals(rows, err)
		}

		queryArgs, err := buildQueryArgs(ctx, args, paramOrder)
		if err != nil {
			return buildRetVals(rows, err)
		}

		logger.Log(ctx, logger.DEBUG, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
		rows, err = querier.QueryContext(ctx, finalQuery, queryArgs...)

		return buildRetVals(rows, err)
	}, nil
}

func makeQuerierImplementation(c context.Context, funcType reflect.Type, query queryHolder, paramOrder []paramInfo) (func(args []reflect.Value) []reflect.Value, error) {
	numOut := funcType.NumOut()
	var builder mapper.Builder
	var err error
	if numOut > 0 {
		builder, err = mapper.MakeBuilder(c, funcType.Out(0))
		if err != nil {
			return nil, err
		}
	}
	buildRetVals := makeQuerierReturnVals(c, funcType, builder)
	return func(args []reflect.Value) []reflect.Value {
		querier := args[0].Interface().(Querier)

		var rows *sql.Rows
		finalQuery, err := query.finalize(c, args)
		if err != nil {
			return buildRetVals(rows, err)
		}

		queryArgs, err := buildQueryArgs(c, args, paramOrder)
		if err != nil {
			return buildRetVals(rows, err)
		}

		logger.Log(c, logger.DEBUG, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
		rows, err = querier.Query(finalQuery, queryArgs...)

		return buildRetVals(rows, err)
	}, nil
}

func makeQuerierReturnVals(c context.Context, funcType reflect.Type, builder mapper.Builder) func(*sql.Rows, error) []reflect.Value {
	numOut := funcType.NumOut()

	//handle the 0,1,2 out parameter cases
	if numOut == 0 {
		return func(*sql.Rows, error) []reflect.Value {
			return []reflect.Value{}
		}
	}

	sType := funcType.Out(0)
	qZero := reflect.Zero(sType)
	if numOut == 1 {
		return func(rows *sql.Rows, err error) []reflect.Value {
			if err != nil {
				return []reflect.Value{qZero}
			}
			// handle mapping
			val, err := handleMapping(c, sType, rows, builder)
			if err != nil {
				return []reflect.Value{qZero}
			}
			if val == nil {
				return []reflect.Value{qZero}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType)}
		}
	}
	if numOut == 2 {
		return func(rows *sql.Rows, err error) []reflect.Value {
			eType := funcType.Out(1)
			if err != nil {
				return []reflect.Value{qZero, reflect.ValueOf(err).Convert(eType)}
			}
			// handle mapping
			val, err := handleMapping(c, sType, rows, builder)
			var eVal reflect.Value
			if err == nil {
				eVal = errZero
			} else {
				eVal = reflect.ValueOf(err).Convert(eType)
			}
			if val == nil {
				return []reflect.Value{qZero, eVal}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType), eVal}
		}
	}

	// impossible case since validation should happen first, but be safe
	return func(*sql.Rows, error) []reflect.Value {
		return []reflect.Value{qZero, reflect.ValueOf(errors.New("should never get here!"))}
	}
}

func handleMapping(c context.Context, sType reflect.Type, rows *sql.Rows, builder mapper.Builder) (interface{}, error) {
	var val interface{}
	var err error
	if sType.Kind() == reflect.Slice {
		s := reflect.MakeSlice(sType, 0, 0)
		var result interface{}
		for {
			result, err = mapRows(c, rows, builder)
			if err != nil {
				return nil, err
			}
			if result == nil {
				break
			}
			s = reflect.Append(s, reflect.ValueOf(result))
		}
		val = s.Interface()
	} else {
		val, err = mapRows(c, rows, builder)
	}
	rows.Close()
	return val, err
}

// Map takes the next value from Rows and uses it to create a new instance of the specified type
// If the type is a primitive and there are more than 1 values in the current row, only the first value is used.
// If the type is a map of string to interface, then the column names are the keys in the map and the values are assigned
// If the type is a struct that has prop tags on its fields, then any matching tags will be associated with values with the associate columns
// Any non-associated values will be set to the zero value
// If any columns cannot be assigned to any types, then an error is returned
// If next returns false, then nil is returned for both the interface and the error
// If an error occurs while processing the current row, nil is returned for the interface and the error is non-nil
// If a value is successfully extracted from the current row, the instance is returned and the error is nil
func mapRows(c context.Context, rows *sql.Rows, builder mapper.Builder) (interface{}, error) {
	//fmt.Println(sType)
	if rows == nil {
		return nil, errors.New("rows must be non-nil")
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	if len(cols) == 0 {
		return nil, errors.New("No values returned from query")
	}

	vals := make([]interface{}, len(cols))
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(vals); i++ {
		newVal := reflect.New(colTypes[i].ScanType()).Interface()
		vals[i] = &newVal
	}

	err = rows.Scan(vals...)
	if err != nil {
		logger.Log(c, logger.WARN, "scan failed")
		return nil, err
	}

	return builder(cols, vals)
}
