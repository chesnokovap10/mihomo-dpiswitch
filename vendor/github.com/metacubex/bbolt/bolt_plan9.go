//go:build plan9

package bbolt

import "time"

func fdatasync(db *DB) error {
	return db.file.Sync()
}

func flock(_ *DB, _ bool, _ time.Duration) error {
	return nil
}

func funlock(_ *DB) error {
	return nil
}

func mmap(db *DB, sz int) error {
	return mmapFallback(db, sz)
}

func munmap(db *DB) error {
	return munmapFallback(db)
}
