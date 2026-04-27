package rag

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

type assertionHelper struct{}

var testAssert assertionHelper

// assertNoError fails the test when err is non-nil.
func assertNoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NoError(t, err, msgAndArgs...)
}

// assertError fails the test when err is nil.
func assertError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Error(t, err, msgAndArgs...)
}

// assertEqual fails the test when want and got differ.
func assertEqual(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Equal(t, want, got, msgAndArgs...)
}

// assertNotEqual fails the test when want and got match.
func assertNotEqual(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotEqual(t, want, got, msgAndArgs...)
}

// assertTrue fails the test when value is false.
func assertTrue(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.True(t, value, msgAndArgs...)
}

// assertTruef fails the test when value is false and formats the supplied message.
func assertTruef(t testing.TB, value bool, msg string, args ...any) bool {
	t.Helper()
	return testAssert.Truef(t, value, msg, args...)
}

// assertFalse fails the test when value is true.
func assertFalse(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.False(t, value, msgAndArgs...)
}

// assertNil fails the test when value is not nil.
func assertNil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Nil(t, value, msgAndArgs...)
}

// assertNotNil fails the test when value is nil.
func assertNotNil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotNil(t, value, msgAndArgs...)
}

// assertLen fails the test when value does not expose the expected length.
func assertLen(t testing.TB, value any, want int, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Len(t, value, want, msgAndArgs...)
}

// assertContains fails the test when value does not contain element.
func assertContains(t testing.TB, value any, element any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Contains(t, value, element, msgAndArgs...)
}

// assertNotContains fails the test when value contains element.
func assertNotContains(t testing.TB, value any, element any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotContains(t, value, element, msgAndArgs...)
}

// assertEmpty fails the test when value is not empty.
func assertEmpty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Empty(t, value, msgAndArgs...)
}

// assertNotEmpty fails the test when value is empty.
func assertNotEmpty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotEmpty(t, value, msgAndArgs...)
}

// assertNotEmptyf fails the test when value is empty and formats the supplied message.
func assertNotEmptyf(t testing.TB, value any, msg string, args ...any) bool {
	t.Helper()
	return testAssert.NotEmptyf(t, value, msg, args...)
}

// assertGreater fails the test when got is not greater than want.
func assertGreater(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Greater(t, got, want, msgAndArgs...)
}

// assertGreaterf fails the test when got is not greater than want and formats the supplied message.
func assertGreaterf(t testing.TB, got any, want any, msg string, args ...any) bool {
	t.Helper()
	return testAssert.Greaterf(t, got, want, msg, args...)
}

// assertGreaterOrEqual fails the test when got is less than want.
func assertGreaterOrEqual(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.GreaterOrEqual(t, got, want, msgAndArgs...)
}

// assertLess fails the test when got is not less than want.
func assertLess(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Less(t, got, want, msgAndArgs...)
}

// assertLessOrEqual fails the test when got is greater than want.
func assertLessOrEqual(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.LessOrEqual(t, got, want, msgAndArgs...)
}

// assertInDelta fails the test when got is outside delta from want.
func assertInDelta(t testing.TB, want any, got any, delta any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.InDelta(t, want, got, delta, msgAndArgs...)
}

// NoError reports whether err is nil.
func (assertionHelper) NoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err != nil {
		return failf(t, "unexpected error: %v", []any{err}, msgAndArgs...)
	}
	return true
}

// Error reports whether err is non-nil.
func (assertionHelper) Error(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err == nil {
		return failf(t, "expected error, got nil", nil, msgAndArgs...)
	}
	return true
}

// Equal reports whether want and got are deeply equal.
func (assertionHelper) Equal(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		return failf(t, "want %v, got %v", []any{want, got}, msgAndArgs...)
	}
	return true
}

// NotEqual reports whether want and got differ.
func (assertionHelper) NotEqual(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	if reflect.DeepEqual(want, got) {
		return failf(t, "expected values to differ, both were %v", []any{got}, msgAndArgs...)
	}
	return true
}

// True reports whether value is true.
func (assertionHelper) True(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if !value {
		return failf(t, "expected true", nil, msgAndArgs...)
	}
	return true
}

// Truef reports whether value is true and formats the supplied message on failure.
func (h assertionHelper) Truef(t testing.TB, value bool, msg string, args ...any) bool {
	t.Helper()
	return h.True(t, value, append([]any{msg}, args...)...)
}

// False reports whether value is false.
func (assertionHelper) False(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if value {
		return failf(t, "expected false", nil, msgAndArgs...)
	}
	return true
}

// Nil reports whether value is nil, including typed nils.
func (assertionHelper) Nil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if !isNil(value) {
		return failf(t, "expected nil, got %v", []any{value}, msgAndArgs...)
	}
	return true
}

// NotNil reports whether value is not nil, including typed nils.
func (assertionHelper) NotNil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if isNil(value) {
		return failf(t, "expected non-nil", nil, msgAndArgs...)
	}
	return true
}

// Len reports whether value has the expected length.
func (assertionHelper) Len(t testing.TB, value any, want int, msgAndArgs ...any) bool {
	t.Helper()
	got, ok := lengthOf(value)
	if !ok {
		return failf(t, "expected value with length, got %T", []any{value}, msgAndArgs...)
	}
	if got != want {
		return failf(t, "want length %d, got %d", []any{want, got}, msgAndArgs...)
	}
	return true
}

// Contains reports whether value contains element.
func (assertionHelper) Contains(t testing.TB, value any, element any, msgAndArgs ...any) bool {
	t.Helper()
	found, ok := contains(value, element)
	if !ok {
		return failf(t, "expected %T to support contains", []any{value}, msgAndArgs...)
	}
	if !found {
		return failf(t, "expected %v to contain %v", []any{value, element}, msgAndArgs...)
	}
	return true
}

// NotContains reports whether value does not contain element.
func (assertionHelper) NotContains(t testing.TB, value any, element any, msgAndArgs ...any) bool {
	t.Helper()
	found, ok := contains(value, element)
	if !ok {
		return failf(t, "expected %T to support contains", []any{value}, msgAndArgs...)
	}
	if found {
		return failf(t, "expected %v not to contain %v", []any{value, element}, msgAndArgs...)
	}
	return true
}

// Empty reports whether value is empty.
func (assertionHelper) Empty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if !isEmpty(value) {
		return failf(t, "expected empty, got %v", []any{value}, msgAndArgs...)
	}
	return true
}

// NotEmpty reports whether value is not empty.
func (assertionHelper) NotEmpty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if isEmpty(value) {
		return failf(t, "expected non-empty", nil, msgAndArgs...)
	}
	return true
}

// NotEmptyf reports whether value is not empty and formats the supplied message on failure.
func (h assertionHelper) NotEmptyf(t testing.TB, value any, msg string, args ...any) bool {
	t.Helper()
	return h.NotEmpty(t, value, append([]any{msg}, args...)...)
}

// Greater reports whether got is greater than want.
func (assertionHelper) Greater(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	cmp, ok := compareOrdered(got, want)
	if !ok {
		return failf(t, "expected ordered values, got %T and %T", []any{got, want}, msgAndArgs...)
	}
	if cmp <= 0 {
		return failf(t, "expected %v to be greater than %v", []any{got, want}, msgAndArgs...)
	}
	return true
}

// Greaterf reports whether got is greater than want and formats the supplied message on failure.
func (h assertionHelper) Greaterf(t testing.TB, got any, want any, msg string, args ...any) bool {
	t.Helper()
	return h.Greater(t, got, want, append([]any{msg}, args...)...)
}

// GreaterOrEqual reports whether got is greater than or equal to want.
func (assertionHelper) GreaterOrEqual(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	cmp, ok := compareOrdered(got, want)
	if !ok {
		return failf(t, "expected ordered values, got %T and %T", []any{got, want}, msgAndArgs...)
	}
	if cmp < 0 {
		return failf(t, "expected %v to be greater than or equal to %v", []any{got, want}, msgAndArgs...)
	}
	return true
}

// Less reports whether got is less than want.
func (assertionHelper) Less(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	cmp, ok := compareOrdered(got, want)
	if !ok {
		return failf(t, "expected ordered values, got %T and %T", []any{got, want}, msgAndArgs...)
	}
	if cmp >= 0 {
		return failf(t, "expected %v to be less than %v", []any{got, want}, msgAndArgs...)
	}
	return true
}

// LessOrEqual reports whether got is less than or equal to want.
func (assertionHelper) LessOrEqual(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	cmp, ok := compareOrdered(got, want)
	if !ok {
		return failf(t, "expected ordered values, got %T and %T", []any{got, want}, msgAndArgs...)
	}
	if cmp > 0 {
		return failf(t, "expected %v to be less than or equal to %v", []any{got, want}, msgAndArgs...)
	}
	return true
}

// InDelta reports whether got is within delta of want.
func (assertionHelper) InDelta(t testing.TB, want any, got any, delta any, msgAndArgs ...any) bool {
	t.Helper()
	wantNumber, wantOK := number(want)
	gotNumber, gotOK := number(got)
	deltaNumber, deltaOK := number(delta)
	if !wantOK || !gotOK || !deltaOK {
		return failf(t, "expected numeric values, got %T, %T, and %T", []any{want, got, delta}, msgAndArgs...)
	}
	if math.Abs(wantNumber-gotNumber) > deltaNumber {
		return failf(t, "expected %v and %v to differ by at most %v", []any{want, got, delta}, msgAndArgs...)
	}
	return true
}

// failf records a fatal assertion failure with optional caller context.
func failf(t testing.TB, format string, args []any, msgAndArgs ...any) bool {
	t.Helper()
	detail := format
	if len(args) > 0 {
		detail = fmt.Sprintf(format, args...)
	}
	if msg := assertionMessage(msgAndArgs...); msg != "" {
		t.Fatalf("%s: %s", msg, detail)
	}
	t.Fatal(detail)
	return false
}

// assertionMessage formats optional assertion context.
func assertionMessage(msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	format, ok := msgAndArgs[0].(string)
	if !ok {
		return fmt.Sprint(msgAndArgs...)
	}
	if len(msgAndArgs) == 1 {
		return format
	}
	return fmt.Sprintf(format, msgAndArgs[1:]...)
}

// isNil reports whether value is nil, including typed nil values.
func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// isEmpty reports whether value is the zero or empty form of its type.
func isEmpty(value any) bool {
	if isNil(value) {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	default:
		return reflect.DeepEqual(value, reflect.Zero(v.Type()).Interface())
	}
}

// lengthOf returns the length of supported collection-like values.
func lengthOf(value any) (int, bool) {
	if value == nil {
		return 0, false
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Map, reflect.Slice:
		if v.IsNil() {
			return 0, false
		}
		return v.Len(), true
	case reflect.Array, reflect.String:
		return v.Len(), true
	case reflect.Func, reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return 0, false
		}
		return 0, false
	default:
		return 0, false
	}
}

// contains reports whether a string, slice, array, or map contains element.
func contains(value any, element any) (bool, bool) {
	if value == nil {
		return false, true
	}
	if s, ok := value.(string); ok {
		sub, ok := element.(string)
		return ok && strings.Contains(s, sub), ok
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		sub, ok := element.(string)
		return ok && strings.Contains(v.String(), sub), ok
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if reflect.DeepEqual(v.Index(i).Interface(), element) {
				return true, true
			}
		}
		return false, true
	case reflect.Map:
		key := reflect.ValueOf(element)
		if !key.IsValid() {
			return false, true
		}
		keyType := v.Type().Key()
		if key.Type().AssignableTo(keyType) {
			return v.MapIndex(key).IsValid(), true
		}
		if key.Type().ConvertibleTo(keyType) {
			return v.MapIndex(key.Convert(keyType)).IsValid(), true
		}
		return false, true
	default:
		return false, false
	}
}

// compareOrdered compares strings and numeric values.
func compareOrdered(left any, right any) (int, bool) {
	if leftString, ok := left.(string); ok {
		rightString, ok := right.(string)
		if !ok {
			return 0, false
		}
		return strings.Compare(leftString, rightString), true
	}

	leftNumber, leftOK := number(left)
	rightNumber, rightOK := number(right)
	if !leftOK || !rightOK {
		return 0, false
	}
	switch {
	case leftNumber < rightNumber:
		return -1, true
	case leftNumber > rightNumber:
		return 1, true
	default:
		return 0, true
	}
}

// number converts supported numeric values to float64 for comparisons.
func number(value any) (float64, bool) {
	if value == nil {
		return 0, false
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return v.Convert(reflect.TypeOf(float64(0))).Float(), true
	default:
		return 0, false
	}
}
