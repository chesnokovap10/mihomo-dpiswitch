//go:build go1.21

package kcp

func clearSlice[S ~[]E, E any](s S) {
	clear(s)
}
