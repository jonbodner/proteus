package mapper

import (
	"fmt"
	"reflect"
)

// ExtractErrorKind identifies the specific path-extraction failure.
// The zero value (AnyExtract) acts as a wildcard for errors.Is matching.
type ExtractErrorKind int

const (
	AnyExtract                 ExtractErrorKind = iota
	NoPathRemaining                             // type extraction: no path left
	SubfieldOfNil                               // cannot descend into nil
	SubfieldUnsupportedKind                     // non-map/struct/slice/array subfield
	ValueNoPathRemaining                        // value extraction: no path left
	ValueMapNonStringKey                        // map key is not a string
	ValueContainedNonMapStruct                  // cannot descend into non-map/struct
	NoSuchFieldType                             // Field: struct field name not found (type extraction)
	NoSuchMapKey                                // Key: map key not found
	NoSuchField                                 // Field: struct field name not found (value extraction)
	InvalidIndex                                // Index: non-integer or out-of-range index; Err: strconv error if present
)

// ExtractError is returned when navigating a dot-separated path through a
// value or type fails.
type ExtractError struct {
	Kind  ExtractErrorKind
	Value string // field name (NoSuchFieldType, NoSuchField, NoSuchMapKey) or index (InvalidIndex)
	Err   error  // InvalidIndex: wrapped strconv error (may be nil)
}

func (e ExtractError) Error() string {
	switch e.Kind {
	case NoPathRemaining:
		return "cannot extract type; no path remaining"
	case SubfieldOfNil:
		return "cannot find the type for the subfield of a nil"
	case SubfieldUnsupportedKind:
		return "cannot find the type for the subfield of anything other than a map, struct, slice, or array"
	case ValueNoPathRemaining:
		return "cannot extract value; no path remaining"
	case ValueMapNonStringKey:
		return "cannot extract value; map does not have a string key"
	case ValueContainedNonMapStruct:
		return "cannot extract value; only maps and structs can have contained values"
	case NoSuchFieldType:
		return "cannot find the type; no such field " + e.Value
	case NoSuchMapKey:
		return "cannot extract value; no such map key " + e.Value
	case NoSuchField:
		return "cannot extract value; no such field " + e.Value
	case InvalidIndex:
		if e.Err != nil {
			return fmt.Sprintf("invalid index: %s :%v", e.Value, e.Err)
		}
		return fmt.Sprintf("invalid index: %s", e.Value)
	default:
		return "unknown extract error"
	}
}

// Is matches any ExtractError when target has AnyExtract kind,
// or matches the exact kind otherwise.
func (e ExtractError) Is(target error) bool {
	t, ok := target.(ExtractError)
	if !ok {
		return false
	}
	return t.Kind == AnyExtract || e.Kind == t.Kind
}

// Unwrap returns the underlying strconv error for InvalidIndex, nil otherwise.
func (e ExtractError) Unwrap() error {
	return e.Err
}

// AssignErrorKind identifies the specific value-assignment failure.
// The zero value (AnyAssign) acts as a wildcard for errors.Is matching.
type AssignErrorKind int

const (
	AnyAssign              AssignErrorKind = iota
	InvalidOutputType                      // output type passed to MakeBuilder is nil
	InvalidMapKeyType                      // map key type is not string
	NilReturnForNonPointer                 // ToType: the non-pointer type that got a nil value
	MapAssign                              // Value, FromType, ToType, Key
	StructPointerAssign                    // Value, FromType, FieldName, ToType
	StructNilAssign                        // FieldName, ToType
	StructAssign                           // Value, FromType, FieldName, ToType
	PrimitiveAssign                        // Value, FromType, ToType
)

// AssignError is returned when a database value cannot be assigned to the
// target Go type or field.
type AssignError struct {
	Kind     AssignErrorKind
	Value    any
	FromType reflect.Type
	ToType   reflect.Type
	Field    string // map key (MapAssign) or struct field name (Struct*Assign kinds)
}

func (e AssignError) Error() string {
	switch e.Kind {
	case InvalidOutputType:
		return "sType cannot be nil"
	case InvalidMapKeyType:
		return "only maps with string keys are supported"
	case NilReturnForNonPointer:
		return fmt.Sprintf("attempting to return nil for non-pointer type %v", e.ToType)
	case MapAssign:
		return fmt.Sprintf("unable to assign value %v of type %v to map value of type %v with key %s", e.Value, e.FromType, e.ToType, e.Field)
	case StructPointerAssign:
		return fmt.Sprintf("unable to assign pointer to value %v of type %v to struct field %s of type %v", e.Value, e.FromType, e.Field, e.ToType)
	case StructNilAssign:
		return fmt.Sprintf("unable to assign nil value to non-pointer struct field %s of type %v", e.Field, e.ToType)
	case StructAssign:
		return fmt.Sprintf("unable to assign value %v of type %v to struct field %s of type %v", e.Value, e.FromType, e.Field, e.ToType)
	case PrimitiveAssign:
		return fmt.Sprintf("unable to assign value %v of type %v to return type of type %v", e.Value, e.FromType, e.ToType)
	default:
		return "unknown assign error"
	}
}

// Is matches any AssignError when target has AnyAssign kind,
// or matches the exact kind otherwise.
func (e AssignError) Is(target error) bool {
	t, ok := target.(AssignError)
	if !ok {
		return false
	}
	return t.Kind == AnyAssign || e.Kind == t.Kind
}
