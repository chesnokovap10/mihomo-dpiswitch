//go:build go1.19 && !go1.22

package randv2

import (
	_ "unsafe" // for go:linkname
)

//go:linkname runtimefastrand64 runtime.fastrand64
func runtimefastrand64() uint64

func (runtimeSource) Uint64() uint64 {
	return runtimefastrand64()
}
