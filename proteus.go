package proteus

import (
	"errors"
	"reflect"

	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
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

// Build is the main entry point into Proteus
func Build(dao interface{}, paramAdapter ParamAdapter, mappers ...QueryMapper) error {
	daoPointerType := reflect.TypeOf(dao)
	//must be a pointer to struct
	if daoPointerType.Kind() != reflect.Ptr {
		return errors.New("Not a pointer")
	}
	daoType := daoPointerType.Elem()
	//if not a struct, error out
	if daoType.Kind() != reflect.Struct {
		return errors.New("Not a pointer to struct")
	}
	daoPointerValue := reflect.ValueOf(dao)
	daoValue := reflect.Indirect(daoPointerValue)
	//for each field in ProductDao that is of type func and has a proteus struct tag, assign it a func
	for i := 0; i < daoType.NumField(); i++ {
		curField := daoType.Field(i)
		query, ok := curField.Tag.Lookup("proq")
		if curField.Type.Kind() != reflect.Func || !ok {
			continue
		}
		funcType := curField.Type

		//validate to make sure that the function matches what we expect
		err := validateFunction(funcType)
		if err != nil {
			log.Warnln("skipping function", curField.Name, "due to error:", err.Error())
			continue
		}

		paramOrder := curField.Tag.Get("prop")
		var nameOrderMap map[string]int
		if len(paramOrder) == 0 {
			nameOrderMap = buildDummyParameters(funcType.NumIn())
		} else {
			nameOrderMap = buildNameOrderMap(paramOrder)
		}

		//check to see if the query is in a QueryMapper
		query, err = lookupQuery(query, mappers)
		if err != nil {
			log.Warnln("skipping function", curField.Name, "due to error:", err.Error())
			continue
		}

		implementation, err := makeImplementation(funcType, query, paramAdapter, nameOrderMap)
		if err != nil {
			log.Warnln("skipping function", curField.Name, "due to error:", err.Error())
			continue
		}
		fieldValue := daoValue.Field(i)
		fieldValue.Set(reflect.MakeFunc(funcType, implementation))
	}
	return nil
}

var (
	exType = reflect.TypeOf((*Executor)(nil)).Elem()
	qType  = reflect.TypeOf((*Querier)(nil)).Elem()
)

func validateFunction(funcType reflect.Type) error {
	//first parameter is Executor
	if funcType.NumIn() == 0 {
		return errors.New("need to supply an Executor or Querier parameter")
	}
	var isExec bool
	switch fType := funcType.In(0); {
	case fType.Implements(exType):
		isExec = true
	case fType.Implements(qType):
		//do nothing isExec is false
	default:
		return errors.New("first parameter must be of type Executor or Querier")
	}
	//no in parameter can be a channel
	for i := 1; i < funcType.NumIn(); i++ {
		if funcType.In(i).Kind() == reflect.Chan {
			return errors.New("no input parameter can be a channel")
		}
	}

	//has 0, 1, or 2 return values
	if funcType.NumOut() > 2 {
		return errors.New("must return 0, 1, or 2 values")
	}

	//if 2 return values, second is error
	if funcType.NumOut() == 2 {
		errType := reflect.TypeOf((*error)(nil)).Elem()
		if !funcType.Out(1).Implements(errType) {
			return errors.New("2nd output parameter must be of type error")
		}
	}

	//if 1 or 2, 1st param is not a channel (handle map, I guess)
	if funcType.NumOut() > 0 {
		if funcType.Out(0).Kind() == reflect.Chan {
			return errors.New("1st output parameter cannot be a channel")
		}
		if isExec && funcType.Out(0).Kind() != reflect.Int64 {
			return errors.New("the 1st output parameter of an Executor must be int64")
		}
	}
	return nil
}

func makeImplementation(funcType reflect.Type, query string, paramAdapter ParamAdapter, nameOrderMap map[string]int) (func([]reflect.Value) []reflect.Value, error) {
	fixedQuery, paramOrder, err := buildFixedQueryAndParamOrder(query, nameOrderMap, funcType, paramAdapter)
	if err != nil {
		return nil, err
	}

	switch fType := funcType.In(0); {
	case fType.Implements(exType):
		return makeExecutorImplementation(funcType, fixedQuery, paramOrder), nil
	case fType.Implements(qType):
		return makeQuerierImplementation(funcType, fixedQuery, paramOrder)
	}
	//this should impossible, since we already validated that the first parameter is either an executor or a querier
	return nil, errors.New("first parameter must be of type Executor or Querier")
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
	return "", fmt.Errorf("no query found for name %s", name)
}
