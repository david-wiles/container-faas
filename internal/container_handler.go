package internal

import (
	"net/http"
	"time"
)

type ContainerHandler struct{}

func (i ContainerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := G.ContainerMgr.get(r.URL.Path)
	if err != nil {
		if ContainerNotFound(err) {
			HTTPError(w, err.Error(), 404)
		} else {
			HTTPError(w, err.Error(), 500)
		}
		return
	}

	// Create the container if it was evicted
	if c.DockerID == "" {
		err = createContainer(c)
		if err != nil {
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	// Start the container if it isn't already running
	if !c.IsRunning {
		err = startContainer(c)
		if err != nil {
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	c.LastInvocation = time.Now()

	c.proxy.ServeHTTP(w, r)
}
