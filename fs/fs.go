package fs

import (
	"path/filepath"
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
	f.statCache.UIHaveOpenResource(path)
	return 0
}

func (f *HanaFS) Open(path string, flags int) (errc int, fh uint64) {
	f.statCache.UIHaveOpenResource(path)
	return 0, 0
}

func (f *HanaFS) Opendir(path string) (int, uint64) {
	f.statCache.UIHaveOpenResource(path)
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

	data := make([]byte, len(buff))

	copy(data, buff)

	stat := &fuse.Stat_t{}
	err := f.Getattr(path, stat, fh)

	if err != 0 {
		return -fuse.ENOENT
	}

	if ofst != 0 {
		content, e := f.client.ReadFile(path)
		if e != nil {
			// log error here
			return -fuse.EFAULT
		}
		data = append(content[0:ofst], data...)
	}

	if e := f.client.WriteFileContent(path, data); e != nil {
		// log error here
		return -fuse.EFAULT
	}

	f.statCache.RefreshStat(path)

	// return length of write data
	return len(buff)
}

func (f *HanaFS) Truncate(path string, size int64, fh uint64) (errc int) {
	// mac os/linux change the file size
	stat, err := f.statCache.GetStat(path)
	if err != nil {
		return -fuse.EIO
	}
	stat.Size = size
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

	for _, w := range dir {
		sPath := w.Path
		oStat := w.Stat
		_, sName := filepath.Split(sPath)
		if len(sPath) > 0 {
			if oStat.Uid == 0 {
				uid, gid, _ := fuse.Getcontext()
				oStat.Uid = uid
				oStat.Gid = gid
			}
			fill(sName, oStat, 0)
		}

	}

	return 0
}

func (f *HanaFS) Rename(oldpath string, newpath string) (errc int) {
	stat, err := f.statCache.GetStat(oldpath)

	if err != nil {
		return -fuse.ENOENT
	}

	err = f.client.Rename(oldpath, newpath, isDir(stat.Mode))

	if err != nil {
		// log error here
		return -fuse.ENOENT
	}

	f.statCache.AddNotExistFileCache(oldpath)
	f.statCache.FileIsExistNow(newpath)

	return 0
}

// Getattr for file/dir
func (f *HanaFS) Getattr(path string, s *fuse.Stat_t, fh uint64) int {

	stat, err := f.statCache.GetStat(path)

	if err != nil {
		return -fuse.ENOENT
	}

	// sometimes, system can not provide correct uid & gid
	// so assign current user later (here)
	if stat.Uid == 0 {
		uid, gid, _ := fuse.Getcontext()
		stat.Uid = uid
		stat.Gid = gid
	}

	*s = *stat

	return 0

}

// Setxattr for OSX
func (f *HanaFS) Setxattr(path string, name string, value []byte, flags int) (errc int) {
	return 0
}

// Getxattr for OSX
func (f *HanaFS) Getxattr(path string, name string) (errc int, xatr []byte) {
	// mac os x attr
	return -fuse.ENOATTR, nil
}

// Read content from path
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

	fs := &HanaFS{client: client, statCache: NewStatCache(client)}

	cronDuration := gron.Every(DefaultRemoteCacheSeconds * time.Second)

	cron.AddFunc(cronDuration, fs.statCache.RefreshCache)

	cron.Start()

	return fs

}
