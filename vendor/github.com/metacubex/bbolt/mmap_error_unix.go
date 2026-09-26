//go:build !windows && !plan9

package bbolt

import (
	"errors"

	"golang.org/x/sys/unix"
)

func isMmapUnsupported(err error) bool {
	return errors.Is(err, unix.ENOSYS) ||
		errors.Is(err, unix.ENODEV) ||
		errors.Is(err, unix.EOPNOTSUPP) ||
		errors.Is(err, unix.ENOTSUP) ||
		errors.Is(err, unix.EINVAL)
}
