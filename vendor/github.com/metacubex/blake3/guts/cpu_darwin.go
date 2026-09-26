package guts

import (
	"syscall"

	"golang.org/x/sys/cpu"
)

var (
	haveAVX2   bool
	haveAVX512 bool
)

func init() {
	haveAVX2 = cpu.X86.HasAVX2
	haveAVX512 = cpu.X86.HasAVX512F
	if !haveAVX512 {
		// On some Macs, AVX512 detection is buggy, so fallback to sysctl
		b, _ := syscall.Sysctl("hw.optional.avx512f")
		haveAVX512 = len(b) > 0 && b[0] == 1
	}
}
