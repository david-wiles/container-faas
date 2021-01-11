package internal

import (
	"time"
)

// A AppManager should be a thin service around some kind of store for
// container or function info. By default, this will be an in-memory map, but
// could be implemented with an external database
type AppManager interface {
	// Get the container or function with this id
	Get(string) (*App, bool)
	// Create the given container or function
	Create(App) (*App, bool)
	// Deletes the container or function with the id
	Delete(string)
}

type DefaultAppManager struct {
	apps map[string]*App
}

func (mgr *DefaultAppManager) Get(id string) (*App, bool) {
	if app, ok := mgr.apps[id]; ok {
		return app, true
	} else {
		return nil, false
	}
}

func (mgr *DefaultAppManager) Create(app App) (*App, bool) {
	if _, ok := mgr.apps[app.Id]; !ok {
		tmp := &App{
			Id:             app.Id,
			LastInvocation: time.Unix(0, 0),
			FrontendUrl:    "http://" + G.Addr + "/app/" + app.Id,
		}
		mgr.apps[app.Id] = tmp
		return tmp, true
	} else {
		return nil, false
	}
}

func (mgr *DefaultAppManager) Delete(id string) {
	if _, ok := mgr.apps[id]; ok {
		delete(mgr.apps, id)
	}
}
