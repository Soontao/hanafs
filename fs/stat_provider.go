package fs

import (
	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
)

// CreateStatProvider func
func CreateStatProvider(client *hana.Client) StatProvider {
	return func(path string) (*fuse.Stat_t, error) {

		path = normalizePath(path)

		hanaStat, err := client.Stat(path)

		if err != nil {
			return nil, err
		}

		now := fuse.Now()

		uid, gid, _ := fuse.Getcontext()

		s := &fuse.Stat_t{
			Nlink: 1,
			Gid:   gid,
			Uid:   uid,
			Atim:  now,
			Mtim:  *ToFuseTimeStamp(hanaStat.TimeStamp),
			Size:  0,
		}

		if hanaStat.Directory {
			s.Mode = fuse.S_IFDIR | 0777
		} else {
			s.Mode = fuse.S_IFREG | 0777 // Regular File.
		}

		return s, nil

	}
}
