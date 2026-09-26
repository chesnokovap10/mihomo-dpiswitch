package bbolt

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isMmapUnsupported(err error) bool {
	return errors.Is(err, windows.ERROR_INVALID_FUNCTION) ||
		errors.Is(err, windows.ERROR_NOT_SUPPORTED)
}
