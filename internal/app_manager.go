package internal

import (
	"sync"
	"time"
)

// AppManager should be a thin service around some kind of store for
// container or function info. By default, this will be an in-memory map, but
// could be implemented with an external database
//
// Implementations of AppManager should be thread-safe since multiple goroutines
// could be reading or writing to the app at the same time.
// For this reason, updates to an app entry should be done in an Update block
// so that the appropriate app entry can be locked during that time
type AppManager interface {
	// Get the container or function with this id
	Get(string) (*App, bool)
	// Create the given container or function
	Create(App) (*App, bool)
	// Updates the app entry if it exists
	Update(string, func() *App) bool
	// Deletes the container or function with the id
	Delete(string)
}

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

func (mgr *DefaultAppManager) Create(app App) (*App, bool) {
	tmp := &App{
		ID:             app.ID,
		LastInvocation: time.Unix(0, 0),
		frontendURL:    "http://" + G.Addr + "/app/" + app.ID,
		Runner:         app.Runner,
	}

	mgr.appMu.Lock()
	defer mgr.appMu.Unlock()
	if _, ok := mgr.apps[app.ID]; !ok {
		mgr.apps[app.ID] = tmp
		return tmp, true
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
