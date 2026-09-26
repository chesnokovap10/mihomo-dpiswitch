//go:build !go1.21

package kcp

func clearSlice[S ~[]E, E any](s S) {
	var zero E
	for i := range s {
		s[i] = zero
	}
}
