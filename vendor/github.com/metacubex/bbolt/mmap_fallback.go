package bbolt

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/metacubex/bbolt/internal/common"
)

func mmapFallback(db *DB, sz int) error {
	b := make([]byte, sz)

	info, err := db.file.Stat()
	if err != nil {
		return fmt.Errorf("file stat: %w", err)
	}

	readSize := int64(sz)
	if info.Size() < readSize {
		readSize = info.Size()
	}
	if readSize > 0 {
		if _, err := db.file.ReadAt(b[:readSize], 0); err != nil && err != io.EOF {
			return fmt.Errorf("file read: %w", err)
		}
	}

	db.dataref = b
	db.data = (*[common.MaxMapSize]byte)(unsafe.Pointer(&b[0]))
	db.datasz = sz
	db.mmapFallback = true
	return nil
}

func munmapFallback(db *DB) error {
	db.dataref = nil
	db.data = nil
	db.datasz = 0
	db.mmapFallback = false
	return nil
}

func (db *DB) copyToMmapFallback(b []byte, off int64) {
	if off < 0 || off >= int64(len(db.dataref)) {
		return
	}
	copy(db.dataref[off:], b)
}
