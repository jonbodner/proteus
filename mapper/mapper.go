package mapper

import (
	"errors"
	"fmt"
	"github.com/jonbodner/gdb/api"
	"reflect"
	"strings"
	log "github.com/Sirupsen/logrus"
)

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
	//fmt.Println(sType)
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
	for i := 0; i < len(vals); i++ {
		vals[i] = new(interface{})
	}

	err = rows.Scan(vals...)
	if err != nil {
		log.Warnln("scan failed")
		return nil, err
	}

	isPtr := false
	if sType.Kind() == reflect.Ptr {
		sType = sType.Elem()
		isPtr = true
	}
	var out reflect.Value
	if sType.Kind() == reflect.Map {
		out, err = buildMap(sType, cols, vals)
	} else if sType.Kind() == reflect.Struct {
		out, err = buildStruct(sType, cols, vals)
	} else {
		// assume primitive
		out, err = buildPrimitive(sType, cols, vals)
	}

	if err != nil {
		return nil, err
	}
	if isPtr {
		out2 := reflect.New(sType)
		out2.Elem().Set(out)
		return out2.Interface(), nil
	}
	return out.Interface(), nil
}

func buildMap(sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
	out := reflect.MakeMap(sType)
	if sType.Key().Kind() != reflect.String {
		return out, errors.New("Only maps with string keys are supported")
	}
	for k, v := range cols {
		curVal := vals[k]
		rv := reflect.ValueOf(curVal)
		if rv.Elem().Elem().Type().ConvertibleTo(sType.Elem()) {
			out.SetMapIndex(reflect.ValueOf(v), rv.Elem().Elem().Convert(sType.Elem()))
		} else {
			return out, fmt.Errorf("Unable to assign value %v of type %v to map value of type %v with key %s", rv.Elem().Elem(), rv.Elem().Elem().Type(), sType.Elem(), v)
		}
	}
	return out, nil
}

func buildStruct(sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
	out := reflect.New(sType).Elem()
	//build map of col names to field names (makes this 2N instead of N^2)
	colFieldMap := map[string]reflect.StructField{}
	for i := 0; i < sType.NumField(); i++ {
		sf := sType.Field(i)
		if tagVal := sf.Tag.Get("gdbf"); tagVal != "" {
			colFieldMap[strings.SplitN(tagVal, ",", 2)[0]] = sf
		}
	}
	for k, v := range cols {
		if sf, ok := colFieldMap[v]; ok {
			curVal := vals[k]
			rv := reflect.ValueOf(curVal)
			if sf.Type.Kind() == reflect.Ptr {
				if rv.Elem().Elem().Type().ConvertibleTo(sf.Type.Elem()) {
					out.FieldByName(sf.Name).Set(reflect.New(sf.Type.Elem()))
					out.FieldByName(sf.Name).Elem().Set(rv.Elem().Elem().Convert(sf.Type.Elem()))
				} else {
					log.Warnln("can't find the field")
					return out, fmt.Errorf("Unable to assign pointer to value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.Name, sf.Type)
				}
			} else {
				if rv.Elem().Elem().Type().ConvertibleTo(sf.Type) {
					out.FieldByName(sf.Name).Set(rv.Elem().Elem().Convert(sf.Type))
				} else {
					log.Warnln("can't find the field")
					return out, fmt.Errorf("Unable to assign value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.Name, sf.Type)
				}
			}
		}
	}
	return out, nil
}

func buildPrimitive(sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
	out := reflect.New(sType).Elem()
	rv := reflect.ValueOf(vals[0])
	if rv.Elem().Elem().Type().ConvertibleTo(sType) {
		out.Set(rv.Elem().Elem().Convert(sType))
	} else {
		return out, fmt.Errorf("Unable to assign value %v of type %v to return type of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sType)
	}
	return out, nil
}