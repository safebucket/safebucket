# Query Validator Middleware Analysis

## Overview

This document outlines bugs, weaknesses, and security concerns found in the query parameter validation middleware (
`internal/middlewares/query_validator.go` and `internal/helpers/query.go`).

## Critical Bugs

### 1. JSON Tag Options Not Stripped (Line 77)

**Severity**: High
**Location**: `query_validator.go:77`

**Issue**: The middleware uses `fieldType.Tag.Get("json")` to get the query parameter name, but doesn't strip tag
options like `,omitempty`, `,string`, etc.

**Example**:

```go
type Params struct {
Field1 string `json:"field1,omitempty"` // Looks for "field1,omitempty" not "field1"
Field2 string `json:"field2,string"`    // Looks for "field2,string" not "field2"
}
```

**Impact**: Fields with JSON tag options will never be parsed from query parameters because the middleware looks for the
wrong parameter names.

**Fix**: Split the JSON tag on comma and use only the first part:

```go
jsonTag := fieldType.Tag.Get("json")
if jsonTag != "" {
parts := strings.Split(jsonTag, ",")
queryParamName = parts[0]
}
```

**Test**: `TestValidateQueryJSONTagParsing/JSON_tag_with_options_-_BUG`

---

### 2. Validator Instance Created Per Request (Line 43)

**Severity**: Medium (Performance)
**Location**: `query_validator.go:43`

**Issue**: A new `validator.New()` instance is created for every single HTTP request.

**Code**:

```go
func ValidateQuery[T any](next http.Handler) http.Handler {
return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
// ...
validate := validator.New() // ← Created on every request
err = validate.Struct(data)
// ...
})
}
```

**Impact**:

- Unnecessary memory allocations
- CPU overhead for validator initialization
- Poor performance under high load

**Fix**: Create validator once as a package-level variable:

```go
var validate = validator.New()

func ValidateQuery[T any](next http.Handler) http.Handler {
return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
// ...
err = validate.Struct(data)
// ...
})
}
```

**Test**: `TestValidateQueryPerformance`

---

### 3. Int32 Overflow Not Detected (Line 123-127)

**Severity**: Medium
**Location**: `query_validator.go:123-127`

**Issue**: When parsing int32 fields, the code uses `ParseInt(value, 10, 64)` which returns int64, then calls `SetInt()`
which silently truncates values outside the int32 range.

**Code**:

```go
case reflect.Int, reflect.Int32, reflect.Int64:
intValue, err := strconv.ParseInt(value, 10, 64) // Always parses as int64
if err != nil {
return err
}
field.SetInt(intValue) // Silently truncates for int32
```

**Example**:

- Input: `?int32_field=2147483648` (2^31, above int32 max)
- Expected: Error
- Actual: Wraps to `-2147483648` (int32 min value)

**Impact**: Data corruption, potential security issues if used for limits or bounds checking.

**Fix**: Check bounds before calling SetInt:

```go
case reflect.Int32:
intValue, err := strconv.ParseInt(value, 10, 32) // Parse with 32-bit limit
if err != nil {
return err
}
field.SetInt(intValue)
```

**Test**: `TestValidateQueryInt32Overflow/Int32_overflow_(above_max)_-_BUG`

---

## Medium Severity Issues

### 4. Multiple Query Parameters - Only First Used (Line 83)

**Severity**: Medium
**Location**: `query_validator.go:83`

**Issue**: `queryParams.Get(queryParamName)` only returns the first value when multiple parameters with the same name
are provided.

**Example**:

- Request: `?status=uploaded&status=trashed`
- Expected: Error or array support
- Actual: Only "uploaded" is processed

**Impact**: Silent data loss, unexpected behavior, no support for array parameters.

**Current Behavior**: No array/slice support at all.

**Fix Options**:

1. Document this limitation clearly
2. Add support for `[]string` fields using `queryParams[queryParamName]`
3. Return error when duplicate parameters are detected

**Test**: `TestValidateQueryMultipleValues/Multiple_values_for_same_parameter_-_BUG`

---

### 5. No Support for Unsigned Integers (Line 119-143)

**Severity**: Low
**Location**: `query_validator.go:119-143`

**Issue**: The switch statement doesn't handle `uint`, `uint32`, `uint64` types.

**Impact**: Unsigned integer fields remain at zero value, silently ignored.

**Example**:

```go
type Params struct {
UintField uint `json:"uint_field"` // Never parsed
}
```

**Fix**: Add cases for unsigned integer types:

```go
case reflect.Uint, reflect.Uint32, reflect.Uint64:
uintValue, err := strconv.ParseUint(value, 10, 64)
if err != nil {
return err
}
field.SetUint(uintValue)
```

**Test**: `TestValidateQueryUnsupportedTypes/Unsupported_types_are_silently_skipped`

---

## Low Severity / Design Decisions

### 6. Empty String Values Silently Skipped (Line 84-86)

**Severity**: Low
**Location**: `query_validator.go:84-86`

**Issue**: Empty query parameter values are skipped entirely.

**Example**:

- Request: `?name=&status=active`
- Behavior: `name` field remains at zero value, validation might fail

**Consideration**: This might be intentional, but it means `?field=` is different from not providing the field at all
only for validation purposes.

**Impact**: Could be confusing for required fields with empty values.

**Test**: `TestValidateQueryEdgeCases/Empty_string_values_are_skipped`

---

### 7. Boolean Parsing Very Permissive (Line 128-133)

**Severity**: Low
**Location**: `query_validator.go:128-133`

**Issue**: `strconv.ParseBool` accepts many variations: "1", "t", "T", "TRUE", "true", "True", "0", "f", "F", "FALSE", "
false", "False"

**Consideration**: This is very permissive and might not be desired for API consistency.

**Recommendation**: Consider restricting to "true"/"false" only for stricter API contracts.

**Test**: `TestValidateQueryBoolVariations`

---

### 8. Unexported Fields Silently Skipped (Line 102, 114)

**Severity**: Low
**Location**: `query_validator.go:102, 114`

**Issue**: If `field.CanSet()` returns false (unexported fields), the function silently continues.

**Impact**: Configuration errors in struct definitions won't be caught.

**Recommendation**: Log a warning or return an error during development if unexported fields have JSON tags.

**Test**: Implicit in all tests (no unexported fields with tags)

---

### 9. Unsupported Types Silently Ignored (Line 140-142)

**Severity**: Low
**Location**: `query_validator.go:140-142`

**Issue**: Unsupported types (slices, maps, structs, complex numbers) silently return nil.

**Example**:

```go
type Params struct {
Tags []string `json:"tags"` // Silently ignored
}
```

**Impact**: No error feedback, fields remain at zero value.

**Recommendation**: Consider returning an error for unsupported types with JSON tags to catch configuration errors
early.

**Test**: `TestValidateQueryUnsupportedTypes/Unsupported_types_are_silently_skipped`

---

### 10. Error Messages Not User-Friendly (Line 50-54)

**Severity**: Low
**Location**: `query_validator.go:50-54`

**Issue**: Validation errors are very technical:

```
"Key: 'BasicQueryParams.Limit' Error:Field validation for 'Limit' failed on the 'min' tag"
```

**Impact**: Poor developer/user experience, exposes internal struct names.

**Recommendation**: Transform validation errors into user-friendly messages:

```go
"limit must be at least 1"
"status must be one of: active, inactive"
```

**Test**: All validation error tests show this

---

## Security Considerations

### 11. No Input Length Limits

**Severity**: Medium
**Location**: General

**Issue**: No limits on query parameter value lengths.

**Impact**: Potential DoS through very long query parameter values.

**Recommendation**: Add max length validation or truncation.

---

### 12. No Rate Limiting on Parsing Errors

**Severity**: Low
**Location**: General

**Issue**: No protection against repeated malformed requests.

**Impact**: Could be used for DoS by causing repeated parsing errors.

**Recommendation**: Implement rate limiting on parse errors per IP.

---

## Test Coverage

Comprehensive tests have been written to cover:

### Basic Functionality

- ✅ Valid query parameters (all types)
- ✅ Missing required fields
- ✅ Validation failures (oneof, min, max)
- ✅ Invalid type formats
- ✅ Empty query parameters

### Pointer Types

- ✅ All pointer fields provided
- ✅ No pointer fields provided
- ✅ Partial pointer fields
- ✅ Pointer validation failures

### Type Support

- ✅ String, int, int32, int64, bool, float32, float64
- ✅ Pointer versions of all types
- ✅ Negative numbers
- ✅ Zero values
- ✅ Scientific notation for floats
- ✅ Very large integers (int64 max)

### Edge Cases

- ✅ Empty string values
- ✅ URL-encoded special characters
- ✅ Unicode characters
- ✅ Multiple values for same parameter (bug confirmed)
- ✅ JSON tag options (bug confirmed)
- ✅ Unsupported types (silent skip confirmed)
- ✅ Int32 overflow (bug confirmed)
- ✅ Boolean variations (13 different formats accepted)

### Context Management

- ✅ Params stored in context correctly
- ✅ Invalid type assertion handling
- ✅ Missing context key handling

### Performance

- ✅ Validator instance creation overhead (bug confirmed)
- ✅ Benchmarks for basic and pointer types

## Recommendations Priority

### High Priority (Fix Immediately)

1. **Fix JSON tag parsing** - Critical for correct parameter parsing
2. **Reuse validator instance** - Significant performance impact

### Medium Priority (Fix Soon)

3. **Add int32 overflow checking** - Data integrity issue
4. **Add uint support** - Missing functionality
5. **Implement input length limits** - Security concern

### Low Priority (Consider for Next Iteration)

6. **Improve error messages** - UX improvement
7. **Add array/slice support** - Feature enhancement
8. **Document empty string behavior** - Documentation
9. **Stricter boolean parsing** - API consistency
10. **Warn on unsupported types** - Developer experience

## Benchmarks

```
BenchmarkValidateQueryMiddleware-12         32284       38438 ns/op    17766 B/op    229 allocs/op
BenchmarkValidateQueryWithPointers-12       29097       39408 ns/op    17916 B/op    235 allocs/op
```

**Analysis**:

- ~38-39 microseconds per request
- ~17-18 KB allocated per request
- ~229-235 allocations per request

Performance could be improved by ~10-15% by reusing the validator instance (reducing allocations from 229 to ~220).
