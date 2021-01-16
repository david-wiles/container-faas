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
		ErrorResponse(w, "Container not found", 404)
		return
	}

	id := parts[2]

	app, ok := G.AppMgr.Get(id)
	if !ok {
		G.Logger.Warning("App not found")
		ErrorResponse(w, "App not found", 404)
		return
	}

	// Create the container if it was evicted
	if !app.Runner.IsReady() {
		if err := app.Runner.Create(); err != nil {
			_ = app.Runner.Cleanup()
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}
	}

	// Hopefully the client will quit waiting if there is a problem
	// This state probably won't happen since we create the app just before this line
	// and return an error message to the client if something goes wrong
	app.Runner.BlockUntilReady()

	trimLen := len("/app/" + app.ID)
	if len(r.URL.Path) < trimLen {
		G.Logger.Error("Invalid URL")
		ErrorResponse(w, "Invalid URL", 404)
		return
	}

	// Rewrite path and pass to proxy
	u := r.URL.String()[trimLen:]
	urlRewrite, err := url.Parse(u)
	if err != nil {
		G.Logger.LogError(err)
		ErrorResponse(w, err.Error(), 500)
		return
	}
	proxyRequest := r.Clone(context.Background())
	proxyRequest.URL = urlRewrite

	G.AppMgr.Update(app.ID, func() *App {
		app.LastInvocation = time.Now()
		return app
	})
	app.Runner.Invoke(w, proxyRequest)
}
