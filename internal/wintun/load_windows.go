//go:build windows

package wintun

import (
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
)

var (
	loadOnce sync.Once
	loadErr  error
	module   windows.Handle
)

// Load preloads Wintun by absolute path. The upstream wintun package later
// resolves the already loaded module by its canonical wintun.dll name.
func Load(path string) error {
	loadOnce.Do(func() {
		module, loadErr = windows.LoadLibrary(path)
		if loadErr != nil {
			loadErr = fmt.Errorf("cannot load Wintun DLL %s: %w", path, loadErr)
		}
	})
	return loadErr
}
