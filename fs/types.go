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

// FileSystemStatWrapper type
type FileSystemStatWrapper struct {
	// Whole path (with base directory, start with '/' )
	Path string
	// stat infomation
	Stat *FileSystemStat
}

// NewFileSystemStatWrapper constructor
func NewFileSystemStatWrapper(path string, stat *FileSystemStat) *FileSystemStatWrapper {
	return &FileSystemStatWrapper{
		Path: path,
		Stat: stat,
	}
}

// FileSystemStat store the stat of file node
type FileSystemStat = fuse.Stat_t

// StatProvider provide the file stat related information
type StatProvider func(string) (*FileSystemStat, error)

// DirectoryProvider provide the directory and child information
type DirectoryProvider func(string, int64) ([]*FileSystemStatWrapper, error)

// FileSizeProvider provide the file size information
type FileSizeProvider func(string) int64
