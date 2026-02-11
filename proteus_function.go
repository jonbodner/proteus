package proteus

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/jonbodner/proteus/mapper"
	"github.com/jonbodner/stackerr"
)

type Builder struct {
	adapter ParamAdapter
	mappers []QueryMapper
}

func NewBuilder(adapter ParamAdapter, mappers ...QueryMapper) Builder {
	return Builder{
		adapter: adapter,
		mappers: mappers,
	}
}

func (fb Builder) BuildFunction(ctx context.Context, f interface{}, query string, names []string) error {
	// make sure that f is of the right type (pointer to function)
	funcPointerType := reflect.TypeOf(f)
	//must be a pointer to func
	if funcPointerType.Kind() != reflect.Ptr {
		return stackerr.New("not a pointer")
	}
	funcType := funcPointerType.Elem()
	//if not a func, error out
	if funcType.Kind() != reflect.Func {
		return stackerr.New("not a pointer to func")
	}

	//validate to make sure that the function matches what we expect
	hasCtx, err := validateFunction(funcType)
	if err != nil {
		return err
	}

	var nameOrderMap map[string]int
	startPos := 1
	if hasCtx {
		startPos = 2
	}
	if len(names) == 0 {
		nameOrderMap = buildDummyParameters(funcType.NumIn(), startPos)
	} else {
		nameOrderMap = buildFuncNameOrderMap(names, startPos)
	}

	//check to see if the query is in a QueryMapper
	query, err = lookupQuery(query, fb.mappers)
	if err != nil {
		return err
	}

	implementation, err := makeImplementation(ctx, funcType, query, fb.adapter, nameOrderMap)
	if err != nil {
		return err
	}

	reflect.ValueOf(f).Elem().Set(reflect.MakeFunc(funcType, implementation))

	return nil
}

func buildFuncNameOrderMap(names []string, startPos int) map[string]int {
	out := map[string]int{}
	for k, v := range names {
		out[strings.TrimSpace(v)] = k + startPos
	}
	return out
}

type sliceTypes []reflect.Type

func (st sliceTypes) In(i int) reflect.Type {
	return st[i]
}

func (fb Builder) Exec(ctx context.Context, e ContextExecutor, query string, params map[string]interface{}) (int64, error) {
	result, err := fb.ExecResult(ctx, e, query, params)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	return count, err
}

func (fb Builder) ExecResult(ctx context.Context, e ContextExecutor, query string, params map[string]interface{}) (sql.Result, error) {
	finalQuery, queryArgs, err := fb.setupDynamicQueries(ctx, query, params)
	if err != nil {
		return nil, err
	}

	slog.Log(ctx, slog.LevelDebug, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
	result, err := e.ExecContext(ctx, finalQuery, queryArgs...)
	return result, err
}

func (fb Builder) Query(ctx context.Context, q ContextQuerier, query string, params map[string]interface{}, output interface{}) error {
	// make sure that output is a pointer to something
	outputPointerType := reflect.TypeOf(output)
	if outputPointerType.Kind() != reflect.Ptr {
		return stackerr.New("not a pointer")
	}

	finalQuery, queryArgs, err := fb.setupDynamicQueries(ctx, query, params)
	if err != nil {
		return err
	}

	slog.Log(ctx, slog.LevelDebug, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
	rows, err := q.QueryContext(ctx, finalQuery, queryArgs...)
	if err != nil {
		return err
	}

	sType := outputPointerType.Elem()
	qZero := reflect.Zero(sType)
	builder, err := mapper.MakeBuilder(ctx, sType)
	if err != nil {
		return err
	}

	val, err := handleMapping(ctx, sType, rows, builder)
	if err != nil {
		return err
	}
	outputValue := reflect.ValueOf(output).Elem()
	if val == nil {
		outputValue.Set(qZero)
		return nil
	}
	outputValue.Set(reflect.ValueOf(val).Convert(sType))

	return nil
}

func (fb Builder) setupDynamicQueries(ctx context.Context, query string, paramsAndNames map[string]interface{}) (string, []interface{}, error) {
	params := make([]interface{}, 0, len(paramsAndNames))
	names := make([]string, 0, len(paramsAndNames))
	for k, v := range paramsAndNames {
		params = append(params, v)
		names = append(names, k)
	}

	nameOrderMap := buildFuncNameOrderMap(names, 0)

	//check to see if the query is in a QueryMapper
	query, err := lookupQuery(query, fb.mappers)
	if err != nil {
		return "", nil, err
	}

	st := make(sliceTypes, 0, len(params))
	args := make([]reflect.Value, 0, len(params))
	for _, v := range params {
		st = append(st, reflect.TypeOf(v))
		args = append(args, reflect.ValueOf(v))
	}

	fixedQuery, paramOrder, err := buildFixedQueryAndParamOrder(ctx, query, nameOrderMap, st, fb.adapter)
	if err != nil {
		return "", nil, err
	}

	finalQuery, err := fixedQuery.finalize(ctx, args)
	if err != nil {
		return "", nil, err
	}

	queryArgs, err := buildQueryArgs(ctx, args, paramOrder)
	if err != nil {
		return "", nil, err
	}

	return finalQuery, queryArgs, nil
}
