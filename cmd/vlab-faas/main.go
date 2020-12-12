package main

import (
	"fmt"
	"github.com/docker/docker/client"
	"github.com/robfig/cron/v3"
	"net/http"
	"os"
	"time"
	"vlab-faas-server/internal"
)

func main() {

	// TODO determine correct opts
	docker, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	internal.G = &internal.Global{
		ContainerMgr: &internal.ContainerManager{},
		Logger:       &internal.Logger{},
		Docker:       docker,
	}

	mux := &internal.RegexMux{
		NotFound: &internal.NotFoundHandler{},
	}
	jobs := cron.New()

	// Set up http handlers
	mux.Handle("/admin/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.AdminHandler{}))
	mux.Handle("/container/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.ContainerHandler{}))
	mux.Handle("/", internal.G.Logger.LogRequestFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			internal.HTTPError(w, "Unsupported method", 400)
			return
		}
		_, _ = fmt.Fprintf(w, "Got your request, but there isn't much to do yet.")
	}))

	// Setup cron jobs
	_, err = jobs.AddFunc("@every 1h", func() {
		// Remove containers that have been stopped for over 1 hour
		err := internal.G.ContainerMgr.EvictContainers(time.Hour)
		if err != nil {
			internal.G.Logger.LogError(err)
		} else {
			internal.G.Logger.Info("Successfully evicted old containers.")
		}
	})
	// Stop containers that have been inactive for over 15 minutes
	_, err = jobs.AddFunc("@every 5m", func() {
		err := internal.G.ContainerMgr.StopContainers(time.Minute * 15)
		if err != nil {
			internal.G.Logger.LogError(err)
		} else {
			internal.G.Logger.Info("Successfully evicted old containers.")
		}
	})
	if err != nil {
		panic(err)
	}

	// Start accepting connections
	jobs.Start()

	_, _ = fmt.Println("Listening for requests on port " + os.Getenv("PORT"))

	err = http.ListenAndServe(":"+os.Getenv("PORT"), mux)
	if err != nil {
		panic(err)
	}

}
