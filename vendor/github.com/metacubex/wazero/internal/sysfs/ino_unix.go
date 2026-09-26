//go:build unix

package sysfs

import (
	"io/fs"
	"syscall"

	experimentalsys "github.com/metacubex/wazero/experimental/sys"
	"github.com/metacubex/wazero/sys"
)

func inoFromFileInfo(_ string, info fs.FileInfo) (sys.Inode, experimentalsys.Errno) {
	switch v := info.Sys().(type) {
	case *sys.Stat_t:
		return v.Ino, 0
	case *syscall.Stat_t:
		return v.Ino, 0
	default:
		return 0, 0
	}
}
