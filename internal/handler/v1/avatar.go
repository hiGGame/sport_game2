package v1

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

var AvatarCacheInstance *AvatarCache

func InitAvatarCache(baseDir, baseURL string) {
	AvatarCacheInstance = NewAvatarCache(baseDir, baseURL)
	AvatarCacheInstance.Get()
}

type AvatarList struct {
	Expert []string `json:"expert"`
	AIDog  []string `json:"aiDog"`
}

type AvatarCache struct {
	mu      sync.RWMutex
	data    *AvatarList
	baseDir string
	baseURL string
}

func NewAvatarCache(baseDir, baseURL string) *AvatarCache {
	return &AvatarCache{baseDir: baseDir, baseURL: baseURL}
}

func (c *AvatarCache) Get() *AvatarList {
	c.mu.RLock()
	if c.data != nil {
		c.mu.RUnlock()
		return c.data
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data != nil {
		return c.data
	}

	c.data = &AvatarList{
		Expert: listFiles(filepath.Join(c.baseDir, "expert"), c.baseURL),
		AIDog:  listFiles(filepath.Join(c.baseDir, "aiDog"), c.baseURL),
	}
	return c.data
}

func (c *AvatarCache) Refresh() {
	c.mu.Lock()
	c.data = nil
	c.mu.Unlock()
	c.Get()
}

func listFiles(dir, urlPrefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		files = append(files, urlPrefix+"/"+filepath.Base(dir)+"/"+name)
	}
	sort.Strings(files)
	return files
}
