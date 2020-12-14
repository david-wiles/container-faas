package internal

import (
	"fmt"
	"net/http"
	"strings"
)

type HealthHandler struct{}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		HTTPError(w, "Unsupported method", 400)
		return
	}

	names, err := getContainerName(strings.TrimLeft(r.URL.Path, "/healthz/"))
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	var c *containerInstance

	for _, name := range names {
		c, err = G.ContainerMgr.get(name[1:])
		if err == nil {
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

	_, _ = fmt.Fprintln(w, "Success")
}
