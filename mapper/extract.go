package mapper

import (
	"errors"
	"reflect"
	log "github.com/Sirupsen/logrus"
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
