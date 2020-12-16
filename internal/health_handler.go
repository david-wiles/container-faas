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

	// remove /health/ from the path and get container name
	names, err := getContainerName(id)
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	var c *containerInstance = nil

	for _, name := range names {
		c, err = G.ContainerMgr.get(name[1:]) // For some reason, Docker starts names with a '/' ??
		if c != nil {
			break
		}
	}

	if err != nil {
		if ContainerNotFound(err) {
			G.Logger.Warning(err.Error())
			HTTPError(w, err.Error(), 404)
			return
		} else {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	c.IsRunning = true

	w.WriteHeader(200)
}
