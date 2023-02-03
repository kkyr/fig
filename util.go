package fig

import (
	"os"
	"reflect"
	"strings"
	"time"
)

// stringSlice converts a Go slice represented as a string
// into an actual slice. The enclosing square brackets
// are not necessary.
// fields should be separated by a comma.
//
//	"[1,2,3]"     --->   []string{"1", "2", "3"}
//	" foo , bar"  --->   []string{" foo ", " bar"}
func stringSlice(s string) []string {
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	return strings.Split(s, ",")
}

// fileExists returns true if the file exists and is not a
// directory.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// isStructPtr reports whether i is a pointer to a struct.
func isStructPtr(i interface{}) bool {
	v := reflect.ValueOf(i)
	return v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct
}

// isZero reports whether v is its zero value for its type.
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Struct:
		if t, ok := v.Interface().(time.Time); ok {
			return t.IsZero()
		}
		return false
	case reflect.Invalid:
		return true
	default:
		return v.IsZero()
	}
}
