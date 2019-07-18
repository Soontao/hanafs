package fs

import (
	"sync"

	"github.com/billziss-gh/cgofuse/fuse"
)

// Directory type
type Directory struct {
	children *ConcurrentMap
}

// ConcurrentMap is goroutine safe map
type ConcurrentMap = sync.Map

// StatProvider provide the file stat related information
type StatProvider func(string) (*fuse.Stat_t, error)

// DirectoryProvider provide the directory and child information
type DirectoryProvider func(string) (*Directory, error)

// FileSizeProvider provide the file size information
type FileSizeProvider func(string) int64
