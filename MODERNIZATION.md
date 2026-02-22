# Proteus Modernization Suggestions

This document describes changes to bring the Proteus codebase up to modern, idiomatic Go (1.21+).

---

## 1. Replace `interface{}` with `any` (Go 1.18+)

Go 1.18 introduced `any` as a built-in alias for `interface{}`. The entire codebase uses `interface{}` — every occurrence should be replaced with `any` for consistency with modern Go style and the standard library itself.

**Scope:** Every `.go` file. Key locations include:

- `api.go` — all interface method signatures (`args ...interface{}`)
- `proteus.go` — `ShouldBuild`/`Build` parameters (`dao interface{}`)
- `proteus_function.go` — `BuildFunction`, `Query`, `Exec` parameters
- `runner.go` — `buildQueryArgs` return type, `handleMapping`, `mapRows`
- `mapper/mapper.go` — `Builder` type, `buildMap`, `buildStruct`, `buildPrimitive`
- `mapper/extract.go` — `Extract` signature
- `builder.go` — `sliceMap`, `doFinalize`

---

## 2. Replace `jonbodner/multierr` with `errors.Join` (Go 1.20+)

Go 1.20 added `errors.Join`, which collects multiple errors into a single error value. This is a direct replacement for the `multierr.Append` pattern used in `proteus.go`.

**Files affected:** `proteus.go` (lines 100, 116, 135, 142, 187, 201, 222, 229)

**Before:**
```go
import "github.com/jonbodner/multierr"
out = multierr.Append(out, err)
```

**After:**
```go
out = errors.Join(out, err)
```

This removes the `jonbodner/multierr` dependency entirely.

---

## 3. Replace `jonbodner/stackerr` with `fmt.Errorf` and `errors.New`

The `stackerr` package provides stack-trace-annotated errors — a pattern from before Go 1.13 added `%w` wrapping. Modern Go uses:

- `errors.New("message")` for simple sentinel errors
- `fmt.Errorf("context: %w", err)` for wrapping with context

**Files affected:** `proteus.go`, `builder.go`, `runner.go`, `mapper/mapper.go`, `mapper/extract.go`

**Before:**
```go
return stackerr.New("not a pointer")
return stackerr.Errorf("no query found for name %s", name)
```

**After:**
```go
return errors.New("not a pointer")
return fmt.Errorf("no query found for name %s", name)
```

For places where `stackerr` wraps an existing error, use `%w`:
```go
// Before
return nil, stackerr.Errorf("invalid index: %s :%w", path[1], err)
// After
return nil, fmt.Errorf("invalid index: %s: %w", path[1], err)
```

This removes the `jonbodner/stackerr` dependency entirely.

---

## 4. Fix `slog` Usage — Structured Logging Done Right

The recent migration to `slog` (commit c52e1f8) left behind anti-patterns. The code uses `slog.Log` with `fmt.Sprintln`/`fmt.Sprintf` to pre-format messages, which defeats the entire purpose of structured logging.

### 4a. Use `slog.DebugContext`/`WarnContext`/`ErrorContext` instead of `slog.Log`

**Before:**
```go
slog.Log(ctx, slog.LevelDebug, fmt.Sprintln("calling", finalQuery, "with params", queryArgs))
slog.Log(ctx, slog.LevelWarn, fmt.Sprintln("skipping function", curField.Name, "due to error:", err.Error()))
```

**After:**
```go
slog.DebugContext(ctx, "calling query", "query", finalQuery, "params", queryArgs)
slog.WarnContext(ctx, "skipping function", "function", curField.Name, "error", err)
```

### 4b. Use structured key-value attributes instead of `fmt.Sprintln`

Every `slog` call should have a static string message and pass dynamic data as key-value pairs. This enables filtering, machine parsing, and proper log aggregation.

**Files affected:** `runner.go` (lines 66, 92, 184, 232, 376), `proteus.go` (lines 202, 222, 229), `mapper/mapper.go` (lines 22, 151, 170, 172, 175, 187, 204, 209), `mapper/extract.go` (lines 68, 69, 71), `proteus_function.go` (lines 102, 119), `builder.go` (line 251)

Also fix the test files:
- `mapper/mapper_test.go` line 39: `slog.AnyValue(err)` is unnecessary — just pass `err` directly.

---

## 5. Use `strings.ReplaceAll` instead of `strings.Replace(..., -1)` (Go 1.12+)

`strings.ReplaceAll` is a clearer, more idiomatic function for replacing all occurrences.

**File:** `builder.go` lines 221-224

**Before:**
```go
name = strings.Replace(name, "DOT", "DOTDOT", -1)
name = strings.Replace(name, ".", "DOT", -1)
name = strings.Replace(name, "DOLLAR", "DOLLARDOLLAR", -1)
name = strings.Replace(name, "$", "DOLLAR", -1)
```

**After:**
```go
name = strings.ReplaceAll(name, "DOT", "DOTDOT")
name = strings.ReplaceAll(name, ".", "DOT")
name = strings.ReplaceAll(name, "DOLLAR", "DOLLARDOLLAR")
name = strings.ReplaceAll(name, "$", "DOLLAR")
```

Also applies to `bench_test.go` lines 229-230.

---

## 6. Use `strings.Builder` instead of `bytes.Buffer` for String Building

`strings.Builder` (Go 1.10+) is purpose-built for building strings and avoids the `[]byte` to `string` copy that `bytes.Buffer.String()` performs.

**Files affected:**
- `builder.go` line 67 — `var out bytes.Buffer` in `buildFixedQueryAndParamOrder`
- `builder.go` line 187 — `var b bytes.Buffer` in `doFinalize`
- `builder.go` line 207 — `var b bytes.Buffer` in `joinFactory`

Also, the `curVar` rune slice in `builder.go` line 76 (`curVar := []rune{}`) could be replaced with a `strings.Builder`, avoiding the `string(curVar)` conversion at line 93.

---

## ~~7. Use `defer rows.Close()` in `handleMapping`~~ (DONE)

Fixed in commit d8acfeb. `rows.Close()` is now deferred.

---

## ~~8. Eliminate `unsafe` Usage in `mapper/mapper.go`~~ (DONE)

Replaced `reflect.NewAt(sType, unsafe.Pointer(nil))` with `reflect.Zero(reflect.PointerTo(sType))`. The `unsafe` import has been removed from the mapper package.

---

## 9. Delete the `cmp/errors.go` Package

**File:** `cmp/errors.go`

This package contains a single function that compares errors by their `.Error()` string — a fragile anti-pattern. With proper use of sentinel errors, `errors.Is`, and `errors.As`, this package becomes unnecessary. The tests that use it should be updated to compare errors structurally.

---

## 10. Deprecation Annotations

### 10a. Use standard `// Deprecated:` format

Go tooling (godoc, gopls, IDE inspections) recognizes `// Deprecated:` as a standard marker. The current comment on `Wrap` doesn't follow this convention.

**File:** `wrapper.go`

**Before:**
```go
// Wrapper is now a no-op func that exists for backward compatibility. It is now deprecated and will be removed in the
// 1.0 release of proteus.
func Wrap(sqle Wrapper) Wrapper {
```

**After:**
```go
// Deprecated: Wrap is a no-op that exists for backward compatibility. Use ContextWrapper directly.
// It will be removed in the 1.0 release of proteus.
func Wrap(sqle Wrapper) Wrapper {
```

### 10b. Deprecate `Build` in favor of `ShouldBuild`

`Build` creates its own `context.Background()` internally, preventing callers from controlling context-dependent behavior (logging, cancellation). It should be marked deprecated:

```go
// Deprecated: Use ShouldBuild instead, which accepts a context.Context for logging control
// and does not populate function fields when errors are found.
func Build(dao any, paramAdapter ParamAdapter, mappers ...QueryMapper) error {
```

### 10c. Deprecate non-context interfaces

The `Executor`, `Querier`, and `Wrapper` interfaces lack context support. They should be marked deprecated in favor of `ContextExecutor`, `ContextQuerier`, and `ContextWrapper`:

```go
// Deprecated: Use ContextExecutor instead for context support.
type Executor interface { ... }

// Deprecated: Use ContextQuerier instead for context support.
type Querier interface { ... }

// Deprecated: Use ContextWrapper instead for context support.
type Wrapper interface { ... }
```

---

## 11. Update Dependencies

**File:** `go.mod`

Several dependencies are significantly out of date:

| Dependency | Current | Latest | Notes |
|---|---|---|---|
| `go-sql-driver/mysql` | v1.5.0 (2020) | v1.8+ | Major version behind |
| `google/go-cmp` | v0.4.0 | v0.6+ | Test-only dep |
| `jonbodner/dbtimer` | 2017 commit | - | Pinned to a 2017 commit hash |
| `jonbodner/multierr` | v1.0.0 | - | Replace with `errors.Join` (see #2) |
| `jonbodner/stackerr` | v1.0.0 | - | Replace with stdlib (see #3) |
| `pkg/profile` | v1.7.0 | - | Only used in `speed/speed.go`; consider removing or moving to a build-tagged file |

After removing `multierr` and `stackerr`, the dependency list shrinks significantly.

---

## 12. Testing Improvements

### 12a. Add `t.Helper()` to test helper functions

Test helpers that call `t.Fatalf`/`t.Errorf` should call `t.Helper()` so that failure messages report the caller's line number, not the helper's.

**Files affected:**
- `proteus_test.go` — `f()` (line 161), `fOk()` (line 175), many `doTest()` closures
- `mapper/extract_test.go` — `f()` helpers at lines 15, 77, 121
- `query_mappers_test.go` — `runMapper()` at line 72

### 12b. Fill in empty test tables

Several test functions have table-driven test infrastructure but empty test case slices with `// TODO` comments:

- `builder_test.go` — `Test_validateFunction`, `Test_buildDummyParameters`, `Test_convertToPositionalParameters`, `Test_joinFactory`, `Test_fixNameForTemplate`, `Test_addSlice`, `Test_validIdentifier`
- `runner_test.go` — `Test_getQArgs`, `Test_buildExec`, `Test_buildQuery`, `Test_handleMapping`
- `proteus_test.go` — `TestBuild`

These should either be populated with test cases or removed.

### 12c. Use `go-cmp` consistently instead of `reflect.DeepEqual`

The project already imports `google/go-cmp`. Some tests use `reflect.DeepEqual` instead — these should be migrated for consistent, readable diff output on failure.

**Files:** `builder_test.go` (lines 102, 132, 154), `runner_test.go` (lines 33, 55, 83, 110)

### 12d. Remove debug `fmt.Println` from tests

- `mapper/extract_test.go` lines 149, 155 — leftover `fmt.Println`/`fmt.Printf` debug output
- `template_test.go` line 28 — `fmt.Println("b:", b.String())`

### 12e. Fix `html/template` import in test

**File:** `template_test.go` line 6

The test imports `html/template` while the production code uses `text/template`. The test should use `text/template` for consistency.

### 12f. Add `t.Parallel()` to pure unit tests

Unit tests that don't touch databases or shared state can run in parallel for faster test execution.

---

## 13. Reduce Duplication Between Context and Non-Context Paths

The codebase has nearly-identical pairs of functions:

- `makeContextExecutorImplementation` / `makeExecutorImplementation`
- `makeContextQuerierImplementation` / `makeQuerierImplementation`

These differ only in whether they extract `context.Context` from `args[0]` and call `ExecContext`/`Exec`. Consider refactoring to reduce this duplication, or at minimum, if the non-context interfaces are deprecated (see #10c), mark the non-context implementations as deprecated too.

---

## 14. Clean Up the Package-Level Comment Block

**File:** `proteus.go` lines 15-52

This large block comment reads like development scratchpad notes (`"next:"`, `"later:"`, numbered implementation steps). It should either be:
- Converted to proper godoc for the `proteus` package (placed before the `package proteus` line in a `doc.go` file)
- Or removed, since the information is better suited for `CLAUDE.md` or a separate design document

---

## ~~15. Makefile Bug~~ (DONE)

Fixed in commit 10f239f. The `sample_ctx` target now runs `cmd/sample-ctx/main.go`, and `docker-compose` was updated to `docker compose`.

---

## 16. Consider Generics for Public API (Go 1.18+)

While the core of Proteus relies heavily on reflection (which generics can't fully replace), a few public APIs could benefit from generic type parameters for better type safety at the call site:

```go
// Before
func ShouldBuild(ctx context.Context, dao interface{}, ...) error

// After — enforces pointer-to-struct at compile time
func ShouldBuild[T any](ctx context.Context, dao *T, ...) error
```

Similarly for `BuildFunction`, `Query`, and `Exec` on the `Builder` type. This is a larger change that would affect the public API, so it warrants careful consideration of backward compatibility. A pragmatic approach would be to add generic wrapper functions alongside the existing ones and deprecate the `interface{}` versions.

---

## ~~17. `unsafe.Pointer(nil)` Causes Nil-Pointer Trap~~ (DONE)

Replaced `reflect.NewAt(sType, unsafe.Pointer(nil))` with `reflect.Zero(reflect.PointerTo(sType))` in `mapper/mapper.go`. Same fix as #8.

---

## ~~18. `defer rows.Close()` Panics When `rows` Is Nil~~ (DONE)

Added a nil guard before `defer rows.Close()` in `handleMapping`. Returns an error instead of panicking.

---

## ~~19. `reflect.TypeOf(nil)` Panic in `setupDynamicQueries`~~ (DONE)

Fixed by adding nil guards in the downstream code: `builder.go` checks for nil `paramType` before calling `.Kind()`, `mapper/extract.go` guards `ExtractType` and `fromPtrType` against nil types, and `runner.go` checks `value.IsValid()` before calling `.Interface()` in `buildQueryArgs`.

---

## ~~20. `fromPtr(nil)` Panics in `mapper/extract.go`~~ (DONE)

Added nil guard in `fromPtr`: checks `st != nil` before calling `.Kind()`. Also added nil guard in `fromPtrType` for the same reason.

---

## ~~21. Unbounded Slice Index in `mapper/extract.go`~~ (DONE)

Added bounds checking (`pos < 0 || pos >= sv.Len()`) before `sv.Index(pos)` in `Extract`. Returns an error instead of panicking.

---

## ~~22. `Build()` Sets Embedded Struct Fields Even on Error~~ (DONE)

Moved `daoValue.Field(i).Set(pv.Elem())` into the `else` branch so the field is only set when the inner `Build` succeeds.

---

## ~~23. `defer stmt.Close()` Lifetime Mismatch With Returned Rows~~ (NOT A BUG)

On closer analysis, this is actually correct. `buildRetVals` calls `handleMapping` synchronously within the same closure call frame. The rows are fully consumed before the closure returns and `defer stmt.Close()` fires. The call chain is: closure -> `buildRetVals` -> `handleMapping` (iterates/closes rows) -> returns -> `defer stmt.Close()` fires. The statement outlives the rows.

---

## 24. Missing `tx.Rollback()` on Error Paths

**Files:** `cmd/sample/main.go`, `cmd/sample-ctx/main.go`, various test files

The pattern used throughout is:
```go
tx, err := db.Begin()
if err != nil {
    panic(err)
}
defer tx.Commit()
```

`defer tx.Commit()` runs even when the function exits due to an error or panic, committing partial transactions. The correct pattern is:
```go
defer func() {
    if err != nil {
        tx.Rollback()
    } else {
        tx.Commit()
    }
}()
```

Note: `cmd/null/main.go` already uses the correct pattern.

---

## 25. Unchecked `Build()` Return Values in Samples/Benchmarks

**Files:** `speed/speed.go` line 144, `bench/bench_test.go` line 29

```go
proteus.Build(&productDao, proteus.Postgres)  // error silently discarded
```

If `Build` returns an error, `productDao` will have nil function fields. Subsequent calls to those fields panic with "invalid memory address or nil pointer dereference."

**Fix:** Check the error and fail/panic immediately.

---

## 26. `defer db.Close()` on Potentially-Nil `*sql.DB`

**Files:** `speed/speed.go` line 109, `cmd/sample/main.go`, `cmd/sample-ctx/main.go`

`setupDbPostgres` can return `nil` on schema creation error. The callers do `defer db.Close()` unconditionally, which panics if `db` is nil.

**Fix:** Check for nil before deferring, or have `setupDbPostgres` call `log.Fatal` on error instead of returning nil.

---

## Summary by Priority

**Critical (panic/corruption in library code) — ALL DONE:**
- ~~#17 — `unsafe.Pointer(nil)` nil-pointer trap in mapper~~ *(DONE: see #8)*
- ~~#18 — `defer rows.Close()` panics on nil rows~~ *(DONE)*
- ~~#19 — `reflect.TypeOf(nil)` panic in `setupDynamicQueries`~~ *(DONE)*
- ~~#20 — `fromPtr(nil)` panic in extract~~ *(DONE)*
- ~~#21 — Unbounded slice index panic in extract~~ *(DONE)*
- ~~#22 — `Build()` installs nil function fields on error~~ *(DONE)*
- ~~#23 — Statement/rows lifetime mismatch~~ *(NOT A BUG)*

**High priority (correctness/safety):**
- ~~#7 — `defer rows.Close()` (resource leak risk)~~ *(DONE)*
- ~~#8 — Remove `unsafe`~~ *(DONE)*
- ~~#15 — Makefile bug fix~~ *(DONE)*
- #24 — Missing `tx.Rollback()` in samples/tests

**Medium priority (idiomatic modernization):**
- #1 — `interface{}` to `any`
- #2 — Replace `multierr` with `errors.Join`
- #3 — Replace `stackerr` with stdlib error handling
- #4 — Fix slog usage for proper structured logging
- #11 — Update dependencies

**Lower priority (cleanup):**
- #5 — `strings.ReplaceAll`
- #6 — `strings.Builder`
- #9 — Delete `cmp/errors.go`
- #10 — Deprecation annotations
- #12 — Testing improvements
- #13 — Reduce duplication
- #14 — Clean up comments
- #25 — Check `Build()` return values in samples/benchmarks
- #26 — Nil-guard `db.Close()` in samples

**Future consideration:**
- #16 — Generics for public API
