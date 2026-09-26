//go:build !darwin
// +build !darwin

package guts

import "golang.org/x/sys/cpu"

var (
	haveAVX2   = cpu.X86.HasAVX2
	haveAVX512 = cpu.X86.HasAVX512F
)
