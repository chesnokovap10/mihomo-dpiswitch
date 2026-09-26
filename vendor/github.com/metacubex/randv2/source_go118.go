//go:build !go1.19

package randv2

import (
	_ "unsafe" // for go:linkname
)

//go:linkname runtime_fastrand runtime.fastrand
func runtime_fastrand() uint32

func (runtimeSource) Uint64() uint64 {
	return (uint64(runtimefastrand()) << 32) | uint64(runtimefastrand())
}
