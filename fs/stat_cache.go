package fs

import (
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Jeffail/tunny"
	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
)

// StatCache type
//
// in memory stat cache
type StatCache struct {
	cache            *ConcurrentMap
	openResource     *ConcurrentMap
	statProvider     StatProvider
	dirProvider      DirectoryProvider
	fileSizeProvider FileSizeProvider
	cacheLock        sync.RWMutex
	refreshLock      sync.Mutex
}

// UIHaveOpenResource means the resource should be refresh
func (sc *StatCache) UIHaveOpenResource(path string) {

	_, opened := sc.openResource.Load(path)

	if !opened {
		log.Printf("open resource: %v", path)
		sc.openResource.Store(path, true)
		sc.RefreshStat(path)

		// if opened resource is dir, preload sub dir of the dir
		if stat, err := sc.GetStat(path); err == nil && isDir(stat.Mode) {

			sc.RefreshDir(path)

			if d, e := sc.GetDir(path); e == nil && d != nil {
				d.children.Range(func(key interface{}, value interface{}) bool {
					if value != nil {
						stat := value.(*fuse.Stat_t)
						if isDir(stat.Mode) {
							subDirPath := filepath.Join(path, key.(string))
							sc.RefreshDir(subDirPath)
						}
					}
					return true
				})

			}

		}
	}

}

// IsOpenedDirectoryFile data
func (sc *StatCache) IsOpenedDirectoryFile(path string) (rt bool) {

	path = normalizePath(path)

	dir, _ := filepath.Split(path)

	if len(dir) > 1 {
		dir = strings.TrimRight(dir, "/")
	}

	_, rt = sc.openResource.Load(dir)

	return

}

func (sc *StatCache) cacheRangeAll(f func(path string, stat *fuse.Stat_t)) {
	sc.cache.Range(func(key interface{}, value interface{}) bool {
		p := key.(string)
		s := value.(*fuse.Stat_t)
		f(p, s)
		return true
	})
}

// CheckIfFileNotExist from cache, return true if not exist
func (sc *StatCache) CheckIfFileNotExist(path string) (rt bool) {
	_, exist := sc.cache.Load(path)
	return !exist
}

// AddNotExistFileCache to cache
func (sc *StatCache) AddNotExistFileCache(path string) {
	// remove from cache
	sc.cache.Delete(path)
}

// FileIsExistNow to remove un-existed cache
func (sc *StatCache) FileIsExistNow(path string) {
	if !sc.CheckIfFileNotExist(path) {
		sc.RefreshStat(path)
	}
}

// FilesIsExistNow to remove un-existed cache
func (sc *StatCache) FilesIsExistNow(pathes []string) {

	for _, path := range pathes {
		sc.FileIsExistNow(path)
	}

}

// GetOrCacheStat func, if not exist, will retrive and cache it
func (sc *StatCache) GetOrCacheStat(path string) (*fuse.Stat_t, error) {
	v, err := sc.GetStat(path)
	if err != nil {
		return nil, err
	}
	sc.PreCacheStat(path, v)
	return v, nil
}

// GetDirStats func, by prefix
func (sc *StatCache) GetDirStats(path string) (rt *Directory) {
	path = normalizePath(path)

	rt = &Directory{
		children: &sync.Map{},
	}

	sc.cache.Range(func(key interface{}, value interface{}) bool {

		base, name := filepath.Split(key.(string))

		if len(base) > 1 {
			base = strings.TrimRight(base, "/")
		}

		if base == path {
			rt.children.Store(name, value)
		}

		return true
	})

	return rt
}

// GetStat directly, if not exist, will retrive
func (sc *StatCache) GetStat(path string) (*fuse.Stat_t, error) {

	if v, exist := sc.cache.Load(path); exist {
		return v.(*fuse.Stat_t), nil
	}

	// if have pre load, but can not found in cache
	// it means not exist
	if sc.IsOpenedDirectoryFile(path) {
		return nil, hana.ErrFileNotFound
	}

	v, err := sc.GetStatDirect(path)

	if err != nil {
		return nil, err
	}

	sc.PreCacheStat(path, v)
	return v, nil
}

// GetStatDirect directly, without cache and pre-cache
func (sc *StatCache) GetStatDirect(path string) (*fuse.Stat_t, error) {

	v, err := sc.statProvider(path)

	if err != nil {
		return nil, err
	}

	if !isDir(v.Mode) {

		if c, exist := sc.cache.Load(path); exist {

			currentStat := c.(*fuse.Stat_t)

			// if changed, retrive the new size
			if sc.IsOpenedDirectoryFile(path) && currentStat.Mtim.Sec != v.Mtim.Sec {
				v.Size = sc.fileSizeProvider(path)
			} else {
				v.Size = currentStat.Size
			}

		} else {
			v.Size = sc.fileSizeProvider(path)
		}

	}

	if err != nil {
		return nil, err
	}

	return v, nil
}

// RefreshCache stats
func (sc *StatCache) RefreshCache() {
	// ensure only one goroutine run refresh job
	sc.refreshLock.Lock()
	defer sc.refreshLock.Unlock()

	wg := sync.WaitGroup{}

	// limit request concurrent
	pool := tunny.NewFunc(30, func(in interface{}) interface{} {

		n := in.(string)
		dir, err := sc.GetDirDirect(n)

		if err != nil {
			if err == hana.ErrFileNotFound {
				sc.RemoveStatCache(n)
				sc.AddNotExistFileCache(n)
			}
			log.Println(err)
			// log error
			return nil
		}

		return dir

	})

	// each directory will update in parallel
	sc.cache.Range(func(key interface{}, value interface{}) bool {

		name := key.(string)

		stat := value.(*fuse.Stat_t)

		// only refresh directory when user opened its parent
		if isDir(stat.Mode) && sc.IsOpenedDirectoryFile(name) {

			wg.Add(1)

			go func(n string) {

				defer wg.Done()

				log.Printf("refresh data for: %v", n)

				pValue := pool.Process(n)

				if pValue != nil {
					// update file stat caches
					sc.PreCacheDirectory(n, pValue.(*Directory))
				}

			}(name)
		}

		return true
	})

	// wait all goroutines finished
	wg.Wait()

	return
}

// PreCacheDirectory value, will not remove
func (sc *StatCache) PreCacheDirectory(path string, v *Directory) {

	path = normalizePath(path)

	v.children.Range(func(key interface{}, value interface{}) bool {

		nodeName := key.(string)
		st := value.(*fuse.Stat_t)

		nodePath := filepath.Join(path, nodeName)

		sc.FileIsExistNow(nodePath)
		sc.PreCacheStat(nodePath, st)

		return true
	})

}

// GetDir directly, if not exist, will retrive and cache it
func (sc *StatCache) GetDir(path string) (*Directory, error) {

	if _, exist := sc.cache.Load(path); exist {
		return sc.GetDirStats(path), nil
	}

	v, err := sc.GetDirDirect(path)

	if err != nil {
		return nil, err
	}

	sc.PreCacheDirectory(path, v)

	return v, nil
}

// GetDirDirect func, without cache & pre cache
func (sc *StatCache) GetDirDirect(path string) (*Directory, error) {

	v, err := sc.dirProvider(path)

	if err != nil {
		return nil, err
	}

	return v, nil

}

// RefreshDir and item stats
func (sc *StatCache) RefreshDir(path string) {
	if dir, err := sc.GetDirDirect(path); err != nil {
		sc.PreCacheDirectory(path, dir)
	}
}

// RefreshStat value
func (sc *StatCache) RefreshStat(path string) {
	log.Printf("refresh stat: %v", path)
	if v, err := sc.GetStatDirect(path); err != nil {
		sc.PreCacheStat(path, v)
	}
}

// PreCacheStat value
func (sc *StatCache) PreCacheStat(path string, v *fuse.Stat_t) {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.ReplaceAll(path, "\\/", "/")

	sc.setCache(path, v)
}

func (sc *StatCache) setCache(path string, v *fuse.Stat_t) {
	sc.cache.Store(path, v)
}

// RemoveStatCache value
func (sc *StatCache) RemoveStatCache(path string) {

	sc.cache.Delete(path)

}
