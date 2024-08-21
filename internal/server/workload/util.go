package workload

// Extract the values from a map with arbitrary key and value types.
func getMapValues[K comparable, V any](m map[K]V) []V {
	values := make([]V, len(m))

	for _, v := range m {
		values = append(values, v)
	}

	return values
}
