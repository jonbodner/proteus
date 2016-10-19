package mapper

import (
	"errors"
	"reflect"

	"database/sql/driver"
	log "github.com/Sirupsen/logrus"
)

func ExtractType(curType reflect.Type, path []string) (reflect.Type, error) {
	// error case path length == 0
	if len(path) == 0 {
		return nil, errors.New("cannot extract type; no path remaining")
	}
	ss := fromPtrType(curType)
	// base case path length == 1
	if len(path) == 1 {
		return ss, nil
	}
	// length > 1, find a match for path[1], and recurse
	if ss.Kind() == reflect.Map {
		//give up -- we can't figure out what's in the map, so just return the type of the value
		return ss.Elem(), nil
	}
	if ss.Kind() != reflect.Struct {
		return nil, errors.New("Cannot find the type for the subfield of anything other than a struct.")
	}
	//make sure the field exists
	if f, exists := ss.FieldByName(path[1]); exists {
		return ExtractType(f.Type, path[1:])
	}
	return nil, errors.New("cannot find the type; no such field " + path[1])
}

func Extract(s interface{}, path []string) (interface{}, error) {
	// error case path length == 0
	if len(path) == 0 {
		return nil, errors.New("cannot extract value; no path remaining")
	}
	// base case path length == 1
	if len(path) == 1 {
		//if this implements the driver.Valuer interface, call Value, otherwise just return it
		if valuer, ok := s.(driver.Valuer); ok {
			return valuer.Value()
		}
		return s, nil
	}
	// length > 1, find a match for path[1], and recurse
	ss := fromPtr(s)
	sv := reflect.ValueOf(ss)
	if sv.Kind() == reflect.Map {
		if sv.Type().Key().Kind() != reflect.String {
			return nil, errors.New("cannot extract value; map does not have a string key")
		}
		log.Debugln(path[1])
		log.Debugln(sv.MapKeys())
		v := sv.MapIndex(reflect.ValueOf(path[1]))
		log.Debugln(v)
		if !v.IsValid() {
			return nil, errors.New("cannot extract value; no such map key " + path[1])
		}
		return Extract(v.Interface(), path[1:])
	}
	if sv.Kind() == reflect.Struct {
		//make sure the field exists
		if _, exists := sv.Type().FieldByName(path[1]); !exists {
			return nil, errors.New("cannot extract value; no such field " + path[1])
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

func fromPtrType(s reflect.Type) reflect.Type {
	for s.Kind() == reflect.Ptr {
		s = s.Elem()
	}
	return s
}
