//go:build !go1.21

package json

import "errors"

var ErrUnsupported = errors.New("unsupported operation")
