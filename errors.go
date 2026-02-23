package proteus

import "fmt"

// ValidationErrorKind identifies the specific validation failure.
// The zero value (AnyValidation) acts as a wildcard for errors.Is matching.
type ValidationErrorKind int

const (
	AnyValidation         ValidationErrorKind = iota
	NotPointer                                // "not a pointer"
	NotPointerToStruct                        // "not a pointer to struct"
	NotPointerToFunc                          // "not a pointer to func"
	NeedExecutorOrQuerier                     // "need to supply an Executor or Querier parameter"
	InvalidFirstParam                         // "first parameter must be of type context.Context, Executor, or Querier"
	ChannelInputParam                         // "no input parameter can be a channel"
	TooManyReturnValues                       // "must return 0, 1, or 2 values"
	SecondReturnNotError                      // "2nd output parameter must be of type error"
	FirstReturnIsChannel                      // "1st output parameter cannot be a channel"
	ExecutorReturnType                        // "the 1st output parameter of an Executor must be int64 or sql.Result"
	SQLResultWithQuerier                      // "output parameters of type sql.Result must be combined with Executor"
	RowsMustBeNonNil                          // "rows must be non-nil"
	NoValuesFromQuery                         // "no values returned from query"
	ShouldNeverGetHere                        // "should never get here"
)

var validationMessages = map[ValidationErrorKind]string{
	NotPointer:            "not a pointer",
	NotPointerToStruct:    "not a pointer to struct",
	NotPointerToFunc:      "not a pointer to func",
	NeedExecutorOrQuerier: "need to supply an Executor or Querier parameter",
	InvalidFirstParam:     "first parameter must be of type context.Context, Executor, or Querier",
	ChannelInputParam:     "no input parameter can be a channel",
	TooManyReturnValues:   "must return 0, 1, or 2 values",
	SecondReturnNotError:  "2nd output parameter must be of type error",
	FirstReturnIsChannel:  "1st output parameter cannot be a channel",
	ExecutorReturnType:    "the 1st output parameter of an Executor must be int64 or sql.Result",
	SQLResultWithQuerier:  "output parameters of type sql.Result must be combined with Executor",
	RowsMustBeNonNil:      "rows must be non-nil",
	NoValuesFromQuery:     "no values returned from query",
	ShouldNeverGetHere:    "should never get here",
}

// ValidationError is returned when a struct, function signature, or type passed
// to Build/ShouldBuild/BuildFunction fails validation, or when a runtime
// invariant is violated.
type ValidationError struct {
	Kind ValidationErrorKind
}

func (e ValidationError) Error() string {
	if msg, ok := validationMessages[e.Kind]; ok {
		return msg
	}
	return "unknown validation error"
}

// Is matches any ValidationError when target has AnyValidation kind,
// or matches the exact kind otherwise.
func (e ValidationError) Is(target error) bool {
	t, ok := target.(ValidationError)
	if !ok {
		return false
	}
	return t.Kind == AnyValidation || e.Kind == t.Kind
}

// QueryErrorKind identifies the specific query or parameter processing failure.
// The zero value (AnyQuery) acts as a wildcard for errors.Is matching.
type QueryErrorKind int

const (
	AnyQuery             QueryErrorKind = iota
	QueryNotFound                       // Name: the missing query name
	MissingClosingColon                 // Query: the full query string
	EmptyVariable                       // Position: byte offset of the empty ::<var>
	ParameterNotFound                   // Name: the parameter name
	NilParameterPath                    // Name: the parameter name
	InvalidParameterType                // Name: the parameter name; TypeKind: the actual kind
)

// QueryError is returned when a query string or its parameters cannot be
// processed (missing query, bad syntax, unknown parameter).
type QueryError struct {
	Kind     QueryErrorKind
	Name     string // query or parameter name
	Query    string // full query string (MissingClosingColon)
	Position int    // byte offset (EmptyVariable)
	TypeKind string // reflect.Kind string (InvalidParameterType)
}

func (e QueryError) Error() string {
	switch e.Kind {
	case QueryNotFound:
		return fmt.Sprintf("no query found for name %s", e.Name)
	case MissingClosingColon:
		return fmt.Sprintf("missing a closing : somewhere: %s", e.Query)
	case EmptyVariable:
		return fmt.Sprintf("empty variable declaration at position %d", e.Position)
	case ParameterNotFound:
		return fmt.Sprintf("query parameter %s cannot be found in the incoming parameters", e.Name)
	case NilParameterPath:
		return fmt.Sprintf("query parameter %s has a path, but the incoming parameter is nil", e.Name)
	case InvalidParameterType:
		return fmt.Sprintf("query parameter %s has a path, but the incoming parameter is not a map or a struct it is %s", e.Name, e.TypeKind)
	default:
		return "unknown query error"
	}
}

// Is matches any QueryError when target has AnyQuery kind,
// or matches the exact kind otherwise.
func (e QueryError) Is(target error) bool {
	t, ok := target.(QueryError)
	if !ok {
		return false
	}
	return t.Kind == AnyQuery || e.Kind == t.Kind
}

// IdentifierErrorKind identifies the specific identifier parsing failure.
// The zero value (AnyIdentifier) acts as a wildcard for errors.Is matching.
type IdentifierErrorKind int

const (
	AnyIdentifier                IdentifierErrorKind = iota
	SemicolonInIdentifier                            // "; is not allowed in an identifier"
	EmptyOrTrailingDotIdentifier                     // "identifiers cannot be empty or end with a ."
	MissingDotInIdentifier                           // ". missing between parts of an identifier"
	LeadingOrDoubleDotIdentifier                     // "identifier cannot start with . or have two . in a row"
	InvalidCharacterInIdentifier                     // "invalid character found in identifier"
)

// IdentifierError is returned when an identifier in a query parameter fails
// syntax validation.
type IdentifierError struct {
	Kind       IdentifierErrorKind
	Identifier string
}

func (e IdentifierError) Error() string {
	switch e.Kind {
	case SemicolonInIdentifier:
		return fmt.Sprintf("; is not allowed in an identifier: %s", e.Identifier)
	case EmptyOrTrailingDotIdentifier:
		return fmt.Sprintf("identifiers cannot be empty or end with a .: %s", e.Identifier)
	case MissingDotInIdentifier:
		return fmt.Sprintf(". missing between parts of an identifier: %s", e.Identifier)
	case LeadingOrDoubleDotIdentifier:
		return fmt.Sprintf("identifier cannot start with . or have two . in a row: %s", e.Identifier)
	case InvalidCharacterInIdentifier:
		return fmt.Sprintf("invalid character found in identifier: %s", e.Identifier)
	default:
		return "unknown identifier error"
	}
}

// Is matches any IdentifierError when target has AnyIdentifier kind,
// or matches the exact kind otherwise.
func (e IdentifierError) Is(target error) bool {
	t, ok := target.(IdentifierError)
	if !ok {
		return false
	}
	return t.Kind == AnyIdentifier || e.Kind == t.Kind
}
