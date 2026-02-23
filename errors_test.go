package proteus

import (
	"errors"
	"testing"
)

func TestValidationErrorIdentity(t *testing.T) {
	err := ValidationError{Kind: NotPointer}
	// exact kind match
	if !errors.Is(err, ValidationError{Kind: NotPointer}) {
		t.Error("ValidationError{NotPointer} should match itself")
	}
	// wildcard match
	if !errors.Is(err, ValidationError{}) {
		t.Error("ValidationError{NotPointer} should match any-ValidationError wildcard")
	}
	// different kind should not match
	if errors.Is(err, ValidationError{Kind: NotPointerToStruct}) {
		t.Error("ValidationError{NotPointer} should not match ValidationError{NotPointerToStruct}")
	}
}

func TestValidationErrorMessage(t *testing.T) {
	cases := []struct {
		kind ValidationErrorKind
		want string
	}{
		{NotPointer, "not a pointer"},
		{NotPointerToStruct, "not a pointer to struct"},
		{NeedExecutorOrQuerier, "need to supply an Executor or Querier parameter"},
		{InvalidFirstParam, "first parameter must be of type context.Context, Executor, or Querier"},
		{RowsMustBeNonNil, "rows must be non-nil"},
	}
	for _, c := range cases {
		e := ValidationError{Kind: c.kind}
		if e.Error() != c.want {
			t.Errorf("Kind %d: expected %q, got %q", c.kind, c.want, e.Error())
		}
	}
}

func TestQueryErrorKinds(t *testing.T) {
	err := QueryError{Kind: QueryNotFound, Name: "foo"}
	if !errors.Is(err, QueryError{Kind: QueryNotFound}) {
		t.Error("QueryError{QueryNotFound} should match exact kind")
	}
	if !errors.Is(err, QueryError{}) {
		t.Error("QueryError should match any-QueryError wildcard")
	}
	if errors.Is(err, QueryError{Kind: ParameterNotFound}) {
		t.Error("QueryError{QueryNotFound} should not match QueryError{ParameterNotFound}")
	}
	if err.Error() != "no query found for name foo" {
		t.Errorf("unexpected message: %s", err.Error())
	}
}

func TestQueryErrorMessages(t *testing.T) {
	cases := []struct {
		err  QueryError
		want string
	}{
		{QueryError{Kind: MissingClosingColon, Query: "select *"}, "missing a closing : somewhere: select *"},
		{QueryError{Kind: EmptyVariable, Position: 10}, "empty variable declaration at position 10"},
		{QueryError{Kind: ParameterNotFound, Name: "p"}, "query parameter p cannot be found in the incoming parameters"},
		{QueryError{Kind: NilParameterPath, Name: "p"}, "query parameter p has a path, but the incoming parameter is nil"},
		{QueryError{Kind: InvalidParameterType, Name: "p", TypeKind: "int"}, "query parameter p has a path, but the incoming parameter is not a map or a struct it is int"},
	}
	for _, c := range cases {
		if c.err.Error() != c.want {
			t.Errorf("expected %q, got %q", c.want, c.err.Error())
		}
	}
}

func TestIdentifierErrorKinds(t *testing.T) {
	err := IdentifierError{Kind: InvalidCharacterInIdentifier, Identifier: "a,b"}
	if !errors.Is(err, IdentifierError{Kind: InvalidCharacterInIdentifier}) {
		t.Error("IdentifierError should match exact kind")
	}
	if !errors.Is(err, IdentifierError{}) {
		t.Error("IdentifierError should match any-IdentifierError wildcard")
	}
	if errors.Is(err, IdentifierError{Kind: SemicolonInIdentifier}) {
		t.Error("IdentifierError{InvalidCharacter} should not match IdentifierError{Semicolon}")
	}
}

func TestIdentifierErrorMessages(t *testing.T) {
	cases := []struct {
		err  IdentifierError
		want string
	}{
		{IdentifierError{Kind: SemicolonInIdentifier, Identifier: "a;b"}, "; is not allowed in an identifier: a;b"},
		{IdentifierError{Kind: EmptyOrTrailingDotIdentifier, Identifier: "a."}, "identifiers cannot be empty or end with a .: a."},
		{IdentifierError{Kind: MissingDotInIdentifier, Identifier: "ab"}, ". missing between parts of an identifier: ab"},
		{IdentifierError{Kind: LeadingOrDoubleDotIdentifier, Identifier: ".a"}, "identifier cannot start with . or have two . in a row: .a"},
		{IdentifierError{Kind: InvalidCharacterInIdentifier, Identifier: "a,b"}, "invalid character found in identifier: a,b"},
	}
	for _, c := range cases {
		if c.err.Error() != c.want {
			t.Errorf("expected %q, got %q", c.want, c.err.Error())
		}
	}
}

func TestValidationErrorPropagation(t *testing.T) {
	wrapped := Error{
		FuncName:      "TestFunc",
		FieldOrder:    0,
		OriginalError: ValidationError{Kind: NotPointer},
	}
	if !errors.Is(wrapped, ValidationError{Kind: NotPointer}) {
		t.Error("ValidationError should be reachable through Error.Unwrap()")
	}
	if !errors.Is(wrapped, ValidationError{}) {
		t.Error("any-ValidationError wildcard should be reachable through Error.Unwrap()")
	}
}

func TestErrorsAsExtraction(t *testing.T) {
	err := QueryError{Kind: QueryNotFound, Name: "myquery"}
	var qe QueryError
	if !errors.As(err, &qe) {
		t.Fatal("errors.As should succeed for QueryError")
	}
	if qe.Name != "myquery" {
		t.Errorf("expected Name=myquery, got %s", qe.Name)
	}

	err2 := IdentifierError{Kind: SemicolonInIdentifier, Identifier: "a;b"}
	var ie IdentifierError
	if !errors.As(err2, &ie) {
		t.Fatal("errors.As should succeed for IdentifierError")
	}
	if ie.Identifier != "a;b" {
		t.Errorf("expected Identifier=a;b, got %s", ie.Identifier)
	}
}
