package main

import (
	"flag"
	"fmt"
	"github.com/robfig/cron/v3"
	"net/http"
	"time"
	"vlab-faas-server/internal"
)

func main() {

	addrPtr := flag.String("addr", "", "Address used to listen for connections")
	stopTimeoutPtr := flag.String("stop-timeout", "", "Amount of time to wait for a container to stop")
	containerStartTimeoutPtr := flag.String("start-timeout", "", "Amount of time to wait for a container to start")
	dockerNetworkPtr := flag.String("network", "", "Name of the docker network the app containers are placed in")
	useNginx := flag.Bool("nginx", false, "Indicates whether the program will run behind an nginx proxy")
	logLevel := flag.Int("log", 0, "Log level. 0 indicates all logs, 4 indicates none")

	flag.Parse()

	var err error
	internal.G, err = internal.ParseArgs(
		*addrPtr,
		*stopTimeoutPtr,
		*containerStartTimeoutPtr,
		*dockerNetworkPtr,
		*useNginx,
		*logLevel,
	)
	if err != nil {
		panic(err)
	}

	mux := &internal.RegexMux{
		NotFound: internal.G.Logger.LogRequests(&internal.NotFoundHandler{}),
	}
	jobs := cron.New()

	// Set up http handlers
	mux.Handle("/admin/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.AdminHandler{}))
	mux.Handle("/container/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.ContainerHandler{}))
	mux.Handle("/health/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.HealthHandler{}))

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
			internal.G.Logger.Info("Successfully stopped inactive containers.")
		}
	})
	if err != nil {
		panic(err)
	}

	jobs.Start()

	// Start accepting connections
	_, _ = fmt.Println("Listening for requests on " + internal.G.Addr)

	err = http.ListenAndServe(internal.G.Addr, mux)
	if err != nil {
		panic(err)
	}

}
