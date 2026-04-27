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

func assertNoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NoError(t, err, msgAndArgs...)
}

func assertError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Error(t, err, msgAndArgs...)
}

func assertEqual(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Equal(t, want, got, msgAndArgs...)
}

func assertNotEqual(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotEqual(t, want, got, msgAndArgs...)
}

func assertTrue(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.True(t, value, msgAndArgs...)
}

func assertTruef(t testing.TB, value bool, msg string, args ...any) bool {
	t.Helper()
	return testAssert.Truef(t, value, msg, args...)
}

func assertFalse(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.False(t, value, msgAndArgs...)
}

func assertNil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Nil(t, value, msgAndArgs...)
}

func assertNotNil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotNil(t, value, msgAndArgs...)
}

func assertLen(t testing.TB, value any, want int, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Len(t, value, want, msgAndArgs...)
}

func assertContains(t testing.TB, value any, element any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Contains(t, value, element, msgAndArgs...)
}

func assertNotContains(t testing.TB, value any, element any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotContains(t, value, element, msgAndArgs...)
}

func assertEmpty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Empty(t, value, msgAndArgs...)
}

func assertNotEmpty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.NotEmpty(t, value, msgAndArgs...)
}

func assertNotEmptyf(t testing.TB, value any, msg string, args ...any) bool {
	t.Helper()
	return testAssert.NotEmptyf(t, value, msg, args...)
}

func assertGreater(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Greater(t, got, want, msgAndArgs...)
}

func assertGreaterf(t testing.TB, got any, want any, msg string, args ...any) bool {
	t.Helper()
	return testAssert.Greaterf(t, got, want, msg, args...)
}

func assertGreaterOrEqual(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.GreaterOrEqual(t, got, want, msgAndArgs...)
}

func assertLess(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.Less(t, got, want, msgAndArgs...)
}

func assertLessOrEqual(t testing.TB, got any, want any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.LessOrEqual(t, got, want, msgAndArgs...)
}

func assertInDelta(t testing.TB, want any, got any, delta any, msgAndArgs ...any) bool {
	t.Helper()
	return testAssert.InDelta(t, want, got, delta, msgAndArgs...)
}

func (assertionHelper) NoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err != nil {
		return failf(t, "unexpected error: %v", []any{err}, msgAndArgs...)
	}
	return true
}

func (assertionHelper) Error(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err == nil {
		return failf(t, "expected error, got nil", nil, msgAndArgs...)
	}
	return true
}

func (assertionHelper) Equal(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		return failf(t, "want %v, got %v", []any{want, got}, msgAndArgs...)
	}
	return true
}

func (assertionHelper) NotEqual(t testing.TB, want any, got any, msgAndArgs ...any) bool {
	t.Helper()
	if reflect.DeepEqual(want, got) {
		return failf(t, "expected values to differ, both were %v", []any{got}, msgAndArgs...)
	}
	return true
}

func (assertionHelper) True(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if !value {
		return failf(t, "expected true", nil, msgAndArgs...)
	}
	return true
}

func (h assertionHelper) Truef(t testing.TB, value bool, msg string, args ...any) bool {
	t.Helper()
	return h.True(t, value, append([]any{msg}, args...)...)
}

func (assertionHelper) False(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if value {
		return failf(t, "expected false", nil, msgAndArgs...)
	}
	return true
}

func (assertionHelper) Nil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if !isNil(value) {
		return failf(t, "expected nil, got %v", []any{value}, msgAndArgs...)
	}
	return true
}

func (assertionHelper) NotNil(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if isNil(value) {
		return failf(t, "expected non-nil", nil, msgAndArgs...)
	}
	return true
}

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

func (assertionHelper) Empty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if !isEmpty(value) {
		return failf(t, "expected empty, got %v", []any{value}, msgAndArgs...)
	}
	return true
}

func (assertionHelper) NotEmpty(t testing.TB, value any, msgAndArgs ...any) bool {
	t.Helper()
	if isEmpty(value) {
		return failf(t, "expected non-empty", nil, msgAndArgs...)
	}
	return true
}

func (h assertionHelper) NotEmptyf(t testing.TB, value any, msg string, args ...any) bool {
	t.Helper()
	return h.NotEmpty(t, value, append([]any{msg}, args...)...)
}

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

func (h assertionHelper) Greaterf(t testing.TB, got any, want any, msg string, args ...any) bool {
	t.Helper()
	return h.Greater(t, got, want, append([]any{msg}, args...)...)
}

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
