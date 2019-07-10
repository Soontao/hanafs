package fs

import (
	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
)

// HanaFS type
type HanaFS struct {
	fuse.FileSystemBase
	client *hana.Client
}

func (f *HanaFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	hanaStat, err := f.client.Stat(path)
	if err != nil {
		return -fuse.ENOENT
	}
	s := &fuse.Stat_t{}

	if hanaStat.Directory {
		s.Mode = fuse.S_IFDIR
	} else {
		s.Mode = fuse.S_IFREG // Regular File.
	}

	return 0
}
