package fs

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
	"github.com/roylee0704/gron"
)

// DefaultRemoteCacheSeconds duration
const DefaultRemoteCacheSeconds = 15

// HanaFS type
type HanaFS struct {
	fuse.FileSystemBase
	client    *hana.Client
	statCache *StatCache
}

func (f *HanaFS) Release(path string, fh uint64) int {
	return 0
}

func (f *HanaFS) Open(path string, flags int) (errc int, fh uint64) {
	return 0, 0
}

func (f *HanaFS) Mkdir(path string, mode uint32) (errc int) {
	base, name := filepath.Split(path)

	if err := f.client.Create(base, name, true); err != nil {
		return -fuse.EIO
	}

	f.statCache.FileIsExistNow(path)

	f.statCache.RefreshStat(path)

	return 0
}

func (f *HanaFS) Fsync(path string, datasync bool, fh uint64) int {
	return 0
}

func (f *HanaFS) Unlink(path string) (errc int) {

	// remove file

	if err := f.client.Delete(path); err != nil {
		return -fuse.EIO
	}

	f.statCache.RemoveStatCache(path)

	return 0
}

func (f *HanaFS) Rmdir(path string) (errc int) {

	// remove directory

	if err := f.client.Delete(path); err != nil {
		return -fuse.EIO
	}

	f.statCache.RemoveStatCache(path)

	return 0
}

func (f *HanaFS) Create(path string, flags int, mode uint32) (int, uint64) {

	base, name := filepath.Split(path)

	if err := f.client.Create(base, name, false); err != nil {
		return -fuse.EIO, 0
	}

	f.statCache.FileIsExistNow(path)
	f.statCache.RefreshStat(path)

	return 0, 0

}

func (f *HanaFS) Utimens(path string, tmsp []fuse.Timespec) (errc int) {

	stat := &fuse.Stat_t{}
	err := f.Getattr(path, stat, 0)

	if err != 0 {
		return -fuse.ENOENT
	}

	if tmsp != nil {
		stat.Atim = tmsp[0]
		stat.Mtim = tmsp[1]
	}

	return 0
}

func (f *HanaFS) Write(path string, buff []byte, ofst int64, fh uint64) (n int) {

	stat := &fuse.Stat_t{}
	err := f.Getattr(path, stat, fh)

	if err != 0 {
		return -fuse.ENOENT
	}

	if ofst != 0 {
		// log error here
		return -fuse.EFAULT
	}

	if e := f.client.WriteFileContent(path, buff); e != nil {
		// log error here
		return -fuse.EFAULT
	}

	f.statCache.RefreshStat(path)

	// return length of write data
	return len(buff)
}

func (f *HanaFS) Truncate(path string, size int64, fh uint64) (errc int) {
	// mac os/linux change the file size

	return 0
}

func (f *HanaFS) Mknod(path string, mode uint32, dev uint64) (errc int) {

	base, name := filepath.Split(path)

	if err := f.client.Create(base, name, false); err != nil {
		return -fuse.EIO
	}

	f.statCache.FileIsExistNow(path)

	f.statCache.RefreshStat(path)

	return 0

}

func (f *HanaFS) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) int {

	dir, err := f.statCache.GetDir(path)

	if err != nil {
		return -fuse.ENOENT
	}

	dir.children.Range(func(key interface{}, value interface{}) bool {
		name := key.(string)
		st := value.(*fuse.Stat_t)

		if len(name) > 0 {
			if st.Uid == 0 {
				uid, gid, _ := fuse.Getcontext()
				st.Uid = uid
				st.Gid = gid
			}
			fill(name, st, 0)
		}

		return true
	})

	return 0
}

func (f *HanaFS) Getattr(path string, s *fuse.Stat_t, fh uint64) int {

	if f.statCache.CheckIfFileNotExist(path) {
		return -fuse.ENOENT
	}

	stat, err := f.statCache.GetStat(path)

	if err != nil {
		f.statCache.AddNotExistFileCache(path)
		return -fuse.ENOENT
	}

	if stat.Uid == 0 {
		uid, gid, _ := fuse.Getcontext()
		stat.Uid = uid
		stat.Gid = gid
	}

	*s = *stat

	return 0

}

func (f *HanaFS) Setxattr(path string, name string, value []byte, flags int) (errc int) {
	return 0
}

func (f *HanaFS) Getxattr(path string, name string) (errc int, xatr []byte) {
	// mac os x attr
	return -fuse.ENOATTR, nil
}

func (f *HanaFS) Read(path string, buff []byte, ofst int64, fh uint64) (n int) {
	contents, err := f.client.ReadFile(path)

	if err != nil {
		return -fuse.ENOENT
	}

	endofst := ofst + int64(len(buff))

	if endofst > int64(len(contents)) {
		endofst = int64(len(contents))
	}

	if endofst < ofst {
		return 0
	}

	n = copy(buff, contents[ofst:endofst])

	return
}

func (f *HanaFS) getDir(path string) (*Directory, error) {

	path = normalizePath(path)

	rt := &Directory{children: &sync.Map{}}
	dir, err := f.client.ReadDirectory(path)

	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}

	for _, hanaChild := range dir.Children {

		nodeName := hanaChild.Name

		nodePath := filepath.Join(path, nodeName)
		wg.Add(1)

		go func(s, p string) {
			defer wg.Done()
			st, err := f.getStat(p)

			if err != nil {
				return
			}
			rt.children.Store(s, st)
		}(nodeName, nodePath)

	}

	// wait parallel requests finished
	wg.Wait()

	return rt, nil
}

func (f *HanaFS) getStat(path string) (*fuse.Stat_t, error) {

	path = normalizePath(path)

	s := &fuse.Stat_t{}
	hanaStat, err := f.client.Stat(path)

	if err != nil {
		return nil, err
	}

	now := fuse.Now()
	uid, gid, _ := fuse.Getcontext()

	s.Nlink = 1
	s.Gid = gid
	s.Uid = uid
	s.Atim = now
	s.Mtim = *ToFuseTimeStamp(hanaStat.TimeStamp)

	if hanaStat.Directory {
		s.Mode = fuse.S_IFDIR | 0777
	} else {
		s.Mode = fuse.S_IFREG | 0777 // Regular File.
		if s.Size == 0 {
			if content, err := f.client.ReadFile(path); err == nil {
				s.Size = int64(len(content))
			}
		}
	}

	return s, nil

}

func (f *HanaFS) Chflags(path string, flags uint32) (errc int) {
	return 0
}

func (f *HanaFS) Setcrtime(path string, tmsp fuse.Timespec) int {
	return 0
}

func (f *HanaFS) Setchgtime(path string, tmsp fuse.Timespec) int {
	return 0
}

var _ fuse.FileSystemChflags = (*HanaFS)(nil)
var _ fuse.FileSystemSetcrtime = (*HanaFS)(nil)
var _ fuse.FileSystemSetchgtime = (*HanaFS)(nil)

// NewHanaFS type, initialize logic
func NewHanaFS(client *hana.Client) *HanaFS {

	cron := gron.New()

	fs := &HanaFS{client: client}

	fs.statCache = &StatCache{
		cache:       &sync.Map{},
		provider:    fs.getStat,
		dirProvider: fs.getDir,
	}

	// Retrieve root dir directly at startup
	fs.statCache.GetDir("/")
	// refresh first and sub directory at startup
	fs.statCache.RefreshCache()

	cron.AddFunc(gron.Every(DefaultRemoteCacheSeconds*time.Second), fs.statCache.RefreshCache)

	cron.Start()

	return fs
}
