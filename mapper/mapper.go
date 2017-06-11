package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"reflect"
	"strings"
	"unsafe"
)

func ptrConverter(isPtr bool, sType reflect.Type, out reflect.Value, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	if isPtr {
		out2 := reflect.New(sType)
		k := out.Type().Kind()
		log.Debugln("kind of out", k)
		if (k == reflect.Interface || k == reflect.Ptr) && out.IsNil() {
			out2 = reflect.NewAt(sType, unsafe.Pointer(nil))
		} else {
			out2.Elem().Set(out)
		}
		return out2.Interface(), nil
	}
	k := out.Type().Kind()
	if (k == reflect.Ptr || k == reflect.Interface) && out.IsNil() {
		return nil, fmt.Errorf("Attempting to return nil for non-pointer type %v", sType)
	}
	return out.Interface(), nil
}

func MakeBuilder(sType reflect.Type) (Builder, error) {
	if sType == nil {
		return nil, errors.New("sType cannot be nil")
	}

	isPtr := false
	if sType.Kind() == reflect.Ptr {
		sType = sType.Elem()
		isPtr = true
	}

	//only going to handle scalar types here
	if sType.Kind() == reflect.Slice {
		sType = sType.Elem()
	}

	if sType.Kind() == reflect.Map {
		if sType.Key().Kind() != reflect.String {
			return nil, errors.New("Only maps with string keys are supported")
		}
		return func(cols []string, vals []interface{}) (interface{}, error) {
			out, err := buildMap(sType, cols, vals)
			return ptrConverter(isPtr, sType, out, err)
		}, nil
	} else if sType.Kind() == reflect.Struct {
		//build map of col names to field names (makes this 2N instead of N^2)
		colFieldMap := map[string]fieldInfo{}
		for i := 0; i < sType.NumField(); i++ {
			sf := sType.Field(i)
			if tagVal := sf.Tag.Get("prof"); tagVal != "" {
				colFieldMap[strings.SplitN(tagVal, ",", 2)[0]] = fieldInfo{
					name:      sf.Name,
					fieldType: sf.Type,
					pos:       i,
				}
			}
		}
		return func(cols []string, vals []interface{}) (interface{}, error) {
			out, err := buildStruct(sType, cols, vals, colFieldMap)
			return ptrConverter(isPtr, sType, out, err)
		}, nil

	} else {
		// assume primitive
		return func(cols []string, vals []interface{}) (interface{}, error) {
			out, err := buildPrimitive(sType, cols, vals)
			return ptrConverter(isPtr, sType, out, err)
		}, nil
	}
}

type Builder func(cols []string, vals []interface{}) (interface{}, error)

type fieldInfo struct {
	name      string
	fieldType reflect.Type
	pos       int
}

func buildMap(sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
	out := reflect.MakeMap(sType)
	for k, v := range cols {
		curVal := vals[k]
		rv := reflect.ValueOf(curVal)
		if rv.Elem().IsNil() {
			continue
		}
		if rv.Elem().Elem().Type().ConvertibleTo(sType.Elem()) {
			out.SetMapIndex(reflect.ValueOf(v), rv.Elem().Elem().Convert(sType.Elem()))
		} else {
			return out, fmt.Errorf("Unable to assign value %v of type %v to map value of type %v with key %s", rv.Elem().Elem(), rv.Elem().Elem().Type(), sType.Elem(), v)
		}
	}
	return out, nil
}

var (
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

func buildStruct(sType reflect.Type, cols []string, vals []interface{}, colFieldMap map[string]fieldInfo) (reflect.Value, error) {
	out := reflect.New(sType).Elem()
	for k, v := range cols {
		if sf, ok := colFieldMap[v]; ok {
			curVal := vals[k]
			rv := reflect.ValueOf(curVal)
			if sf.fieldType.Kind() == reflect.Ptr {
				//log.Debugln("isPtr", sf, rv, rv.Type(), rv.Elem(), curVal, sType)
				if rv.Elem().IsNil() {
					//log.Debugln("nil", sf, rv, curVal, sType)
					continue
				}
				//log.Debugln("isPtr Not Nil", rv.Elem().Type())
				if sf.fieldType.Implements(scannerType) {
					toScan := (reflect.New(sf.fieldType).Elem().Interface()).(sql.Scanner)
					err := toScan.Scan(rv.Elem().Elem().Interface())
					if err != nil {
						return out, err
					}
					out.Field(sf.pos).Set(reflect.ValueOf(toScan))
				} else if rv.Elem().Elem().Type().ConvertibleTo(sf.fieldType.Elem()) {
					out.Field(sf.pos).Set(reflect.New(sf.fieldType.Elem()))
					out.Field(sf.pos).Elem().Set(rv.Elem().Elem().Convert(sf.fieldType.Elem()))
				} else {
					log.Warnln("can't find the field")
					return out, fmt.Errorf("Unable to assign pointer to value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.name, sf.fieldType)
				}
			} else {
				if rv.Elem().IsNil() {
					log.Warnln("Attempting to assign a nil to a non-pointer field")
					return out, fmt.Errorf("Unable to assign nil value to non-pointer struct field %s of type %v", sf.name, sf.fieldType)
				}
				if reflect.PtrTo(sf.fieldType).Implements(scannerType) {
					toScan := (reflect.New(sf.fieldType).Interface()).(sql.Scanner)
					err := toScan.Scan(rv.Elem().Elem().Interface())
					if err != nil {
						return out, err
					}
					out.Field(sf.pos).Set(reflect.ValueOf(toScan).Elem())
				} else if rv.Elem().Elem().Type().ConvertibleTo(sf.fieldType) {
					out.Field(sf.pos).Set(rv.Elem().Elem().Convert(sf.fieldType))
				} else {
					log.Warnln("can't find the field")
					return out, fmt.Errorf("Unable to assign value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.name, sf.fieldType)
				}
			}
		}
	}
	return out, nil
}

func buildPrimitive(sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
	out := reflect.New(sType).Elem()
	//vals[0] is of type *interface{}, because everything in vals is of type *interface{}
	rv := reflect.ValueOf(vals[0])
	if rv.Elem().IsNil() {
		return rv.Elem(), nil
	}
	if rv.Elem().Elem().Type().ConvertibleTo(sType) {
		out.Set(rv.Elem().Elem().Convert(sType))
	} else {
		return out, fmt.Errorf("Unable to assign value %v of type %v to return type of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sType)
	}
	return out, nil
}
