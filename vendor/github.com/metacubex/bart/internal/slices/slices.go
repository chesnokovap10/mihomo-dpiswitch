// Package slices provides the subset of the standard-library slices package
// used by bart while retaining compatibility with Go 1.20.
package slices

import "sort"

// Equal reports whether two slices contain equal elements in the same order.
func Equal[S ~[]E, E comparable](s1, s2 S) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

// Delete removes s[i:j] and clears the obsolete elements.
func Delete[S ~[]E, E any](s S, i, j int) S {
	_ = s[i:j:len(s)]
	if i == j {
		return s
	}
	oldLen := len(s)
	s = append(s[:i], s[j:]...)
	var zero E
	tail := s[len(s):oldLen]
	for k := range tail {
		tail[k] = zero
	}
	return s
}

// Clone returns a shallow copy of s and preserves a nil input.
func Clone[S ~[]E, E any](s S) S {
	if s == nil {
		return nil
	}
	return append(S([]E{}), s...)
}

// Reverse reverses s in place.
func Reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Concat returns a new slice containing the concatenation of the supplied slices.
func Concat[S ~[]E, E any](ss ...S) S {
	size := 0
	for _, s := range ss {
		size += len(s)
		if size < 0 {
			panic("len out of range")
		}
	}
	var result S
	if size > 0 {
		result = make(S, 0, size)
	}
	for _, s := range ss {
		result = append(result, s...)
	}
	return result
}

// SortFunc sorts s according to cmp.
func SortFunc[S ~[]E, E any](s S, cmp func(a, b E) int) {
	sort.Slice(s, func(i, j int) bool { return cmp(s[i], s[j]) < 0 })
}

// IsSortedFunc reports whether s is sorted according to cmp.
func IsSortedFunc[S ~[]E, E any](s S, cmp func(a, b E) int) bool {
	for i := len(s) - 1; i > 0; i-- {
		if cmp(s[i], s[i-1]) < 0 {
			return false
		}
	}
	return true
}
