package fs

import (
	"github.com/billziss-gh/cgofuse/fuse"
)

// ToFuseTimeStamp type
func ToFuseTimeStamp(timestamp int64) *fuse.Timespec {

	return &fuse.Timespec{
		Sec:  timestamp / 1000,
		Nsec: 0,
	}

}
