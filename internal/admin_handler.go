// admin_handler.go
// Implementation of admin routes on the application server
// The logic in this file creates, deletes, and gets the status
// of apps at a high level.
//
// A more detailed of the usage of these routes can be found in the README
package internal

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type AdminHandler struct{}

// Splits actions based on the HTTP method
// Each method will use a different function since there is little shared
// functionality between the intended action of verbs
func (h AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	case "POST":
		h.post(w, r)
	case "DELETE":
		h.delete(w, r)
	default:
		ErrorResponse(w, "HTTP Method not supported", 400)
	}
}

// Get information about a specific app, based on the id passed in the route
// If the app does not exist, a 404 message is returned along with a readable message
func (AdminHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := trimPath("/admin/", r)
	if err != nil {
		BasicResponse(w, "Resource not found", 404)
		return
	}

	if app, ok := G.AppMgr.Get(id); ok {
		b, err := json.Marshal(app)
		if err != nil {
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	} else {
		G.Logger.Warning("App not found: " + id)
		BasicResponse(w, "App not found", 404)
	}
}

type containerPostRequest struct {
	Image string   `json:"image"`
	Cmd   string   `json:"cmd"`
	Dir   string   `json:"dir"`
	Env   []string `json:"env"`
}

// POSTing a message to this route will create a new app based on the parameters
// in the request body. The app will be created and started, and if the service
// uses an ingress server then it will be re-configured to serve the new app.
// If any errors occur during app creation, then the app will be cleaned up
// to prevent bad or inconsistent states
func (AdminHandler) post(w http.ResponseWriter, r *http.Request) {

	id, err := trimPath("/admin/", r)
	if err != nil {
		BasicResponse(w, "Resource not found", 404)
		return
	}

	if _, ok := G.AppMgr.Get(id); ok {
		BasicResponse(w, "Container already exists", 200)
		return
	}

	// Parse request body
	reqBody := &containerPostRequest{}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(reqBody)
	if err != nil {
		G.Logger.LogError(err)
		ErrorResponse(w, "Could not parse request body: "+err.Error(), 500)
		return
	}

	// Create the app in the app management service
	if app, ok := G.AppMgr.Create(&App{
		ID:             id,
		LastInvocation: time.Unix(0, 0),
		frontendURL:    "http://" + G.Addr + "/app/" + id,
		Runner: NewDockerContainer(
			id,                              // docker id
			reqBody.Image,                   // docker image
			id,                              // app id
			reqBody.Dir,                     // mounted dir
			strings.Split(reqBody.Cmd, " "), // start command
			reqBody.Env,                     // environment variables
		),
	}); ok {
		// Initialize, create, and start the app
		if err := app.Init(); err != nil {
			_ = app.Runner.Cleanup()
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}

		// Create ingress for the app
		u, err := initAppIngress(app)
		if err != nil {
			_ = G.Ingress.Remove(app)
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}

		G.AppMgr.Update(app.ID, func() *App {
			app.ExternalURL = u
			return app
		})

		G.Logger.Info("Successfully built container")

		b, err := json.Marshal(app)
		if err != nil {
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	} else {
		G.AppMgr.Delete(id)
		G.Logger.LogError(err)
		ErrorResponse(w, err.Error(), 500)
		return
	}
}

// Deletes any app specified and removes it from the service
func (AdminHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := trimPath("/admin/", r)
	if err != nil {
		ErrorResponse(w, "Resource not found", 404)
		return
	}

	if app, ok := G.AppMgr.Get(id); ok {
		// Remove the app's runner
		if err = app.Runner.Cleanup(); err != nil {
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}

		// Remove the app's ingress
		if err = removeAppIngress(app); err != nil {
			G.Logger.LogError(err)
			ErrorResponse(w, err.Error(), 500)
			return
		}

		G.AppMgr.Delete(id)

		w.WriteHeader(200)

	} else {
		G.Logger.Warning("Container not found")
		ErrorResponse(w, "Resource not found", 404)
		return
	}
}

// Write and reload app ingress. If NoIngress is used, this is a nop
func initAppIngress(app *App) (string, error) {
	u, err := G.Ingress.Write(app)
	if err != nil {
		return "", err
	}

	if err := G.Ingress.Reload(); err != nil {
		return "", err
	}

	return u, nil
}

// Removes and resets app ingress without the specified app
func removeAppIngress(app *App) error {
	if err := G.Ingress.Remove(app); err != nil {
		return err
	}

	if err := G.Ingress.Reload(); err != nil {
		return err
	}

	return nil
}
