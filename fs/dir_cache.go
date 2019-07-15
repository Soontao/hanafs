package fs

import (
	"path/filepath"
	"sync"

	"github.com/Soontao/hanafs/hana"
	"github.com/billziss-gh/cgofuse/fuse"
)

// Directory type
type Directory struct {
	children map[string]*fuse.Stat_t
}

// DirectoryCache type
type DirectoryCache struct {
	cache     map[string]*Directory
	provider  func(string) (*Directory, error)
	statCache *StatCache
	cacheLock sync.RWMutex
}

// RefreshCache values
func (sc *DirectoryCache) RefreshCache() {

	wg := sync.WaitGroup{}

	// each directory will update in parallel

	for name := range sc.cache {
		go func(n string) {
			wg.Add(1)
			defer wg.Done()
			v, err := sc.GetDirDirect(n)
			if err != nil {
				if err == hana.ErrFileNotFound {
					sc.RemoveDirectoryCache(n)
				}
				// log error

				return
			}

			// update file stat caches
			sc.PreCacheDirectory(n, v)

		}(name)
	}

	// wait all goroutines finished
	wg.Wait()

	return
}

// GetDir directly, if not exist, will retrive and cache it
func (sc *DirectoryCache) GetDir(path string) (*Directory, error) {

	if _, exist := sc.cache[path]; exist {
		return sc.statCache.GetDirStats(path), nil
	}

	v, err := sc.GetDirDirect(path)

	if err != nil {
		return nil, err
	}

	sc.PreCacheDirectory(path, v)

	return v, nil
}

// GetDirDirect func, without cache & pre cache
func (sc *DirectoryCache) GetDirDirect(path string) (*Directory, error) {

	v, err := sc.provider(path)

	if err != nil {
		return nil, err
	}

	return v, nil

}

// PreCacheDirectory value, will not remove
func (sc *DirectoryCache) PreCacheDirectory(path string, v *Directory) {
	if len(path) == 0 {
		path = "/"
	}

	sc.setCache(path, v)

	for nodeName, nodeStat := range v.children {

		nodePath := filepath.Join(path, nodeName)

		sc.statCache.FileIsExistNow(nodePath)
		sc.statCache.PreCacheStat(nodePath, nodeStat)

		unixTerminalCheckFile := filepath.Join(path, "._"+nodeName)

		if _, exist := v.children[unixTerminalCheckFile]; !exist {
			sc.statCache.AddNotExistFileCache(unixTerminalCheckFile)
		}

	}

}

func (sc *DirectoryCache) setCache(path string, v *Directory) {
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
