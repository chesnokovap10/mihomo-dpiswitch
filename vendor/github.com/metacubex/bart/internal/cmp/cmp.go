// Package cmp provides the ordered comparison needed by Go versions before
// the standard-library cmp package was introduced.
package cmp

// Ordered is the set of types that support the <, <=, >=, and > operators.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// Compare returns -1, 0, or 1 depending on whether x is less than, equal to,
// or greater than y. NaN values compare before non-NaN values.
func Compare[T Ordered](x, y T) int {
	xNaN := x != x
	yNaN := y != y
	if x < y || (xNaN && !yNaN) {
		return -1
	}
	if x > y || (!xNaN && yNaN) {
		return 1
	}
	return 0
}
