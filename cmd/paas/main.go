package main

import (
	"container-paas/internal"
	"net/http"
)

func main() {

	var err error
	internal.G, err = internal.FromEnv()
	if err != nil {
		panic(err)
	}

	mux := &internal.RegexMux{
		NotFound: internal.G.Logger.LogRequests(&internal.NotFoundHandler{}),
	}

	// Set up http handlers
	mux.Handle("/admin/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.AdminHandler{}))
	mux.Handle("/app/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.AppHandler{}))
	mux.Handle("/health/[a-zA-Z0-9_-]+", internal.G.Logger.LogRequests(&internal.HealthHandler{}))

	internal.G.Jobs.Start()

	// Create accepting connections
	internal.G.Logger.Info("Listening for requests on " + internal.G.Addr)
	err = http.ListenAndServe(internal.G.Addr, mux)
	if err != nil {
		panic(err)
	}

}
