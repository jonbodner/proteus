package mapper

// These tests demonstrate critical bugs found during the code audit.
// Each test is expected to panic or fail, proving the bug exists.
// Once the underlying bugs are fixed, these tests should pass.

import (
	"context"
	"reflect"
	"testing"
)

// Bug #17: unsafe.Pointer(nil) creates a nil-pointer trap in ptrConverter.
// When a DB column is NULL and the output type is a pointer, ptrConverter
// wraps a nil pointer in a non-nil interface via reflect.NewAt(sType, unsafe.Pointer(nil)).
// The returned value appears non-nil but dereferences to address 0 — a panic waiting to happen.
func TestBug_PtrConverterNilPointerTrap(t *testing.T) {
	ctx := context.Background()

	// Build a mapper for *int (pointer to int).
	sType := reflect.TypeFor[*int]()
	builder, err := MakeBuilder(ctx, sType)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a NULL database value: vals contains a *interface{} pointing to nil.
	var nilVal interface{} = nil
	cols := []string{"value"}
	vals := []interface{}{&nilVal}

	result, err := builder(cols, vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The bug: result is a non-nil interface wrapping a nil pointer at address 0.
	// A correct implementation should return a typed nil pointer (*int)(nil),
	// which is also a non-nil interface but safe to check with == nil on the concrete type.
	if result == nil {
		// This is actually the safe outcome, meaning the bug might not manifest here.
		t.Log("result is nil interface — safe")
		return
	}

	// If result is non-nil, it should be a *int. Check if the concrete pointer is nil.
	ptrVal, ok := result.(*int)
	if !ok {
		t.Fatalf("expected *int, got %T", result)
	}

	if ptrVal != nil {
		t.Fatalf("expected nil *int for NULL column, got %v", *ptrVal)
	}

	// The key test: ptrVal should be a normal Go nil pointer, not an unsafe pointer to address 0.
	// Verify that reflect sees it as a proper nil pointer.
	rv := reflect.ValueOf(result)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		t.Errorf("result is a non-nil reflect.Ptr but concrete value is nil — this is the unsafe.Pointer(nil) trap")
	}
}

// Bug #20: fromPtr(nil) panics because reflect.TypeOf(nil) returns nil
// and calling .Kind() on it causes a nil pointer dereference.
func TestBug_FromPtrNil(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("fromPtr(nil) panicked: %v", r)
		}
	}()

	result := fromPtr(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

// Bug #20 (via Extract): Extract with a nil value and multi-segment path
// triggers fromPtr(nil) which panics.
func TestBug_ExtractNilMultiSegmentPath(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Extract panicked on nil value with multi-segment path: %v", r)
		}
	}()

	// A multi-segment path causes Extract to call fromPtr on the value.
	// If the value is nil, fromPtr panics.
	_, err := Extract(ctx, nil, []string{"root", "child"})
	if err == nil {
		t.Error("expected an error for nil value with multi-segment path, got nil")
	}
}

// Bug #21: Unbounded slice index in Extract causes panic.
// sv.Index(pos) panics when pos >= len(slice) with no bounds checking.
func TestBug_ExtractSliceIndexOutOfBounds(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Extract panicked on out-of-bounds slice index: %v", r)
		}
	}()

	data := []string{"a", "b", "c"}
	// Try to access index 10, which is out of bounds.
	_, err := Extract(ctx, data, []string{"data", "10"})
	if err == nil {
		t.Error("expected an error for out-of-bounds index, got nil")
	}
}

// Bug #21 (boundary): Negative index should also be caught.
func TestBug_ExtractSliceNegativeIndex(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Extract panicked on negative slice index: %v", r)
		}
	}()

	data := []string{"a", "b", "c"}
	// strconv.Atoi("-1") succeeds, so this reaches sv.Index(-1) which panics.
	_, err := Extract(ctx, data, []string{"data", "-1"})
	if err == nil {
		t.Error("expected an error for negative index, got nil")
	}
}
