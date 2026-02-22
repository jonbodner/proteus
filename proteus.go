package proteus

import (
	"context"
	"log/slog"
	"reflect"

	"fmt"
	"strings"

	"github.com/jonbodner/multierr"
	"github.com/jonbodner/stackerr"
)

/*
struct tags:
proq - SQL code to execute
	If the first parameter is an Executor, returns new id (if sql.Result has a non-zero value for LastInsertId) or number of rows changed.
	If the first parameter is a Querier, returns single entity or list of entities.
prop - The parameter names. Should be in order for the function parameters (skipping over the first Executor parameter)
prof - The fields on the dto that are mapped to select parameters in a query
next:
Put a reference to a public string instead of the query and that public string will be used as the query
later:
struct tags to mark as CRUD operations

converting name parameterized queries to positional queries
1. build a map of prop entry->position in parameter list
2. For each : in the input query
3. Find it
4. Find the end of the term (whitespace, comma, or end of string)
5. Create a map of querypos -> struct {parameter position (int), field path (string), isSlice (bool)}
6. If is slice, replace with ??
7. Otherwise, replace with ?
8. Capture the new string and the map in the closure
9. On run, do the replacements directly
10. If there are slices (??), replace with series of ? separated by commas, blow out slice in args slice

Return type is 0, 1, or 2 values
If zero, suppress all errors and return values (not great)
If 1:
For exec, return LastInsertId if !=0, otherwise return # of row changes (int in either case)
For query, if return type is []Entity, map to the entities.
For query, if return type is Entity and there are > 1 value, return the first. If there are zero, return the zero value of the Entity.
If 2:
Same as 1, 2nd parameter is error
Exception: if return type is entity and there are 0 values or > 1 value, return error indicating this.

On mapping for query, any unmappable parameters are ignored
If the entity is a primitive, then the first value returned for a row must be of that type, or it's an error. All other values for that row will be ignored.

*/

type Error struct {
	FuncName      string
	FieldOrder    int
	OriginalError error
}

func (pe Error) Error() string {
	return fmt.Sprintf("error in field #%d (%s): %v", pe.FieldOrder, pe.FuncName, pe.OriginalError)
}

func (pe Error) Unwrap() error {
	return pe.OriginalError
}

// ShouldBuild works like Build, with two differences:
//
// 1. It will not populate any function fields if there are errors.
//
// 2. The context passed in to ShouldBuild can be used to specify the logging level used during ShouldBuild and
// when the generated functions are invoked. This overrides any logging level specified using the SetLogLevel
// function.
func ShouldBuild(ctx context.Context, dao interface{}, paramAdapter ParamAdapter, mappers ...QueryMapper) error {
	daoPointerType := reflect.TypeOf(dao)
	//must be a pointer to struct
	if daoPointerType.Kind() != reflect.Pointer {
		return stackerr.New("not a pointer")
	}
	daoType := daoPointerType.Elem()
	//if not a struct, error out
	if daoType.Kind() != reflect.Struct {
		return stackerr.New("not a pointer to struct")
	}
	var out error
	funcs := make([]reflect.Value, daoType.NumField())
	daoPointerValue := reflect.ValueOf(dao)
	daoValue := reflect.Indirect(daoPointerValue)
	//for each field in ProductDao that is of type func and has a proteus struct tag, assign it a func
	for i := 0; i < daoType.NumField(); i++ {
		curField := daoType.Field(i)

		//Implement embedded fields -- if we have a field of type struct and it's anonymous,
		//recurse
		if curField.Type.Kind() == reflect.Struct && curField.Anonymous {
			pv := reflect.New(curField.Type)
			embeddedErrs := ShouldBuild(ctx, pv.Interface(), paramAdapter, mappers...)
			if embeddedErrs != nil {
				out = multierr.Append(out, embeddedErrs)
			} else {
				funcs[i] = pv.Elem()
			}
			continue
		}

		query, ok := curField.Tag.Lookup("proq")
		if curField.Type.Kind() != reflect.Func || !ok {
			continue
		}
		funcType := curField.Type

		//validate to make sure that the function matches what we expect
		hasCtx, err := validateFunction(funcType)
		if err != nil {
			out = multierr.Append(out, Error{FuncName: curField.Name, FieldOrder: i, OriginalError: err})
			continue
		}

		paramOrder := curField.Tag.Get("prop")
		var nameOrderMap map[string]int
		startPos := 1
		if hasCtx {
			startPos = 2
		}
		if len(paramOrder) == 0 {
			nameOrderMap = buildDummyParameters(funcType.NumIn(), startPos)
		} else {
			nameOrderMap = buildNameOrderMap(paramOrder, startPos)
		}

		//check to see if the query is in a QueryMapper
		query, err = lookupQuery(query, mappers)
		if err != nil {
			out = multierr.Append(out, Error{FuncName: curField.Name, FieldOrder: i, OriginalError: err})
			continue
		}

		implementation, err := makeImplementation(ctx, funcType, query, paramAdapter, nameOrderMap)
		if err != nil {
			out = multierr.Append(out, Error{FuncName: curField.Name, FieldOrder: i, OriginalError: err})
			continue
		}
		funcs[i] = reflect.MakeFunc(funcType, implementation)
	}
	if out == nil {
		for i, v := range funcs {
			if v.IsValid() {
				fieldValue := daoValue.Field(i)
				fieldValue.Set(v)
			}
		}
	}
	return out
}

// Build is the main entry point into Proteus. It takes in a pointer to a DAO struct to populate,
// a proteus.ParamAdapter, and zero or more proteus.QueryMapper instances.
//
// As of version v0.12.0, all errors found during building will be reported back. Also, prefer using
// proteus.ShouldBuild over proteus.Build.
func Build(dao interface{}, paramAdapter ParamAdapter, mappers ...QueryMapper) error {
	ctx := context.Background()
	daoPointerType := reflect.TypeOf(dao)
	//must be a pointer to struct
	if daoPointerType.Kind() != reflect.Pointer {
		return stackerr.New("not a pointer")
	}
	daoType := daoPointerType.Elem()
	//if not a struct, error out
	if daoType.Kind() != reflect.Struct {
		return stackerr.New("not a pointer to struct")
	}
	daoPointerValue := reflect.ValueOf(dao)
	daoValue := reflect.Indirect(daoPointerValue)
	var outErr error
	//for each field in ProductDao that is of type func and has a proteus struct tag, assign it a func
	for i := 0; i < daoType.NumField(); i++ {
		curField := daoType.Field(i)

		//Implement embedded fields -- if we have a field of type struct and it's anonymous,
		//recurse
		if curField.Type.Kind() == reflect.Struct && curField.Anonymous {
			pv := reflect.New(curField.Type)
			err := Build(pv.Interface(), paramAdapter, mappers...)
			if err != nil {
				outErr = multierr.Append(outErr, err)
			} else {
				daoValue.Field(i).Set(pv.Elem())
			}
			continue
		}

		query, ok := curField.Tag.Lookup("proq")
		if curField.Type.Kind() != reflect.Func || !ok {
			continue
		}
		funcType := curField.Type

		//validate to make sure that the function matches what we expect
		hasCtx, err := validateFunction(funcType)
		if err != nil {
			slog.Log(ctx, slog.LevelWarn, fmt.Sprintln("skipping function", curField.Name, "due to error:", err.Error()))
			outErr = multierr.Append(outErr, err)
			continue
		}

		paramOrder := curField.Tag.Get("prop")
		var nameOrderMap map[string]int
		startPos := 1
		if hasCtx {
			startPos = 2
		}
		if len(paramOrder) == 0 {
			nameOrderMap = buildDummyParameters(funcType.NumIn(), startPos)
		} else {
			nameOrderMap = buildNameOrderMap(paramOrder, startPos)
		}

		//check to see if the query is in a QueryMapper
		query, err = lookupQuery(query, mappers)
		if err != nil {
			slog.Log(ctx, slog.LevelWarn, fmt.Sprintln("skipping function", curField.Name, "due to error:", err.Error()))
			outErr = multierr.Append(outErr, err)
			continue
		}

		implementation, err := makeImplementation(ctx, funcType, query, paramAdapter, nameOrderMap)
		if err != nil {
			slog.Log(ctx, slog.LevelWarn, fmt.Sprintln("skipping function", curField.Name, "due to error:", err.Error()))
			outErr = multierr.Append(outErr, err)
			continue
		}
		fieldValue := daoValue.Field(i)
		fieldValue.Set(reflect.MakeFunc(funcType, implementation))
	}
	if outErr != nil {
		return outErr
	}
	return nil
}

var (
	exType      = reflect.TypeFor[Executor]()
	qType       = reflect.TypeFor[Querier]()
	contextType = reflect.TypeFor[context.Context]()
	conExType   = reflect.TypeFor[ContextExecutor]()
	conQType    = reflect.TypeFor[ContextQuerier]()
	errType     = reflect.TypeFor[error]()
)

func validateFunction(funcType reflect.Type) (bool, error) {
	//the first parameter is Executor
	if funcType.NumIn() == 0 {
		return false, stackerr.New("need to supply an Executor or Querier parameter")
	}
	var isExec bool
	var hasContext bool
	switch fType := funcType.In(0); {
	case fType.Implements(contextType):
		hasContext = true
	case fType.Implements(exType):
		isExec = true
	case fType.Implements(qType):
		//do nothing isExec is false
	default:
		return false, stackerr.New("first parameter must be of type context.Context, Executor, or Querier")
	}
	start := 1
	if hasContext {
		start = 2
		switch fType := funcType.In(1); {
		case fType.Implements(conExType):
			isExec = true
		case fType.Implements(conQType):
			//do nothing isExec is false
		default:
			return false, stackerr.New("first parameter must be of type context.Context, Executor, or Querier")
		}
	}
	//no in parameter can be a channel
	for i := start; i < funcType.NumIn(); i++ {
		if funcType.In(i).Kind() == reflect.Chan {
			return false, stackerr.New("no input parameter can be a channel")
		}
	}

	//has 0, 1, or 2 return values
	if funcType.NumOut() > 2 {
		return false, stackerr.New("must return 0, 1, or 2 values")
	}

	//if 2 return values, second is error
	if funcType.NumOut() == 2 {
		if !funcType.Out(1).Implements(errType) {
			return false, stackerr.New("2nd output parameter must be of type error")
		}
	}

	//if 1 or 2, the 1st param is not a channel (handle map, I guess)
	if funcType.NumOut() > 0 {
		if funcType.Out(0).Kind() == reflect.Chan {
			return false, stackerr.New("1st output parameter cannot be a channel")
		}
		if isExec && funcType.Out(0).Kind() != reflect.Int64 &&
			funcType.Out(0) != sqlResultType {
			return false, stackerr.New("the 1st output parameter of an Executor must be int64 or sql.Result")
		}

		//sql.Result only useful with executor.
		if !isExec && funcType.Out(0) == sqlResultType {
			return false, stackerr.New("output parameters of type sql.Result must be combined with Executor")
		}
	}
	return hasContext, nil
}

func makeImplementation(ctx context.Context, funcType reflect.Type, query string, paramAdapter ParamAdapter, nameOrderMap map[string]int) (func([]reflect.Value) []reflect.Value, error) {
	fixedQuery, paramOrder, err := buildFixedQueryAndParamOrder(ctx, query, nameOrderMap, funcType, paramAdapter)
	if err != nil {
		return nil, err
	}

	switch fType := funcType.In(0); {
	case fType.Implements(contextType):
		switch fType2 := funcType.In(1); {
		case fType2.Implements(conExType):
			return makeContextExecutorImplementation(ctx, funcType, fixedQuery, paramOrder), nil
		case fType2.Implements(conQType):
			return makeContextQuerierImplementation(ctx, funcType, fixedQuery, paramOrder)
		}
	case fType.Implements(exType):
		return makeExecutorImplementation(ctx, funcType, fixedQuery, paramOrder), nil
	case fType.Implements(qType):
		return makeQuerierImplementation(ctx, funcType, fixedQuery, paramOrder)
	}
	//this should be impossible, since we already validated that the first parameter is either an executor or a querier
	return nil, stackerr.New("first parameter must be of type Executor or Querier")
}

func lookupQuery(query string, mappers []QueryMapper) (string, error) {
	if !strings.HasPrefix(query, "q:") {
		return query, nil
	}
	name := query[2:]
	for _, v := range mappers {
		if q := v.Map(name); q != "" {
			return q, nil
		}
	}
	return "", stackerr.Errorf("no query found for name %s", name)
}
