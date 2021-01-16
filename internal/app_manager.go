package internal

import (
	"sync"
)

// AppManager should be a wrapper around some kind of store for
// container or function info. By default, this will be an in-memory map, but
// could be implemented with an external database
//
// Implementations of AppManager should be thread-safe since multiple goroutines
// could be reading or writing to the app at the same time.
// For this reason, updates to an app entry should be done in an Update block
// so that the appropriate app entry can be locked during that time
type AppManager interface {
	// Get the container or function with this id
	// The newly-created app entry should be returned
	Get(string) (*App, bool)

	// Create the given container or function
	// Return the modified app
	Create(*App) (*App, bool)

	// Updates the app entry if it exists
	// Since the app must be write-locked during any updates,
	// this should be done in a function to allow for wrapping the
	// action within a mutex lock (for in-memory stores)
	Update(string, func() *App) bool

	// Deletes the container or function with the id
	// No success bool is returned since the end-result should be
	// the same: no reference to the app should remain
	Delete(string)
}

// The default AppManager is a simple kv mapping the app id to the
// app entry. This struct also contains a mutex to ensure that all
// data will be consistent across every request
type DefaultAppManager struct {
	apps  map[string]*App
	appMu *sync.Mutex
}

func (mgr *DefaultAppManager) Get(id string) (*App, bool) {
	mgr.appMu.Lock()
	defer mgr.appMu.Unlock()
	if app, ok := mgr.apps[id]; ok {
		return app, true
	} else {
		return nil, false
	}
}

func (mgr *DefaultAppManager) Create(app *App) (*App, bool) {
	mgr.appMu.Lock()
	defer mgr.appMu.Unlock()
	if _, ok := mgr.apps[app.ID]; !ok {
		mgr.apps[app.ID] = app
		return app, true
	} else {
		return nil, false
	}
}

func (mgr *DefaultAppManager) Update(id string, updater func() *App) bool {
	if _, ok := mgr.apps[id]; ok {
		mgr.appMu.Lock()
		mgr.apps[id] = updater()
		mgr.appMu.Unlock()
	}
	return false
}

func (mgr *DefaultAppManager) Delete(id string) {
	mgr.appMu.Lock()
	delete(mgr.apps, id)
	mgr.appMu.Unlock()
}
