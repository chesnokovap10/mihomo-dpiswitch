//go:build plan9

package bbolt

import "errors"

// mlock locks memory of db file
func mlock(_ *DB, _ int) error {
	return errors.New("mlock is not supported on plan9")
}

// munlock unlocks memory of db file
func munlock(_ *DB, _ int) error {
	return errors.New("munlock is not supported on plan9")
}
