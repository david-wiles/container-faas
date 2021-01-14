package internal

import (
	"net/http"
	"time"
)

type App struct {
	ID             string    `json:"id"`             // Unique ID for this app instance
	LastInvocation time.Time `json:"lastInvocation"` // Time of the last invocation
	ExternalURL    string    `json:"externalUrl"`

	// Reverse proxy-facing url, could be user-facing if no ingress
	frontendURL string

	// Interface to the service itself, since the app could be on a number of runtimes
	Runner AppServiceRunner `json:"runner"`
}

func (app *App) Init() error {
	if err := app.Runner.Create(); err != nil {
		return err
	}
	return nil
}

// The AppServiceRunner interface describes functions necessary for a type
// to represent a running application
type AppServiceRunner interface {
	Create() error
	Cleanup() error
	IsReady() bool
	BlockUntilReady()
	Invoke(w http.ResponseWriter, r *http.Request)
}
