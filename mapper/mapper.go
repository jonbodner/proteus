package mapper

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/jonbodner/proteus/logger"
	"github.com/jonbodner/stackerr"
)

func ptrConverter(ctx context.Context, isPtr bool, sType reflect.Type, out reflect.Value, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	if isPtr {
		out2 := reflect.New(sType)
		k := out.Type().Kind()
		logger.Log(ctx, logger.DEBUG, fmt.Sprintln("kind of out", k))
		if (k == reflect.Interface || k == reflect.Ptr) && out.IsNil() {
			out2 = reflect.NewAt(sType, unsafe.Pointer(nil))
		} else {
			out2.Elem().Set(out)
		}
		return out2.Interface(), nil
	}
	k := out.Type().Kind()
	if (k == reflect.Ptr || k == reflect.Interface) && out.IsNil() {
		return nil, stackerr.Errorf("attempting to return nil for non-pointer type %v", sType)
	}
	return out.Interface(), nil
}

func MakeBuilder(ctx context.Context, sType reflect.Type) (Builder, error) {
	if sType == nil {
		return nil, stackerr.New("sType cannot be nil")
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
			return nil, stackerr.New("only maps with string keys are supported")
		}
		return func(cols []string, vals []interface{}) (interface{}, error) {
			out, err := buildMap(ctx, sType, cols, vals)
			return ptrConverter(ctx, isPtr, sType, out, err)
		}, nil
	} else if sType.Kind() == reflect.Struct {
		//build map of col names to field names (makes this 2N instead of N^2)
		colFieldMap := map[string]fieldInfo{}
		buildColFieldMap(sType, fieldInfo{}, colFieldMap)
		return func(cols []string, vals []interface{}) (interface{}, error) {
			out, err := buildStruct(ctx, sType, cols, vals, colFieldMap)
			return ptrConverter(ctx, isPtr, sType, out, err)
		}, nil

	} else {
		// assume primitive
		return func(cols []string, vals []interface{}) (interface{}, error) {
			out, err := buildPrimitive(ctx, sType, cols, vals)
			return ptrConverter(ctx, isPtr, sType, out, err)
		}, nil
	}
}

func buildColFieldMap(sType reflect.Type, parentFieldInfo fieldInfo, colFieldMap map[string]fieldInfo) {
	for i := 0; i < sType.NumField(); i++ {
		sf := sType.Field(i)
		if tagVal := sf.Tag.Get("prof"); tagVal != "" {
			childFieldInfo := fieldInfo{
				name:      append([]string(nil), parentFieldInfo.name...),
				fieldType: append([]reflect.Type(nil), parentFieldInfo.fieldType...),
				pos:       append([]int(nil), parentFieldInfo.pos...),
			}
			childFieldInfo.name = append(childFieldInfo.name, sf.Name)
			childFieldInfo.fieldType = append(childFieldInfo.fieldType, sf.Type)
			childFieldInfo.pos = append(childFieldInfo.pos, i)
			//if this is a struct, recurse
			// we are going to use an prof value as a prefix
			// then we go through each of the fields in the struct
			// if any of them aren't structs and have prof tags, we
			// prepend the parent struct and store off a fieldi
			// only if this doesn't implement a scanner. If it does, then go with the scanner
			// another special case: time.Time isn't recursed into
			if sf.Type.Kind() == reflect.Struct && !reflect.PtrTo(sf.Type).Implements(scannerType) && sf.Type.Name() != "Time" && sf.Type.PkgPath() != "time" {
				buildColFieldMap(sf.Type, childFieldInfo, colFieldMap)
			} else {
				colFieldMap[strings.SplitN(tagVal, ",", 2)[0]] = childFieldInfo
			}
			continue
		}

		//last chance, if it's an anonymous struct, you can recurse into it without having a prof tag
		if sf.Type.Kind() == reflect.Struct && sf.Anonymous {
			childFieldInfo := fieldInfo{
				name:      append([]string(nil), parentFieldInfo.name...),
				fieldType: append([]reflect.Type(nil), parentFieldInfo.fieldType...),
				pos:       append([]int(nil), parentFieldInfo.pos...),
			}
			childFieldInfo.name = append(childFieldInfo.name, "")
			childFieldInfo.fieldType = append(childFieldInfo.fieldType, sf.Type)
			childFieldInfo.pos = append(childFieldInfo.pos, i)
			buildColFieldMap(sf.Type, childFieldInfo, colFieldMap)
		}
	}
}

type Builder func(cols []string, vals []interface{}) (interface{}, error)

type fieldInfo struct {
	name      []string
	fieldType []reflect.Type
	pos       []int
}

func buildMap(ctx context.Context, sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
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
			return out, stackerr.Errorf("unable to assign value %v of type %v to map value of type %v with key %s", rv.Elem().Elem(), rv.Elem().Elem().Type(), sType.Elem(), v)
		}
	}
	return out, nil
}

var (
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

func buildStruct(ctx context.Context, sType reflect.Type, cols []string, vals []interface{}, colFieldMap map[string]fieldInfo) (reflect.Value, error) {
	logger.Log(ctx, logger.DEBUG, fmt.Sprintf("sType: %s cols: %v vals: %+v colFieldMap: %+v", sType, cols, vals, colFieldMap))
	out := reflect.New(sType).Elem()
	for k, v := range cols {
		if sf, ok := colFieldMap[v]; ok {
			curVal := vals[k]
			rv := reflect.ValueOf(curVal)
			err := buildStructInner(ctx, sType, out, sf, curVal, rv, 0)
			if err != nil {
				return out, err
			}
		}
	}
	return out, nil
}

func buildStructInner(ctx context.Context, sType reflect.Type, out reflect.Value, sf fieldInfo, curVal interface{}, rv reflect.Value, depth int) error {
	field := out.Field(sf.pos[depth])
	curFieldType := sf.fieldType[depth]
	if curFieldType.Kind() == reflect.Ptr {
		logger.Log(ctx, logger.DEBUG, fmt.Sprintln("isPtr", sf, rv, rv.Type(), rv.Elem(), curVal, sType))
		if rv.Elem().IsNil() {
			logger.Log(ctx, logger.DEBUG, fmt.Sprintln("nil", sf, rv, curVal, sType))
			return nil
		}
		logger.Log(ctx, logger.DEBUG, fmt.Sprintln("isPtr Not Nil", rv.Elem().Type()))
		if curFieldType.Implements(scannerType) {
			toScan := (reflect.New(curFieldType).Elem().Interface()).(sql.Scanner)
			err := toScan.Scan(rv.Elem().Elem().Interface())
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(toScan))
		} else if rv.Elem().Elem().Type().ConvertibleTo(curFieldType.Elem()) {
			field.Set(reflect.New(curFieldType.Elem()))
			field.Elem().Set(rv.Elem().Elem().Convert(curFieldType.Elem()))
		} else {
			logger.Log(ctx, logger.ERROR, fmt.Sprintln("can't find the field"))
			return stackerr.Errorf("unable to assign pointer to value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.name[depth], curFieldType)
		}
	} else {
		if reflect.PtrTo(curFieldType).Implements(scannerType) {
			toScan := (reflect.New(curFieldType).Interface()).(sql.Scanner)
			err := toScan.Scan(rv.Elem().Interface())
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(toScan).Elem())
		} else if field.Kind() == reflect.Struct && field.Type().Name() != "Time" && field.Type().PkgPath() != "time" {
			err := buildStructInner(ctx, field.Type(), field, sf, curVal, rv, depth+1)
			if err != nil {
				return err
			}
		} else if rv.Elem().IsNil() {
			logger.Log(ctx, logger.ERROR, fmt.Sprintln("Attempting to assign a nil to a non-pointer field"))
			return stackerr.Errorf("unable to assign nil value to non-pointer struct field %s of type %v", sf.name[depth], curFieldType)
		} else if rv.Elem().Elem().Type().ConvertibleTo(curFieldType) {
			field.Set(rv.Elem().Elem().Convert(curFieldType))
		} else {
			logger.Log(ctx, logger.ERROR, fmt.Sprintln("can't find the field"))
			return stackerr.Errorf("unable to assign value %v of type %v to struct field %s of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sf.name[depth], curFieldType)
		}
	}
	return nil
}

func buildPrimitive(ctx context.Context, sType reflect.Type, cols []string, vals []interface{}) (reflect.Value, error) {
	out := reflect.New(sType).Elem()
	//vals[0] is of type *interface{}, because everything in vals is of type *interface{}
	rv := reflect.ValueOf(vals[0])
	if rv.Elem().IsNil() {
		return rv.Elem(), nil
	}
	if rv.Elem().Elem().Type().ConvertibleTo(sType) {
		out.Set(rv.Elem().Elem().Convert(sType))
	} else {
		return out, stackerr.Errorf("unable to assign value %v of type %v to return type of type %v", rv.Elem().Elem(), rv.Elem().Elem().Type(), sType)
	}
	return out, nil
}
