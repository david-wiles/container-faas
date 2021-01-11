package internal

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AppHandler struct{}

func (AppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {
		G.Logger.Warning("Invalid request: " + r.URL.Path)
		HTTPError(w, "Container not found", 404)
		return
	}

	id := parts[2]

	app, ok := G.AppMgr.Get(id)
	if !ok {
		G.Logger.Warning("App not found")
		HTTPError(w, "App not found", 404)
		return
	}

	// Create the container if it was evicted
	if !app.Runner.IsReady() {
		if err := app.Runner.Create(); err != nil {
			_ = app.Runner.Cleanup()
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	if err := waitForReady(app); err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	// Rewrite path and pass to proxy
	urlRewrite, err := url.Parse(strings.ReplaceAll(r.URL.String(), "/app/"+app.Id, ""))
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}
	proxyRequest := r.Clone(context.Background())
	proxyRequest.URL = urlRewrite

	app.LastInvocation = time.Now()
	app.Runner.Invoke(w, proxyRequest)
}

type containerStartError struct {
	msg string
}

func (err *containerStartError) Error() string {
	if err != nil {
		return err.msg
	} else {
		return ""
	}
}

// Once the container is running, it will make a request to the server's /healthz/<container> endpoint to indicate its
// status. If this doesn't happen within a certain period of time, we should return an error
func waitForReady(app *App) error {

	start := time.Now()

	for !app.Runner.IsReady() {
		time.Sleep(time.Second)

		if time.Now().After(start.Add(G.StartTimeout)) {
			return &containerStartError{"Could not start the container"}
		}
	}

	return nil

}
