package mapper

import (
	"reflect"
	"errors"
	"fmt"
	"github.com/jonbodner/gdb/api"
)

func Extract(s interface{}, path []string) (interface{}, error) {
	// error case path length == 0
	if len(path) == 0 {
		return nil, errors.New("cannot extract value; no path remaining")
	}
	// base case path length == 1
	if len(path) == 1 {
		return fromPtr(s), nil
	}
	// length > 1, find a match for path[1], and recurse
	ss := fromPtr(s)
	sv := reflect.ValueOf(ss)
	if sv.Kind() == reflect.Map {
		if sv.Type().Key().Kind() != reflect.String {
			return nil, errors.New("cannot extract value; map does not have a string key")
		}
		v := sv.MapIndex(reflect.ValueOf(path[1]))
		return Extract(v.Interface(), path[1:])
	}
	if sv.Kind() == reflect.Struct {
		//make sure the field exists
		if _, exists := sv.Type().FieldByName(path[1]); !exists {
			return nil, errors.New("cannot extract value; no such field "+path[1])
		}

		v := sv.FieldByName(path[1])
		return Extract(v.Interface(), path[1:])
	}
	return nil, errors.New("cannot extract value; only maps and structs can have contained values")
}

func fromPtr(s interface{}) interface{} {
	st := reflect.TypeOf(s)
	if st.Kind() == reflect.Ptr {
		sp := reflect.ValueOf(s).Elem()
		return sp.Interface()

	}
	return s
}

// Build takes the next value from Rows and uses it to create a new instance of the specified type
// If the type is a primitive and there are more than 1 values in the current row, only the first value is used.
// If the type is a map of string to interface, then the column names are the keys in the map and the values are assigned
// If the type is a struct that has gdbp tags on its fields, then any matching tags will be associated with values with the associate columns
// Any non-associated values will be set to the zero value
// If any columns cannot be assigned to any types, then an error is returned
// If next returns false, then nil is returned for both the interface and the error
// If an error occurs while processing the current row, nil is returned for the interface and the error is non-nil
// If a value is successfuly extracted from the current row, the instance is returned and the error is nil
func Build(rows api.Rows, sType reflect.Type) (interface{}, error) {
	if rows == nil || sType == nil {
		return nil, errors.New("Both rows and sType must be non-nil")
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

	err = rows.Scan(vals...)
	if err != nil {
		return nil, err
	}

	isPtr := false
	if sType.Kind() == reflect.Ptr {
		sType = sType.Elem()
	}
	var out reflect.Value
	if sType.Kind() == reflect.Map {
		if sType.Key().Kind() != reflect.String {
			return nil, errors.New("Only maps with string keys are supported")
		}
		out = reflect.MakeMap(sType)
		for k, v := range cols {
			curVal := vals[k]
			rv := reflect.ValueOf(curVal)
			if rv.Type().AssignableTo(sType.Elem()) {
				out.SetMapIndex(reflect.ValueOf(v), rv)
			} else {
				return nil, fmt.Errorf("Unable to assign value %v of type %v to map value of type %v",curVal,rv.Type(), sType.Elem())
			}
		}
	} else if sType.Kind() == reflect.Struct {
		out = reflect.Zero(sType)
		//build map of col names to field names (makes this 2N instead of N^2)
		colFieldMap := map[string]reflect.StructField{}
		for i := 0;i<sType.NumField();i++ {
			sf := sType.Field(i)
			if tagVal, ok := sf.Tag.Lookup("gdbp"); ok {
				colFieldMap[tagVal] = sf
			}
		}
		for k, v := range cols {
			if sf, ok := colFieldMap[v]; ok {
				curVal := vals[k]
				rv := reflect.ValueOf(curVal)
				if rv.Type().AssignableTo(sf.Type) {
					out.FieldByName(sf.Name).Set(rv)
				} else {
					return nil, fmt.Errorf("Unable to assign value %v of type %v to struct field %s of type %v",curVal,rv.Type(), sf.Name, sf.Type)
				}
			}
		}
	} else {
		// assume primitive
		out = reflect.Zero(sType)
		rv := reflect.ValueOf(vals[0])
		if rv.Type().AssignableTo(sType) {
			out.Set(rv)
		} else {
			return nil, fmt.Errorf("Unable to assign value %v of type %v to return type of type %v",vals[0],rv.Type(), sType)
		}
	}

	if isPtr {
		oi := out.Interface()
		return &oi, nil
	}
	return out.Interface(), nil
}
