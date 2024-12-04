package workload

import (
	"reflect"
	"sync"
)

var (
	headerRegister sync.Map
)

// indexOf returns the index of the specified string in the specified string slice.
//
// If the specified string is not present within the string slice, then this will return -1.
func indexOf(arr []string, target string) int {
	for index, value := range arr {
		if value == target {
			return index
		}
	}

	return -1
}

// removeIndex removes the value at the specified index from the specified slice.
func removeIndex(s []string, index int) []string {
	ret := make([]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// Extract the values from a map with arbitrary key and value types.
func getMapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, len(m))

	for _, v := range m {
		values = append(values, v)
	}

	return values
}

// maxInt returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	} else if b > a {
		return b
	}

	return a
}

type CSVHeaderProvider interface {
	MarshalCSVHeader(tag string) string
}

func PatchCSVHeader(s interface{}) {
	v := reflect.ValueOf(s).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		v := reflect.TypeOf((*CSVHeaderProvider)(nil)).Elem()
		if field.Type().Implements(v) {
			csvTag := fieldType.Tag.Get("csv")
			if csvTag != "" && csvTag != "-" {
				headerRegister.Store(csvTag, field.Interface())
			}
		}
	}
}
