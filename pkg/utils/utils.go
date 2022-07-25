// Package utils holds logic for utility functions
package utils

// GetMapKeys is a generic utility function for copying map keys into a slice
func GetMapKeys[K comparable, V any](m map[K]V) []K {
	var keys []K
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
