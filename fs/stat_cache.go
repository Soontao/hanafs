package fs

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
)

// StatCache type
//
// in memory stat cache
type StatCache struct {
	cache       *sync.Map
	provider    func(string) (*fuse.Stat_t, error)
	dirProvider func(string) (*Directory, error)
	cacheLock   sync.RWMutex
	notExist    []string
}

// CheckIfFileNotExist from cache
func (sc *StatCache) CheckIfFileNotExist(path string) (rt bool) {
	sc.cacheLock.RLock()
	defer sc.cacheLock.RUnlock()
	for _, f := range sc.notExist {
		if f == path {
			return true
		}
	}
	return false
}

// AddNotExistFileCache to cache
func (sc *StatCache) AddNotExistFileCache(path string) {

	if !sc.CheckIfFileNotExist(path) {
		sc.cacheLock.Lock()
		defer sc.cacheLock.Unlock()
		sc.notExist = append(sc.notExist, path)
	}

}

// FileIsExistNow to remove un-existed cache
func (sc *StatCache) FileIsExistNow(path string) {

	if sc.CheckIfFileNotExist(path) {
		sc.cacheLock.Lock()
		defer sc.cacheLock.Unlock()
		for i, notExistPath := range sc.notExist {
			if notExistPath == path {
				sc.notExist = append(sc.notExist[:i], sc.notExist[i+1:]...)
				break
			}
		}
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

	v, err := sc.GetStatDirect(path)

	if err != nil {
		return nil, err
	}

	sc.PreCacheStat(path, v)
	return v, nil
}

// GetStatDirect directly, without cache and pre-cache
func (sc *StatCache) GetStatDirect(path string) (*fuse.Stat_t, error) {

	v, err := sc.provider(path)

	if err != nil {
		return nil, err
	}

	return v, nil
}

func (sc *StatCache) RefreshCache() {

	wg := sync.WaitGroup{}

	// each directory will update in parallel
	sc.cache.Range(func(key interface{}, value interface{}) bool {

		name := key.(string)

		stat := value.(*fuse.Stat_t)

		if (stat.Mode & fuse.S_IFMT) == fuse.S_IFDIR {

			go func(n string) {

				wg.Add(1)
				defer wg.Done()
				v, err := sc.GetDirDirect(n)

				if err != nil {
					if err == hana.ErrFileNotFound {
						sc.RemoveStatCache(n)
						sc.AddNotExistFileCache(n)
					}
					// log error
					return
				}

				// update file stat caches
				sc.PreCacheDirectory(n, v)

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

	if len(path) == 0 {
		path = "/"
	}

	v.children.Range(func(key interface{}, value interface{}) bool {

		nodeName := key.(string)
		st := value.(*fuse.Stat_t)

		nodePath := filepath.Join(path, nodeName)

		sc.FileIsExistNow(nodePath)
		sc.PreCacheStat(nodePath, st)
		unixTerminalCheckFile := "._" + nodeName
		unixTerminalCheckFilePath := filepath.Join(path, unixTerminalCheckFile)

		if _, exist := v.children.Load(unixTerminalCheckFile); !exist {
			sc.AddNotExistFileCache(unixTerminalCheckFilePath)
		}

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

// RefreshStat value
func (sc *StatCache) RefreshStat(path string) {
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
