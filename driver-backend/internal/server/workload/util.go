package workload

import (
	"github.com/zhangjyr/gocsv"
	"reflect"
	"sync"
)

var (
	headerRegister sync.Map
)

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

func init() {
	gocsv.SetHeaderNormalizer(func(s string) string {
		if v, ok := headerRegister.Load(s); ok {
			if v, ok := v.(CSVHeaderProvider); ok {
				return v.MarshalCSVHeader(s)
			}
		}
		return s
	})
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
