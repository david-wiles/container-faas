package internal

import (
	"net/http"
)

type HealthHandler struct{}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		HTTPError(w, "Unsupported method", 400)
		return
	}

	id, err := trimPath("/health/", r)
	if err != nil {
		HTTPError(w, "Resource not found", 404)
		return
	}

	app, ok := G.AppMgr.Get(id)
	if !ok {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	app.Runner.SetIsReady()

	w.WriteHeader(200)
}
