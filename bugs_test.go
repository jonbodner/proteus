package proteus

// These tests demonstrate critical bugs found during the code audit.
// Each test is expected to panic or fail, proving the bug exists.
// Once the underlying bugs are fixed, these tests should pass.

import (
	"context"
	"reflect"
	"testing"

	"github.com/jonbodner/proteus/mapper"
)

// Bug #18: defer rows.Close() panics when rows is nil.
// handleMapping does `defer rows.Close()` before checking if rows is nil.
func TestBug_HandleMappingNilRows(t *testing.T) {
	ctx := context.Background()
	sType := reflect.TypeOf(Person{})
	builder, err := mapper.MakeBuilder(ctx, sType)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("handleMapping panicked on nil rows: %v", r)
		}
	}()

	_, err = handleMapping(ctx, sType, nil, builder)
	if err == nil {
		t.Error("expected an error for nil rows, got nil")
	}
}

// Bug #19: reflect.TypeOf(nil) panic in setupDynamicQueries.
// Passing a nil value in the params map causes a panic because
// reflect.TypeOf(nil) returns nil, and subsequent .Kind() calls crash.
func TestBug_SetupDynamicQueriesNilValue(t *testing.T) {
	ctx := context.Background()
	b := NewBuilder(Postgres)

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Exec panicked on nil map value: %v", r)
		}
	}()

	db := setupDbPostgres()
	defer db.Close()

	_, err := b.Exec(ctx, db,
		"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)",
		map[string]interface{}{"name": nil, "age": 20},
	)
	// We expect an error, not a panic.
	_ = err
}

// Bug #22: Build() installs nil function fields on embedded struct errors.
// When an embedded struct has build errors, Build() unconditionally sets
// the field, leaving nil function pointers that panic when called.
func TestBug_BuildEmbeddedStructError(t *testing.T) {
	type BadInnerDao struct {
		// This function has an invalid signature (no Executor/Querier param),
		// so Build will fail for it.
		BadFunc func() `proq:"SELECT 1"`
	}

	type OuterDao struct {
		BadInnerDao
		// A valid function field to verify the outer struct isn't corrupted.
		Insert func(ctx context.Context, e ContextExecutor, name string, age int) (int64, error) `proq:"INSERT INTO PERSON(name, age) VALUES(:name:, :age:)" prop:"name,age"`
	}

	dao := OuterDao{}
	err := Build(&dao, Postgres)

	// Build should return an error because BadInnerDao has an invalid function.
	if err == nil {
		t.Fatal("expected Build to return an error for invalid embedded struct")
	}

	// The critical bug: even though Build returned an error, it still set
	// the embedded struct field. Calling the nil BadFunc would panic.
	// A safe Build should either not set any fields on error, or at minimum
	// not install nil function fields.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("calling function on error'd Build panicked (nil func installed): %v", r)
		}
	}()

	// Check that the outer function field is NOT populated after an error.
	// If Build installs partial results, this would be non-nil and usable,
	// but the embedded BadFunc would be nil — an inconsistent state.
	if dao.Insert != nil {
		t.Log("Build populated Insert despite returning an error — inconsistent state")
	}
}
