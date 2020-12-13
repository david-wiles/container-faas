package internal

import (
	"fmt"
	"net/http"
)

type StatusHandler struct{}

func (h *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		HTTPError(w, "Unsupported method", 400)
		return
	}

	id := r.URL.Path

	c, err := G.ContainerMgr.get(id)
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
