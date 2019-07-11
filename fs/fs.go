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

func (f *HanaFS) Open(path string, flags int) (errc int, fh uint64) {
	return 0, 0
}

func (f *HanaFS) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) int {

	dir, err := f.client.ReadDirectory(path)

	if err != nil {
		return -fuse.ENOENT
	}

	now := fuse.Now()
	uid, gid, _ := fuse.Getcontext()

	for _, hanaChild := range dir.Children {

		fsChildStat := &fuse.Stat_t{
			Flags: 0,
			Uid:   uid,
			Gid:   gid,
			Ctim:  now,
			Atim:  now,
			Mtim:  now,
		}

		if hanaChild.Directory {
			fsChildStat.Mode = fuse.S_IFDIR | fuse.S_IRWXU
		} else {
			fsChildStat.Mode = fuse.S_IFREG | fuse.S_IRWXU // Regular File.
		}

		fill(hanaChild.Name, fsChildStat, 0)

	}

	return 0
}

func (f *HanaFS) Getattr(path string, s *fuse.Stat_t, fh uint64) int {

	hanaStat, err := f.client.Stat(path)

	if err != nil {
		return -fuse.ENOENT
	}

	now := fuse.Now()
	uid, gid, _ := fuse.Getcontext()

	s.Gid = gid
	s.Uid = uid
	s.Ctim = now
	s.Atim = now
	s.Mtim = now

	if hanaStat.Directory {
		s.Mode = fuse.S_IFDIR | fuse.S_IRWXU
	} else {
		s.Mode = fuse.S_IFREG | fuse.S_IRWXU // Regular File.
	}

	return 0
}

// NewHanaFS type
func NewHanaFS(client *hana.Client) *HanaFS {
	return &HanaFS{client: client}
}
