package internal

import (
	"net/http"
	"time"
)

type App struct {
	Id             string           `json:"id"`             // Unique ID for this app instance
	LastInvocation time.Time        `json:"lastInvocation"` // Time of the last invocation
	FrontendUrl    string           `json:"frontendUrl"`    // User-facing or reverse proxy-facing url
	Runner         AppServiceRunner `json:"runner"`         // Interface to the service itself, since the app could be on a number of runtimes
}

func (app *App) Init() error {
	if err := app.Runner.Create(); err != nil {
		return err
	}

	if err := app.Runner.InitCleanupJobs(&app.LastInvocation); err != nil {
		return err
	}

	return nil
}

// The AppServiceRunner interface describes functions necessary for a type
// to represent a running application
type AppServiceRunner interface {
	Create() error
	Cleanup() error
	InitCleanupJobs(*time.Time) error
	SetIsReady()
	IsReady() bool
	Invoke(w http.ResponseWriter, r *http.Request)
}
