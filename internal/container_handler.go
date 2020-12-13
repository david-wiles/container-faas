package internal

import (
	"net/http"
	"strings"
	"time"
)

type ContainerHandler struct{}

func (i ContainerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimLeft(r.URL.Path, "/container/")

	c, err := G.ContainerMgr.get(id)
	if err != nil {
		if ContainerNotFound(err) {
			G.Logger.Warning(err.Error())
			HTTPError(w, err.Error(), 404)
		} else {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
		}
		return
	}

	// Create the container if it was evicted
	if c.DockerID == "" {
		err = createContainer(c)
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	// Start the container if it isn't already running
	if !c.IsRunning {
		err = startContainer(c)
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}

		err = waitForRun(c)
		if err != nil {
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	c.LastInvocation = time.Now()

	c.proxy.ServeHTTP(w, r)
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

// Once the container is running, it will make a request to the server's /status/<container> endpoint to indicate its
// status. If this doesn't happen within a certain period of time, we should return an error
func waitForRun(c *containerInstance) error {

	start := time.Now()

	for !c.IsRunning {
		time.Sleep(time.Second)

		if time.Now().After(start.Add(G.ContainerStartTimeout)) {
			return &containerStartError{"Could not start the container"}
		}
	}

	return nil

}
