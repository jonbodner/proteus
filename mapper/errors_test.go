package mapper

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestAssignErrorKinds(t *testing.T) {
	stringType := reflect.TypeOf("")
	err := AssignError{Kind: NilReturnForNonPointer, ToType: stringType}

	if !errors.Is(err, AssignError{Kind: NilReturnForNonPointer}) {
		t.Error("AssignError should match exact kind")
	}
	if !errors.Is(err, AssignError{}) {
		t.Error("AssignError should match any-AssignError wildcard")
	}
	if errors.Is(err, AssignError{Kind: MapAssign}) {
		t.Error("AssignError{NilReturnForNonPointer} should not match AssignError{MapAssign}")
	}
}

func TestAssignErrorMessages(t *testing.T) {
	stringType := reflect.TypeOf("")
	intType := reflect.TypeOf(0)
	cases := []struct {
		err  AssignError
		want string
	}{
		{AssignError{Kind: InvalidOutputType}, "sType cannot be nil"},
		{AssignError{Kind: InvalidMapKeyType}, "only maps with string keys are supported"},
		{AssignError{Kind: NilReturnForNonPointer, ToType: stringType}, "attempting to return nil for non-pointer type string"},
		{AssignError{Kind: StructNilAssign, Field: "Name", ToType: stringType}, "unable to assign nil value to non-pointer struct field Name of type string"},
		{AssignError{Kind: PrimitiveAssign, Value: 42, FromType: intType, ToType: stringType}, "unable to assign value 42 of type int to return type of type string"},
	}
	for _, c := range cases {
		if c.err.Error() != c.want {
			t.Errorf("expected %q, got %q", c.want, c.err.Error())
		}
	}
}

func TestAssignErrorsAsExtraction(t *testing.T) {
	stringType := reflect.TypeOf("")
	err := AssignError{Kind: NilReturnForNonPointer, ToType: stringType}
	if ae, ok := errors.AsType[AssignError](err); ok {
		if ae.ToType != stringType {
			t.Errorf("expected ToType=string, got %v", ae.ToType)
		}
	} else {
		t.Fatal("errors.As should succeed for AssignError")
	}
}

func TestExtractErrorKinds(t *testing.T) {
	err := ExtractError{Kind: NoSuchField, Value: "Name"}
	if !errors.Is(err, ExtractError{Kind: NoSuchField}) {
		t.Error("ExtractError should match exact kind")
	}
	if !errors.Is(err, ExtractError{}) {
		t.Error("ExtractError should match any-ExtractError wildcard")
	}
	if errors.Is(err, ExtractError{Kind: NoSuchMapKey}) {
		t.Error("ExtractError{NoSuchField} should not match ExtractError{NoSuchMapKey}")
	}
}

func TestExtractErrorMessages(t *testing.T) {
	cases := []struct {
		err  ExtractError
		want string
	}{
		{ExtractError{Kind: NoPathRemaining}, "cannot extract type; no path remaining"},
		{ExtractError{Kind: ValueNoPathRemaining}, "cannot extract value; no path remaining"},
		{ExtractError{Kind: NoSuchField, Value: "Foo"}, "cannot extract value; no such field Foo"},
		{ExtractError{Kind: NoSuchMapKey, Value: "bar"}, "cannot extract value; no such map key bar"},
		{ExtractError{Kind: NoSuchFieldType, Value: "Baz"}, "cannot find the type; no such field Baz"},
		{ExtractError{Kind: InvalidIndex, Value: "xyz"}, "invalid index: xyz"},
	}
	for _, c := range cases {
		if c.err.Error() != c.want {
			t.Errorf("expected %q, got %q", c.want, c.err.Error())
		}
	}
}

func TestExtractErrorInvalidIndexUnwrap(t *testing.T) {
	_, parseErr := strconv.Atoi("abc")
	err := ExtractError{Kind: InvalidIndex, Value: "abc", Err: parseErr}

	if !errors.Is(err, ExtractError{Kind: InvalidIndex}) {
		t.Error("InvalidIndex ExtractError should match its kind")
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Error("InvalidIndex ExtractError should unwrap to strconv.ErrSyntax")
	}
	noWrap := ExtractError{Kind: InvalidIndex, Value: "5"}
	if noWrap.Unwrap() != nil {
		t.Error("Unwrap should return nil when Err is nil")
	}
}

func TestExtractErrorsAsExtraction(t *testing.T) {
	err := ExtractError{Kind: NoSuchField, Value: "MyField"}
	if ee, ok := errors.AsType[ExtractError](err); ok {
		if ee.Value != "MyField" {
			t.Errorf("expected Field=MyField, got %s", ee.Value)
		}
	} else {
		t.Fatal("errors.As should succeed for ExtractError")
	}
}
