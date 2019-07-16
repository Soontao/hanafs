package fs

import (
	"strings"

	"github.com/billziss-gh/cgofuse/fuse"
)

// ToFuseTimeStamp type
func ToFuseTimeStamp(timestamp int64) *fuse.Timespec {

	return &fuse.Timespec{
		Sec:  timestamp / 1000,
		Nsec: 0,
	}

}

func normalizePath(p string) string {
	if len(p) == 0 {
		p = "/"
	}
	// normalize path with windows
	return strings.ReplaceAll(strings.ReplaceAll(p, "\\", "/"), "\\/", "/")
}
