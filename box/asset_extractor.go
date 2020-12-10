package box

import (
	"io/ioutil"
	"os"
	"sync"
)

var once sync.Once
var singleton AssetExtractor

type AssetExtractor interface {
	Get(path string) string
	Close()
}

type asset struct {
	mu     sync.Mutex
	closed bool
	paths  map[string]string
}

func GetAssetExtractor() AssetExtractor {
	once.Do(func() {
		singleton = &asset{
			paths: make(map[string]string),
		}
	})
	return singleton
}

func (a *asset) Get(path string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	osPath, ok := a.paths[path]
	if ok {
		return osPath
	}
	binary := Get(path)
	if binary == nil {
		return ""
	}
	osFile, err := ioutil.TempFile(os.TempDir(), "G14Manager-")
	if err != nil {
		return ""
	}
	if _, err = osFile.Write(binary); err != nil {
		return ""
	}
	if err := osFile.Close(); err != nil {
		return ""
	}
	a.paths[path] = osFile.Name()
	return a.paths[path]
}

func (a *asset) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.closed {
		return
	}

	for _, p := range a.paths {
		os.Remove(p)
	}
	a.closed = true
}
