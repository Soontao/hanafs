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

// GetStat directly, if not exist, will retrive but not cache
func (sc *StatCache) GetStat(path string) (*fuse.Stat_t, error) {
	if v, exist := sc.cache[path]; exist {
		return v, nil
	}
	v, err := sc.provider(path)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// GetCacheAndRemoveCachedStat func, if not exist, will retrive directly, and will not cache it
func (sc *StatCache) GetCacheAndRemoveCachedStat(path string) (*fuse.Stat_t, error) {
	if v, exist := sc.cache[path]; exist {
		defer sc.RemoveStatCache(path)
		return v, nil
	}
	v, err := sc.provider(path)
	if err != nil {
		return nil, err
	}
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
	sc.cacheLock.Lock()
	defer sc.cacheLock.Unlock()
	if _, exist := sc.cache[path]; exist {
		delete(sc.cache, path)
	}
}
