package fs

import (
	"sync"
	"time"

	"github.com/billziss-gh/cgofuse/fuse"
)

type CachedDirectory struct {
	children map[string]*fuse.Stat_t
}

type DirectoryCache struct {
	cache     map[string]*CachedDirectory
	provider  func(string) (*CachedDirectory, error)
	cacheLock sync.RWMutex
}

// GetDir directly, if not exist, will retrive but not cache
func (sc *DirectoryCache) GetDir(path string) (*CachedDirectory, error) {
	if v, exist := sc.cache[path]; exist {
		return v, nil
	}
	v, err := sc.provider(path)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// PreDirectoryCacheSeconds value, remove value after seconds
func (sc *DirectoryCache) PreDirectoryCacheSeconds(path string, v *CachedDirectory, second int) {

	sc.setCache(path, v)

	go func() {
		time.Sleep(time.Second * time.Duration(second))
		// refresh value
		newV, _ := sc.provider(path)
		sc.PreDirectoryCacheSeconds(path, newV, second)
	}()

}
func (sc *DirectoryCache) setCache(path string, v *CachedDirectory) {
	sc.cacheLock.Lock()
	defer sc.cacheLock.Unlock()
	sc.cache[path] = v
}

// RemoveDirectoryCache value
func (sc *DirectoryCache) RemoveDirectoryCache(path string) {
	sc.cacheLock.Lock()
	defer sc.cacheLock.Unlock()
	if _, exist := sc.cache[path]; exist {
		delete(sc.cache, path)
	}
}
