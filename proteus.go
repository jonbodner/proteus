package proteus

import (
	"errors"
	"reflect"

	log "github.com/Sirupsen/logrus"
	"github.com/jonbodner/proteus/api"
)

/*
struct tags:
proq - Query run by Executor.Query. Returns single entity or list of entities
proe - Query run by Executor.Exec. Returns new id (if sql.Result has a non-zero value for LastInsertId) or number of rows changed
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
func Build(dao interface{}, pa api.ParamAdapter) error {
	t := reflect.TypeOf(dao)
	//must be a pointer to struct
	if t.Kind() != reflect.Ptr {
		return errors.New("Not a pointer")
	}
	t2 := t.Elem()
	//if not a struct, error out
	if t2.Kind() != reflect.Struct {
		return errors.New("Not a pointer to struct")
	}
	svp := reflect.ValueOf(dao)
	sv := reflect.Indirect(svp)
	//for each field in ProductDao that is of type func and has a proteus struct tag, assign it a func
	for i := 0; i < t2.NumField(); i++ {
		curField := t2.Field(i)
		proq := curField.Tag.Get("proq")
		proe := curField.Tag.Get("proe")
		prop := curField.Tag.Get("prop")
		if curField.Type.Kind() == reflect.Func && (proq != "" || proe != "") {
			funcType := curField.Type
			//validate to make sure that the function matches what we expect
			err := validateFunction(funcType, proe != "")
			if err != nil {
				log.Warnln("skipping function", curField.Name, "due to error:", err.Error())
				continue
			}

			paramMap := buildParamMap(prop)

			var query string
			var curFunc funcBuilder
			if proq != "" {
				query = proq
				curFunc = buildQuery
			} else {
				query = proe
				curFunc = buildExec
			}

			positionalQuery, qps, err := convertToPositionalParameters(query, paramMap, funcType, pa)
			if err != nil {
				return err
			}

			toFunc, err := curFunc(funcType, qps, positionalQuery)

			if err != nil {
				log.Warnln("skipping function", curField.Name, "due to error:", err.Error())
				continue
			}
			fv := sv.FieldByName(curField.Name)
			fv.Set(reflect.MakeFunc(funcType, toFunc))
		}
	}
	return nil
}

type funcBuilder func(reflect.Type, queryParams, processedQuery) (func([]reflect.Value) []reflect.Value, error)
