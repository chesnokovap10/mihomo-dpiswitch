//go:build plan9

package bbolt

func isMmapUnsupported(error) bool {
	return false
}
