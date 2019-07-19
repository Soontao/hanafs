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
	maxDepthLock     sync.RWMutex
	refreshLock      sync.Mutex
	maxDepth         int64
}

func (sc *StatCache) setMaxDepth(depth int64) {
	sc.maxDepthLock.Lock()
	defer sc.maxDepthLock.Unlock()
	sc.maxDepth = depth
}

// GetMaxDepth for current
func (sc *StatCache) GetMaxDepth() int64 {
	sc.maxDepthLock.RLock()
	defer sc.maxDepthLock.RUnlock()
	return sc.maxDepth
}

// UIHaveOpenResource means the resource should be refresh
func (sc *StatCache) UIHaveOpenResource(path string) {

	_, opened := sc.openResource.Load(path)

	if !opened {

		sc.openResource.Store(path, true)
		sc.RefreshStat(path)

		// if opened resource is dir, preload sub dir of the dir
		if stat, err := sc.GetStat(path); err == nil && isDir(stat.Mode) {

			// refresh current dir stat
			sc.RefreshDir(path)

			if d, e := sc.GetDir(path); e == nil && d != nil {

				for _, w := range d {
					sPath := w.Path
					oStat := w.Stat
					if oStat != nil {
						if isDir(oStat.Mode) {
							sc.RefreshDir(sPath)
						} else {
							sc.RefreshStat(sPath)
						}
					}
				}

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
func (sc *StatCache) GetDirStats(path string) (rt []*FileSystemStatWrapper) {

	path = normalizePath(path)

	sc.cacheRangeAll(func(aPath string, oStat *fuse.Stat_t) {
		// ignore path stat itself
		if aPath == path {
			return
		}
		base, _ := filepath.Split(aPath)

		if len(base) > 1 {
			base = strings.TrimRight(base, "/")
		}

		if base == path {
			rt = append(rt, NewFileSystemStatWrapper(aPath, oStat))
		}

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
			if sc.IsOpenedDirectoryFile(path) && (currentStat.Mtim.Sec != v.Mtim.Sec || currentStat.Size == 0) {
				v.Size = sc.fileSizeProvider(path)
			} else {
				return currentStat, nil
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

				pValue := pool.Process(n)

				if pValue != nil {
					// update file stat caches
					sc.PreCacheDirectory(n, pValue.([]*FileSystemStatWrapper))
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
func (sc *StatCache) PreCacheDirectory(path string, v []*FileSystemStatWrapper) {

	path = normalizePath(path)

	for _, w := range v {
		aPath := w.Path
		oStat := w.Stat
		if c, exist := sc.cache.Load(aPath); exist {
			currentStat := c.(*FileSystemStat)
			if currentStat.Mtim.Sec != oStat.Mtim.Sec {
				// if updated, update file size info.
				oStat.Size = sc.fileSizeProvider(aPath)
			}

		}
		sc.PreCacheStat(aPath, oStat)

	}

}

// GetDir inner content directly, if not exist, will retrive and cache it
func (sc *StatCache) GetDir(path string) ([]*FileSystemStatWrapper, error) {

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
func (sc *StatCache) GetDirDirect(path string) ([]*FileSystemStatWrapper, error) {

	v, err := sc.dirProvider(path, sc.GetMaxDepth()+2)

	if err != nil {
		return nil, err
	}

	return v, nil

}

// RefreshDir and item stats
func (sc *StatCache) RefreshDir(path string) {

	dir, err := sc.GetDirDirect(path)

	if err == nil {
		sc.PreCacheDirectory(path, dir)
	} else {
		log.Println(err)
	}

}

// RefreshStat value
func (sc *StatCache) RefreshStat(path string) {
	if v, err := sc.GetStatDirect(path); err == nil {
		sc.PreCacheStat(path, v)
	} else {
		if err == hana.ErrFileNotFound {
			sc.RemoveStatCache(path)
		} else {
			log.Println(err)
		}
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

// NewStatCache constructor
func NewStatCache(client *hana.Client) *StatCache {
	return &StatCache{
		cache:            &ConcurrentMap{},
		statProvider:     CreateStatProvider(client),
		dirProvider:      CreateDirectoryProvider(client),
		fileSizeProvider: CreateFileSizeProvider(client),
		openResource:     &ConcurrentMap{},
		maxDepth:         3,
	}
}
