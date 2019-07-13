package fs

import (
	"sync"
	"time"

	"github.com/billziss-gh/cgofuse/fuse"
)

// StatCache type
//
// in memory stat cache
type StatCache struct {
	cache     map[string]*fuse.Stat_t
	provider  func(string) (*fuse.Stat_t, error)
	cacheLock sync.RWMutex
	notExist  []string
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
	sc.PreStatCache(path, v)
	return v, nil
}

// GetStat directly, if not exist, will retrive
func (sc *StatCache) GetStat(path string) (*fuse.Stat_t, error) {
	if v, exist := sc.cache[path]; exist {
		return v, nil
	}

	v, err := sc.provider(path)

	if err != nil {
		return nil, err
	}

	sc.PreStatCacheSeconds(path, v, DefaultRemoteCacheSeconds)
	return v, nil
}

// PreStatCache value
func (sc *StatCache) PreStatCache(path string, v *fuse.Stat_t) {
	sc.setCache(path, v)
}

// PreStatCacheSeconds value, remove value after seconds
func (sc *StatCache) PreStatCacheSeconds(path string, v *fuse.Stat_t, second int) {

	sc.setCache(path, v)

	go func() {
		time.Sleep(time.Second * time.Duration(second))
		// refresh value
		newV, _ := sc.provider(path)
		sc.PreStatCacheSeconds(path, newV, second)
	}()

}
func (sc *StatCache) setCache(path string, v *fuse.Stat_t) {
	sc.cacheLock.Lock()
	defer sc.cacheLock.Unlock()
	sc.cache[path] = v
}

// RemoveStatCache value
func (sc *StatCache) RemoveStatCache(path string) {
	sc.cacheLock.RLock()
	defer sc.cacheLock.RUnlock()
	if _, exist := sc.cache[path]; exist {
		sc.cacheLock.Lock()
		delete(sc.cache, path)
		sc.cacheLock.Unlock()
	}
}
