package mapper

import (
	"context"
	"database/sql/driver"
	"log/slog"
	"reflect"
	"strconv"

	"github.com/jonbodner/stackerr"
)

func ExtractType(ctx context.Context, curType reflect.Type, path []string) (reflect.Type, error) {
	// error case path length == 0
	if len(path) == 0 {
		return nil, stackerr.New("cannot extract type; no path remaining")
	}
	ss := fromPtrType(curType)
	// base case path length == 1
	if len(path) == 1 {
		return ss, nil
	}
	// length > 1, find a match for path[1], and recurse
	if ss == nil {
		return nil, stackerr.New("cannot find the type for the subfield of a nil")
	}
	switch ss.Kind() {
	case reflect.Map:
		//give up -- we can't figure out what's in the map, so just return the type of the value
		return ss.Elem(), nil
	case reflect.Struct:
		//make sure the field exists
		if f, exists := ss.FieldByName(path[1]); exists {
			return ExtractType(ctx, f.Type, path[1:])
		}
		return nil, stackerr.New("cannot find the type; no such field " + path[1])
	case reflect.Array, reflect.Slice:
		// handle slices and arrays
		_, err := strconv.Atoi(path[1])
		if err != nil {
			return nil, fmt.Errorf("invalid index: %s :%w", path[1], err)
		}
		return ExtractType(ctx, ss.Elem(), path[1:])
	default:
		return nil, stackerr.New("cannot find the type for the subfield of anything other than a map, struct, slice, or array")
	}
}

func Extract(ctx context.Context, s any, path []string) (any, error) {
	// error case path length == 0
	if len(path) == 0 {
		return nil, stackerr.New("cannot extract value; no path remaining")
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
	switch sv.Kind() {
	case reflect.Map:
		if sv.Type().Key().Kind() != reflect.String {
			return nil, stackerr.New("cannot extract value; map does not have a string key")
		}
		slog.DebugContext(ctx, "map extract", "key", path[1], "availableKeys", sv.MapKeys())
		v := sv.MapIndex(reflect.ValueOf(path[1]))
		slog.DebugContext(ctx, "map extract result", "value", v)
		if !v.IsValid() {
			return nil, stackerr.New("cannot extract value; no such map key " + path[1])
		}
		return Extract(ctx, v.Interface(), path[1:])
	case reflect.Struct:
		//make sure the field exists
		if _, exists := sv.Type().FieldByName(path[1]); !exists {
			return nil, stackerr.New("cannot extract value; no such field " + path[1])
		}

		v := sv.FieldByName(path[1])
		return Extract(ctx, v.Interface(), path[1:])
	case reflect.Array, reflect.Slice:
		// handle slices and arrays
		pos, err := strconv.Atoi(path[1])
		if err != nil {
			return nil, fmt.Errorf("invalid index: %s :%w", path[1], err)
		}
		if pos < 0 || pos >= sv.Len() {
			return nil, fmt.Errorf("invalid index: %s", path[1])
		}
		v := sv.Index(pos)
		return Extract(ctx, v.Interface(), path[1:])
	default:
		return nil, stackerr.New("cannot extract value; only maps and structs can have contained values")
	}
}

func fromPtr(s any) any {
	st := reflect.TypeOf(s)
	if st != nil && st.Kind() == reflect.Pointer {
		sp := reflect.ValueOf(s).Elem()
		return sp.Interface()
	}
	return s
}

func fromPtrType(s reflect.Type) reflect.Type {
	if s == nil {
		return nil
	}
	for s.Kind() == reflect.Pointer {
		s = s.Elem()
	}
	return s
}
