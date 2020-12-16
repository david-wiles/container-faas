package internal

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ContainerHandler struct{}

func (i ContainerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	if len(parts) < 3 {
		G.Logger.Warning("Invalid request: " + r.URL.Path)
		HTTPError(w, "Container not found", 404)
		return
	}

	id := parts[2]

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

	// Create the container if it was evicted
	if c.DockerID == "" {
		err = createContainer(c)
		if err != nil {
			_ = G.ContainerMgr.reset(id)
			G.Logger.LogError(err)
			HTTPError(w, err.Error(), 500)
			return
		}
	}

	// Start the container if it isn't already running
	if !c.IsRunning {
		err = startContainer(c)
		if err != nil {

			_ = removeContainer(c)
			c.IsRunning = false

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

	// Rewrite path and pass to proxy
	urlRewrite, err := url.Parse(strings.ReplaceAll(r.URL.String(), "/container/"+c.Id, ""))
	if err != nil {
		G.Logger.LogError(err)
		HTTPError(w, err.Error(), 500)
		return
	}

	proxyRequest := r.Clone(context.Background())
	proxyRequest.URL = urlRewrite
	c.proxy.ServeHTTP(w, proxyRequest)
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
