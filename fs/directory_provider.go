package fs

import (
	"strings"

	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
	"github.com/tidwall/gjson"
)

func trimBasePath(fullpath, basepath string) string {
	rt := strings.TrimLeft(fullpath, basepath)
	if !strings.HasPrefix(rt, "/") {
		rt = "/" + rt
	}

	return rt
}

func deepSearchDirStat(children []hana.Child, basePath string) (rt []*FileSystemStatWrapper) {

	basePath = strings.TrimRight(basePath, "/")

	for _, c := range children {

		now := fuse.Now()
		uid, gid, _ := fuse.Getcontext()
		path := ""

		s := &FileSystemStat{
			Nlink: 1,
			Gid:   gid,
			Uid:   uid,
			Atim:  now,
			Size:  0,
		}

		if c.Directory {
			s.Mode = fuse.S_IFDIR | 0777
			path = trimBasePath(c.ContentLocation, basePath)
		} else {
			s.Mode = fuse.S_IFREG | 0777
			path = trimBasePath(c.RunLocation, basePath)
			// file
			if sBackPack, ok := c.SapBackPack.(string); ok {
				ts := gjson.Get(sBackPack, "ActivatedAt").Int()
				s.Mtim = *ToFuseTimeStamp(ts)
				// refresh size in another location
			}

		}

		rt = append(rt, NewFileSystemStatWrapper(path, s))

		if c.Directory {
			rt = append(rt, deepSearchDirStat(c.Children, basePath)...)
		}

	}

	return
}

// CreateDirectoryProvider func
func CreateDirectoryProvider(client *hana.Client) DirectoryProvider {
	return func(path string, depth int64) ([]*FileSystemStatWrapper, error) {

		path = normalizePath(path)

		dir, err := client.ReadDirectory(path, 3)

		if err != nil {
			return nil, err
		}

		return deepSearchDirStat(dir.Children, client.GetBaseDirectory()), nil
	}
}
